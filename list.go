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

func (g *Commands) List() (err error) {
	root := g.context.AbsPathOf("")
	var relPath string
	var relPaths []string
	var remotes []*File

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
		r, rErr := g.rem.FindByPath(relPath)
		if rErr != nil {
			fmt.Printf("%v: '%s'\n", rErr, relPath)
			return
		}
		remotes = append(remotes, r)
	}

	for _, r := range remotes {
		g.depthFirst(r.Id, "/"+r.Name, g.opts.Depth)
	}

	return
}

func breakdownFile(p string, f *drive.File) {
	if f == nil {
		return
	}
	fmt.Printf("%s/%s\n", p, f.Title)
}

func (g *Commands) depthFirst(parentId, parentName string, depth int) bool {
	if depth == 0 {
		return false
	}
	if depth > 0 {
		depth -= 1
	}
	pageToken := ""

	for {
		req := g.rem.service.Files.List()
		req.Q(fmt.Sprintf("'%s' in parents and trashed=false", parentId))
		// TODO: Get pageSize from g.opts
		req.MaxResults(30)

		if pageToken != "" {
			req = req.PageToken(pageToken)
		}
		res, err := req.Do()
		if err != nil {
			return false
		}

		for _, file := range res.Items {
			breakdownFile(parentName, file)
			g.depthFirst(file.Id, parentName+"/"+file.Title, depth)
		}

		pageToken = res.NextPageToken
		if pageToken == "" {
			break
		}

		var input string
		fmt.Printf("---Next---")
		fmt.Scanln(&input)
		if len(input) >= 1 && strings.ToLower(input[:1]) == "q" {
			return false
		}
	}
	return true
}
