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
	drive "code.google.com/p/google-api-go-client/drive/v2"
	"fmt"
	"path/filepath"
	"strings"
)

var BytesPerKB = int64(1000)

type byteDescription func(b int64) string

func memoizeBytes() byteDescription {
	cache := map[int64]string{}
	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	maxLen := len(suffixes) - 1

	f := func(b int64) string {
		description, ok := cache[b]
		if ok {
			return description
		}

		i := 0
		for {
			if b/BytesPerKB < 1 || i >= maxLen {
				return fmt.Sprintf("%v%s", b, suffixes[i])
			}
			b /= BytesPerKB
			i += 1
		}
	}

	return f
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
		if !g.breadthFirst(r.Id, "/"+r.Name, g.opts.Depth, false) {
			break
		}
	}

	return
}

type attribute struct {
	human   bool
	minimal bool
	parent  string
}

func (f *File) pretty(opt attribute) {
	if opt.minimal {
		fmt.Printf("%s/%s\n", opt.parent, f.Name)
		return
	}

	if f.IsDir {
		fmt.Printf("d")
	} else {
		fmt.Printf("-")
	}
	if f.Shared {
		fmt.Printf("s ")
	} else {
		fmt.Printf("- ")
	}
	if f.UserPermission != nil {
		fmt.Printf("%-10s ", f.UserPermission.Role)
	}
	fPath := fmt.Sprintf("%s/%s", opt.parent, f.Name)
	fmt.Printf("%-10s %-6s %s", prettyBytes(f.Size), fPath, f.ModTime)
	fmt.Println()
}

func (g *Commands) breadthFirst(parentId, parentName string, depth int, inTrash bool) bool {
	if depth == 0 {
		return false
	}
	if depth > 0 {
		depth -= 1
	}
	pageToken := ""

	req := g.rem.service.Files.List()
	var expr string
	if inTrash || g.opts.InTrash {
		expr = "trashed=true"
	} else {
		expr = fmt.Sprintf("'%s' in parents and trashed=false", parentId)
	}
	req.Q(expr)

	// TODO: Get pageSize from g.opts
	req.MaxResults(20)

	var children []*drive.File

	for {
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		res, err := req.Do()
		if err != nil {
			return false
		}

		opt := attribute{
			human:   true,
			minimal: inTrash,
			parent:  parentName,
		}
		for _, file := range res.Items {
			rem := NewRemoteFile(file)
			if !g.opts.NoPrompt {
				rem.pretty(opt)
			}
			children = append(children, file)
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}

		if !nextPage() {
			return false
		}
	}

	if !inTrash && !g.opts.InTrash {
		for _, file := range children {
			if !g.breadthFirst(file.Id, parentName+"/"+file.Title, depth, inTrash) {
				return false
			}
		}
		return true
	}
	return len(children) >= 1
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
