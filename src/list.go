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
	drive "github.com/google/google-api-go-client/drive/v2"
	"path/filepath"
	"strings"

	spinner "github.com/odeke-em/cli-spinner"
)

var BytesPerKB = float64(1024)

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
	var relPaths []string
	var remotes []*File

	resolver := g.rem.FindByPath
	if g.opts.InTrash {
		resolver = g.rem.FindByPathTrashed
	}

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
		relPaths = append(relPaths, relPath)
		r, rErr := resolver(relPath)
		if rErr != nil {
			fmt.Printf("%v: '%s'\n", rErr, relPath)
			return
		}
		remotes = append(remotes, r)
	}

	for _, r := range remotes {
		if !g.breadthFirst(r.Id, "", r.Name, g.opts.Depth, g.opts.TypeMask, false) {
			break
		}
	}

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
				sFile.pretty(opt)
			}
		}
	}

	return
}

func (f *File) pretty(opt attribute) {
	fmtdPath := fmt.Sprintf("%s/%s", opt.parent, f.Name)
	if opt.minimal {
		fmt.Println(fmtdPath)

		if owners(opt.mask) && len(f.OwnerNames) >= 1 {
			fmt.Printf(" %s ", strings.Join(f.OwnerNames, " & "))
		}
		return
	}

	if f.IsDir {
		fmt.Printf("d")
	} else {
		fmt.Printf("-")
	}
	if f.Shared {
		fmt.Printf("s")
	} else {
		fmt.Printf("-")
	}

	if f.UserPermission != nil {
		fmt.Printf(" %-10s ", f.UserPermission.Role)
	}

	if owners(opt.mask) && len(f.OwnerNames) >= 1 {
		fmt.Printf(" %s ", strings.Join(f.OwnerNames, " & "))
	}

	fmt.Printf(" %-10s\t%-60s\t\t%-20s\n", prettyBytes(f.Size), fmtdPath, f.ModTime)
}

func buildExpression(parentId string, typeMask int, inTrash bool) string {
	var exprBuilder []string

	if inTrash || (typeMask&InTrash) != 0 {
		exprBuilder = append(exprBuilder, "trashed=true")
	} else {
		exprBuilder = append(exprBuilder, fmt.Sprintf("'%s' in parents", parentId), "trashed=false")
	}

	// Folder and NonFolder are mutually exclusive.
	if (typeMask & Folder) != 0 {
		exprBuilder = append(exprBuilder, fmt.Sprintf("mimeType = '%s'", DriveFolderMimeType))
	}
	return strings.Join(exprBuilder, " and ")
}

func (g *Commands) breadthFirst(parentId, parent,
	child string, depth, typeMask int, inTrash bool) bool {

	// A depth of < 0 means traverse as deep as you can
	if depth == 0 {
		// At the end of the line, this was successful.
		return true
	}
	if depth > 0 {
		depth -= 1
	}

	headPath := ""
	if parent != "" {
		headPath = parent
	}
	if child != "" {
		headPath = headPath + "/" + child
	}

	pageToken := ""
	expr := "trashed=true"

	if !inTrash {
		expr = buildExpression(parentId, typeMask, g.opts.InTrash)
	}

	req := g.rem.service.Files.List()
	req.Q(expr)
	req.MaxResults(g.opts.PageSize)

	var children []*drive.File
	onlyFiles := (typeMask & NonFolder) != 0

	opt := attribute{
		minimal: isMinimal(g.opts.TypeMask),
		mask:    typeMask,
		parent:  headPath,
	}

	spin := spinner.New(10)
	for {
		spin.Reset()
		spin.Start()

		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		res, err := req.Do()
		if err != nil {
			fmt.Println(err)
			return false
		}

		spin.Stop()
		for _, file := range res.Items {
			rem := NewRemoteFile(file)
			if isHidden(file.Title, g.opts.Hidden) {
				continue
			}
			children = append(children, file)

			// The case in which only directories wanted is covered by the buildExpression clause
			// reason being that only folder are allowed to be roots, including the only files clause
			// would result in incorrect traversal since non-folders don't have children.
			// Just don't print it, however, the folder will still be explored.
			if onlyFiles && rem.IsDir {
				continue
			}
			rem.pretty(opt)
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}

		spin.Stop()
		if !g.opts.NoPrompt && !nextPage() {
			return false
		}
	}

	spin.Stop()

	if !inTrash && !g.opts.InTrash {
		for _, file := range children {
			if !g.breadthFirst(file.Id, headPath, file.Title, depth, typeMask, inTrash) {
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
