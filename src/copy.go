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
	"errors"
	"fmt"
)

var ErrPathNotDir = errors.New("not a directory")

func (g *Commands) Copy() error {
	argc := len(g.opts.Sources)
	if argc < 2 {
		return fmt.Errorf("expecting src [src1....] dest got: %v", g.opts.Sources)
	}

	sources, dest := g.opts.Sources[:argc-1], g.opts.Sources[argc-1]
	destFile, err := g.rem.FindByPath(dest)
	if err != nil && err != ErrPathNotExists {
		return fmt.Errorf("destination: %s err: %v", dest, err)
	}

	if destFile != nil && !destFile.IsDir && len(sources) > 1 {
		return fmt.Errorf("%s: ", dest, ErrPathNotDir)
	}

	dirCount := 0
	regCount := 0
	destId := ""
	dirDest := false
	if destFile != nil {
		destId = destFile.Id
		dirDest = destFile.IsDir
	}

	files := make([]*File, len(sources))
	for i, relToRootPath := range sources {
		file, err := g.rem.FindByPath(relToRootPath)
		if err != nil {
			return err
		}
		if !file.IsDir {
			regCount += 1
		} else if file.IsDir {
			if !dirDest {
				return fmt.Errorf("%s: %v yet %s is a directory",
					dest, ErrPathNotDir, relToRootPath)
			}
			if !g.opts.Recursive {
				return fmt.Errorf("%s is a folder yet `recursive` is not defined", relToRootPath)
			}
			dirCount += 1
		}
		if file.Id == destId {
			return fmt.Errorf("%s and %s are the same file", relToRootPath, dest)
		}

		files[i] = file
	}

	if dirDest {
		destFile, err = g.remoteMkdirAll(dest)
		if err != nil {
			return fmt.Errorf("mkdirAll %s: %v", dest, err)
		}
	}

	for _, f := range files {
		copied, err := g.copy(f.Name, f, destFile)
		fmt.Println(copied, err)
	}

	return nil
}

func (g *Commands) copy(destTitle string, src, dest *File) (*File, error) {
	if !src.IsDir {
		parentId := ""
		if destTitle == "" {
			destTitle = src.Name
		}
		if dest != nil && dest.IsDir {
			parentId = dest.Id
		}
		if !src.Copyable {
			return nil, fmt.Errorf("%s (%s) is not copyable", src.Name, src.Id)
		}
		return g.rem.copy(destTitle, src.Id, parentId)
	} else {
		content := g.rem.findChildren(src.Id)
		for file := range content {
			fmt.Println("Patch me!", file.Name)
		}
	}
	return nil, nil
}
