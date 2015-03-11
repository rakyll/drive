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
)

func (g *Commands) Move() (err error) {
	argc := len(g.opts.Sources)
	if argc < 2 {
		return fmt.Errorf("move: expected <src> [src...] <dest>, instead got: %v", g.opts.Sources)
	}

	var newParent *File
	rest, dest := g.opts.Sources[:argc-1], g.opts.Sources[argc-1]

	if newParent, err = g.rem.FindByPath(dest); err != nil {
		return fmt.Errorf("dest: '%s' %v", dest, err)
	}

	if newParent == nil || !newParent.IsDir {
		return fmt.Errorf("dest: '%s' must be an existant folder", dest)
	}

	for _, src := range rest {
		g.move(src, newParent)
	}

	return nil
}

func (g *Commands) move(srcPath string, newParent *File) (err error) {
	var remSrc *File
	if remSrc, err = g.rem.FindByPath(srcPath); err != nil {
		return fmt.Errorf("src('%s') %v", srcPath, err)
	}

	if remSrc == nil {
		return fmt.Errorf("src: '%s' doesn't exist", srcPath)
	}
	// Avoid self-nesting
	if remSrc.Id == newParent.Id {
		return fmt.Errorf("move: cannot move to self")
	}

	if err = g.removeParent(remSrc.Id, srcPath); err != nil {
		return err
	}

	return g.rem.insertParent(remSrc.Id, newParent.Id)
}

func (g *Commands) removeParent(fileId, relToRootPath string) error {
	parentPath := g.parentPather(relToRootPath)
	parent, pErr := g.rem.FindByPath(parentPath)
	if pErr != nil {
		return pErr
	}
	if parent == nil {
		return fmt.Errorf("non existant parent '%s' for src", parentPath)
	}
	return g.rem.removeParent(fileId, parent.Id)
}

func (g *Commands) Rename() error {
	if len(g.opts.Sources) < 2 {
		return fmt.Errorf("rename: expecting <src> <newname>")
	}

	src, newName := g.opts.Sources[0], g.opts.Sources[1]
	remSrc, err := g.rem.FindByPath(src)
	if err != nil {
		return fmt.Errorf("%s: %v", src, err)
	}
	if remSrc == nil {
		return fmt.Errorf("%s does not exist", src)
	}

	parentPath := g.parentPather(src)
	newFullPath := filepath.Join(parentPath, newName)
	var dupCheck *File
	dupCheck, err = g.rem.FindByPath(newFullPath)
	if err == nil && dupCheck != nil {
		if dupCheck.Id == remSrc.Id { // Trying to rename self
			return nil
		}
		if !g.opts.Force {
			return fmt.Errorf("%s already exists. Use `%s` flag to override this behaviour", newFullPath, ForceKey)
		}
	}

	_, err = g.rem.rename(remSrc.Id, newName)
	return err
}
