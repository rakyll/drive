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
	"sync"
)

type AccountType int

const (
	Anyone = 1 << iota
	User
	Domain
	Group
)

type Role int

const (
	Owner = 1 << iota
	Reader
	Writer
	Commenter
)

func (r *Role) String() string {
	switch *r {
	case Owner:
		return "owner"
	case Reader:
		return "reader"
	case Writer:
		return "writer"
	case Commenter:
		return "commenter"
	}
	return "unknown"
}

func (a *AccountType) String() string {
	switch *a {
	case Anyone:
		return "anyone"
	case User:
		return "user"
	case Domain:
		return "domain"
	case Group:
		return "group"
	}
	return "unknown"
}

func (g *Commands) resolveRemotePaths(relToRootPaths []string) (files []*File) {
	var wg sync.WaitGroup

	wg.Add(len(relToRootPaths))
	for _, relToRoot := range relToRootPaths {
		go func(p string, wgg *sync.WaitGroup) {
			defer wgg.Done()
			file, err := g.rem.FindByPath(p)
			if err != nil || file == nil {
				return
			}
			files = append(files, file)
		}(relToRoot, &wg)
	}
	wg.Wait()
	return files
}

func (c *Commands) Share() (err error) {
	return fmt.Errorf("Unimplemented")
}
