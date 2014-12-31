// Copyright 2013 Google Inc. All Rights Reserved.
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
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

type dirList struct {
	remote *File
	local  *File
}

func (d *dirList) Name() string {
	if d.remote != nil {
		return d.remote.Name
	}
	return d.local.Name
}

// Resolves the local path relative to the root directory
// Returns the path relative to the remote, the abspath on disk and an error if any
func (g *Commands) pathResolve() (relPath, absPath string, err error) {
	root := g.context.AbsPathOf("")
	absPath = g.context.AbsPathOf(g.opts.Path)
	relPath = ""

	if absPath != root {
		if relPath, err = filepath.Rel(root, absPath); err != nil {
			return
		}
	} else {
		var cwd string
		if cwd, err = os.Getwd(); err != nil {
			return
		}
		if cwd == root {
			relPath = ""
		} else if relPath, err = filepath.Rel(root, cwd); err != nil {
			return
		}
	}

	relPath = strings.Join([]string{"", relPath}, "/")
	return
}

func (g *Commands) changeListResolve(isPush bool) (cl []*Change, err error) {
	var relPath, absPath string
	relPath, absPath, err = g.pathResolve()
	if err != nil {
		return
	}
	var r, l *File
	if r, err = g.rem.FindByPath(relPath); err != nil {
		// We cannot pull from a non-existant remote
		if !isPush {
			return
		}
	}
	localinfo, _ := os.Stat(absPath)
	if localinfo != nil {
		l = NewLocalFile(relPath, localinfo)
	}

	fmt.Println("Resolving...")
	cl, err = g.resolveChangeListRecv(isPush, relPath, r, l)
	if err != nil {
		return
	}

	if isPush {
		cl = append(cl, clForPush(g)...)
	}
	return
}

// Resolves the local path relative to the root directory
// then performs either Push or Pull depending on 'isPush'
func (g *Commands) syncByRelativePath(isPush bool) (err error) {
	defer g.clearMountPoints()

	var cl []*Change
	cl, err = g.changeListResolve(isPush)

	ok := printChangeList(cl, g.opts.IsNoPrompt, g.opts.NoClobber)
	if ok {
		if isPush {
			return g.playPushChangeList(cl)
		}
		return g.playPullChangeList(cl, g.opts.Exports)
	}
	return
}

func clForPush(g *Commands) (cl []*Change) {
	absPath := g.context.AbsPathOf("/")
	for _, indrivePath := range g.opts.Sources {
		lcl, eerr := lonePush(g, indrivePath, absPath)
		if eerr == nil {
			cl = append(cl, lcl...)
		}
	}

	for _, mt := range g.opts.Mounts {
		ccl, cerr := lonePush(g, mt.Name, mt.MountPath)
		if cerr == nil {
			cl = append(cl, ccl...)
		}
	}

	return
}

func (g *Commands) clearMountPoints() {
	for _, mtpt := range g.opts.Mounts {
		mtpt.Unmount()
	}
}

func lonePush(g *Commands, absPath, path string) (cl []*Change, err error) {
	r, err := g.rem.FindByPath(absPath)
	if err != nil && err != ErrPathNotExists {
		return
	}

	var l *File
	localinfo, _ := os.Stat(path)
	if localinfo != nil {
		l = NewLocalFile(absPath, localinfo)
	}

	return g.resolveChangeListRecv(true, absPath, r, l)
}

func (g *Commands) resolveChangeListRecv(
	isPush bool, p string, r *File, l *File) (cl []*Change, err error) {
	var change *Change
	if isPush {
		// Handle the case of doc files for which we don't have a direct download
		// url but have exportable links. These files should not be clobbered on push
		if hasExportLinks(r) {
			return cl, nil
		}
		change = &Change{Path: p, Src: l, Dest: r}
	} else {
		change = &Change{Path: p, Src: r, Dest: l}
	}

	if change.CoercedOp(g.opts.NoClobber) != OpNone {
		cl = append(cl, change)
	}
	if !g.opts.IsRecursive {
		return cl, nil
	}
	// TODO: handle cases where remote and local type don't match
	if !isPush && r != nil && !r.IsDir {
		return cl, nil
	}

	if isPush && l != nil && !l.IsDir {
		return cl, nil
	}

	// look-up for children
	var localChildren []*File
	if l != nil {
		localChildren, err = list(g.context, p, g.opts.Hidden)
		if err != nil {
			return
		}
	}

	var remoteChildren []*File
	if r != nil {
		if remoteChildren, err = g.rem.FindByParentId(r.Id); err != nil {
			return
		}
	}

	// TODO: limit the number of active tasks for children lookups
	dirlist := merge(remoteChildren, localChildren)
	var wg sync.WaitGroup
	wg.Add(len(dirlist))
	for _, l := range dirlist {
		go func(wg *sync.WaitGroup, isPush bool, cl *[]*Change, p string, l *dirList) {
			defer wg.Done()
			childChanges, _ := g.resolveChangeListRecv(isPush, path.Join(p, l.Name()), l.remote, l.local)
			*cl = append(*cl, childChanges...)
		}(&wg, isPush, &cl, p, l)
	}
	wg.Wait()
	return cl, nil
}

func merge(remotes, locals []*File) (merged []*dirList) {
	for _, r := range remotes {
		list := &dirList{remote: r}
		// look for local
		for i, l := range locals {
			if l.Name == r.Name {
				list.local = l
				locals = append(locals[:i], locals[i+1:]...)
				break
			}
		}
		merged = append(merged, list)
	}
	// if anything left in locals, add to the dir listing
	for _, l := range locals {
		merged = append(merged, &dirList{local: l})
	}
	return
}

func printChangeList(changes []*Change, isNoPrompt bool, noClobber bool) bool {
	for _, c := range changes {
		if c.Op() != OpNone {
			fmt.Println(c.Symbol(), c.Path)
		}
	}
	if len(changes) == 0 {
		fmt.Println("Everything is up-to-date.")
		return false
	}
	if isNoPrompt {
		return true
	}
	input := "Y"
	fmt.Print("Proceed with the changes? [Y/n]: ")
	fmt.Scanln(&input)
	return strings.ToUpper(input) == "Y"
}
