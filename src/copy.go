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

type copyArgs struct {
	destPath string
	src      *File
	dest     *File
}

func (g *Commands) Copy() error {
	argc := len(g.opts.Sources)
	if argc < 2 {
		return fmt.Errorf("expecting src [src1....] dest got: %v", g.opts.Sources)
	}

	end := argc - 1
	sources, dest := g.opts.Sources[:end], g.opts.Sources[end]

	destFile, err := g.rem.FindByPath(dest)
	if err != nil && err != ErrPathNotExists {
		return fmt.Errorf("destination: %s err: %v", dest, err)
	}

	multiPaths := len(sources) > 1
	if multiPaths {
		if destFile != nil && !destFile.IsDir {
			return fmt.Errorf("%s: %v", dest, ErrPathNotDir)
		}
		_, err := g.remoteMkdirAll(dest)
		if err != nil {
			return err
		}
	}

	for _, srcPath := range sources {
		srcFile, srcErr := g.rem.FindByPath(srcPath)
		if srcErr != nil {
			g.log.LogErrf("%s: %v\n", srcPath, srcErr)
			continue
		}

		_, copyErr := g.copy(srcFile, dest)
		if copyErr != nil {
			g.log.LogErrf("%s: %v\n", srcPath, copyErr)
		}
	}

	return nil
}

func (g *Commands) copy(src *File, destPath string) (*File, error) {
	if src == nil {
		return nil, fmt.Errorf("non existant src")
	}

	if !src.IsDir {
		if !src.Copyable {
			return nil, fmt.Errorf("%s is non-copyable", src.Name)
		}

		destDir, destBase := g.pathSplitter(destPath)
		destParent, destParErr := g.remoteMkdirAll(destDir)

		if destParErr != nil {
			return nil, destParErr
		}

		parentId := destParent.Id
		destFile, destErr := g.rem.FindByPath(destPath)
		if destErr != nil && destErr != ErrPathNotExists {
			return nil, destErr
		}
		if destFile != nil && destFile.IsDir {
			parentId = destFile.Id
			destBase = src.Name
		}
		return g.rem.copy(destBase, parentId, src)
	}

	destFile, destErr := g.remoteMkdirAll(destPath)
	if destErr != nil {
		return nil, destErr
	}

	children := g.rem.findChildren(src.Id, false)
	for child := range children {
		_, childErr := g.copy(child, destPath+"/"+child.Name)
		if childErr != nil {
			return nil, childErr
		}
	}

	return destFile, nil
}
