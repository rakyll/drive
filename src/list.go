// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package drive

import (
	"fmt"
	"path/filepath"
	"strings"

	spinner "github.com/odeke-em/cli-spinner"
	"github.com/odeke-em/log"
)

var BytesPerKB = float64(1024)

const RemoteDriveRootPath = "My Drive"

const (
	InTrash = 1 << iota
	Folder
	NonFolder
	Minimal
	Shared
	Owners
	CurrentVersion
)

type attribute struct {
	minimal bool
	mask    int
	parent  string
}

type byteDescription func(b int64) string

func memoizeBytes() byteDescription {
	cache := map[int64]string{}
	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	maxLen := len(suffixes) - 1

	return func(b int64) string {
		description, ok := cache[b]
		if ok {
			return description
		}

		bf := float64(b)
		i := 0
		description = ""
		for {
			if bf/BytesPerKB < 1 || i >= maxLen {
				description = fmt.Sprintf("%.2f%s", bf, suffixes[i])
				break
			}
			bf /= BytesPerKB
			i += 1
		}
		cache[b] = description
		return description
	}
}

var prettyBytes = memoizeBytes()

func (g *Commands) List() (err error) {
	root := g.context.AbsPathOf("")
	var relPath string

	resolver := g.rem.FindByPath
	if g.opts.InTrash {
		resolver = g.rem.FindByPathTrashed
	}

	var kvList []*keyValue

	for _, p := range g.opts.Sources {
		relP := g.context.AbsPathOf(p)
		relPath, err = filepath.Rel(root, relP)
		if err != nil {
			return
		}

		if relPath == "." {
			relPath = ""
		}
		relPath = "/" + relPath
		r, rErr := resolver(relPath)
		if rErr != nil {
			g.log.LogErrf("%v: '%s'\n", rErr, relPath)
			return
		}
		kvList = append(kvList, &keyValue{key: relPath, value: r})
	}

	spin := spinner.New(10)
	spin.Start()
	for _, kv := range kvList {
		if kv == nil || kv.value == nil {
			continue
		}
		if !g.breadthFirst(kv.value.(*File), "", kv.key, g.opts.Depth, g.opts.TypeMask, false) {
			break
		}
	}
	spin.Stop()

	// No-op for now for explicitly traversing shared content
	if false {
		// TODO: Allow traversal of shared content as well as designated paths
		// Next for shared
		sharedRemotes, sErr := g.rem.FindByPathShared("")
		if sErr == nil {
			opt := attribute{
				minimal: isMinimal(g.opts.TypeMask),
				parent:  "",
				mask:    g.opts.TypeMask,
			}
			for sFile := range sharedRemotes {
				sFile.pretty(g.log, opt)
			}
		}
	}

	return
}

func (f *File) pretty(logy *log.Logger, opt attribute) {
	fmtdPath := fmt.Sprintf("%s/%s", opt.parent, urlToPath(f.Name, false))

	if opt.minimal {
		logy.Logln(fmtdPath)
		if owners(opt.mask) && len(f.OwnerNames) >= 1 {
			logy.Logf(" %s ", strings.Join(f.OwnerNames, " & "))
		}
		return
	}

	if f.IsDir {
		logy.Logf("d")
	} else {
		logy.Logf("-")
	}
	if f.Shared {
		logy.Logf("s")
	} else {
		logy.Logf("-")
	}

	if f.UserPermission != nil {
		logy.Logf(" %-10s ", f.UserPermission.Role)
	}

	if owners(opt.mask) && len(f.OwnerNames) >= 1 {
		logy.Logf(" %s ", strings.Join(f.OwnerNames, " & "))
	}

	logy.Logf(" %-10s\t%-10s\t\t%-20s\t%-50s\n", prettyBytes(f.Size), f.Id, f.ModTime, fmtdPath)
}

func (g *Commands) breadthFirst(f *File, walkTrail, prefixPath string, depth int, mask int, inTrash bool) bool {
	headPath := ""
	if !rootLike(prefixPath) {
		headPath = prefixPath
	}

	opt := attribute{
		minimal: isMinimal(g.opts.TypeMask),
		mask:    mask,
		parent:  headPath,
	}

	if f.Name != RemoteDriveRootPath {
		if f.Name != "" && walkTrail != "" {
			headPath = headPath + "/" + f.Name
		}
	}
	if !f.IsDir {
		if walkTrail == "" {
			f.pretty(g.log, opt)
		}
		return true
	}

	// A depth of < 0 means traverse as deep as you can
	if depth == 0 {
		// At the end of the line, this was successful.
		return true
	}
	if depth > 0 {
		depth -= 1
	}

	expr := buildExpression(f.Id, mask, inTrash)

	req := g.rem.service.Files.List()
	req.Q(expr)
	req.MaxResults(g.opts.PageSize)

	fileChan := reqDoPage(req, g.opts.Hidden, !g.opts.NoPrompt)

	var children []*File
	onlyFiles := (g.opts.TypeMask & NonFolder) != 0

	opt.parent = headPath

	for file := range fileChan {
		if file == nil {
			return false
		}
		if isHidden(file.Name, g.opts.Hidden) {
			continue
		}

		children = append(children, file)

		// The case in which only directories wanted is covered by the buildExpression clause
		// reason being that only folder are allowed to be roots, including the only files clause
		// would result in incorrect traversal since non-folders don't have children.
		// Just don't print it, however, the folder will still be explored.
		if onlyFiles && file.IsDir {
			continue
		}
		file.pretty(g.log, opt)
	}

	if !inTrash && !g.opts.InTrash {
		for _, file := range children {
			if !g.breadthFirst(file, "bread-crumbs", headPath, depth, g.opts.TypeMask, inTrash) {
				return false
			}
		}
		return true
	}
	return len(children) >= 1
}

func isHidden(p string, ignore bool) bool {
	if strings.HasPrefix(p, ".") {
		return !ignore
	}
	return false
}

func isMinimal(mask int) bool {
	return (mask & Minimal) != 0
}

func owners(mask int) bool {
	return (mask & Owners) != 0
}

func version(mask int) bool {
	return (mask & CurrentVersion) != 0
}

func nextPage() bool {
	var input string
	fmt.Printf("---More---")
	fmt.Scanln(&input)
	if len(input) >= 1 && strings.ToLower(input[:1]) == "q" {
		return false
	}
	return true
}
