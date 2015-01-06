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

type sizeCounter struct {
	count int64
	src   int64
	dest  int64
}

func (t *sizeCounter) String() string {
	str := fmt.Sprintf("count %v", t.count)
	if t.src > 0 {
		str = fmt.Sprintf("%s src: %v", str, prettyBytes(t.src))
	}
	if t.dest > 0 {
		str = fmt.Sprintf("%s src: %v", str, prettyBytes(t.dest))
	}
	return str
}

// Resolves the local path relative to the root directory
// Returns the path relative to the remote, the abspath on disk and an error if any
func (g *Commands) pathResolve() (relPath, absPath string, err error) {
	root := g.context.AbsPathOf("")
	absPath = g.context.AbsPathOf(g.opts.Path)
	relPath = ""

	if absPath != root {
		relPath, err = filepath.Rel(root, absPath)
		if err != nil {
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

func (g *Commands) changeListResolve(relToRoot, fsPath string, isPush bool) (cl []*Change, err error) {
	var r, l *File
	r, err = g.rem.FindByPath(relToRoot)
	if err != nil {
		fmt.Println(err)
		// We cannot pull from a non-existant remote
		if !isPush {
			return
		}
	}

	localinfo, _ := os.Stat(fsPath)
	if localinfo != nil {
		l = NewLocalFile(fsPath, localinfo)
	}

	fmt.Println("Resolving...")
	cl, err = g.resolveChangeListRecv(isPush, relToRoot, relToRoot, r, l)
	return
}

func (g *Commands) clearMountPoints() {
	for _, mtpt := range g.opts.Mounts {
		mtpt.Unmount()
	}
}

func (g *Commands) resolveChangeListRecv(
	isPush bool, d, p string, r *File, l *File) (cl []*Change, err error) {
	var change *Change
	if isPush {
		// Handle the case of doc files for which we don't have a direct download
		// url but have exportable links. These files should not be clobbered on push
		if hasExportLinks(r) {
			return cl, nil
		}
		change = &Change{Path: p, Src: l, Dest: r, Parent: d, Force: g.opts.Force}
	} else {
		if !g.opts.Force && hasExportLinks(r) {
			// The case when we have files that don't provide the download urls
			// but exportable links, we just need to check that mod times are the same.
			if r.MatchDirness(l) && r.ModTime.Equal(l.ModTime) {
				return cl, nil
			}
		}
		change = &Change{Path: p, Src: r, Dest: l, Parent: d, Force: g.opts.Force}
	}

	if g.opts.Force || change.CoercedOp(g.opts.NoClobber) != OpNone {
		cl = append(cl, change)
	}
	if !g.opts.Recursive {
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
		remoteChildren, err = g.rem.FindByParentId(r.Id, g.opts.Hidden)
		if err != nil {
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
			// Avoiding path.Join which normalizes '/+' to '/'
			var joined string
			if p == "/" {
				joined = "/" + l.Name()
			} else {
				joined = strings.Join([]string{p, l.Name()}, "/")
			}
			childChanges, _ := g.resolveChangeListRecv(isPush, p, joined, l.remote, l.local)
			*cl = append(*cl, childChanges...)
		}(&wg, isPush, &cl, p, l)
	}
	wg.Wait()

	return cl, nil
}

func merge(remotes, locals []*File) (merged []*dirList) {
	localMap := map[string]*File{}

	// Add support for FileSystems that allow same names but different files.

	for _, l := range locals {
		localMap[l.Name] = l
	}

	for _, r := range remotes {
		list := &dirList{remote: r}

		// look for local
		l, ok := localMap[r.Name]
		if ok {
			list.local = l
			delete(localMap, r.Name)
		}
		merged = append(merged, list)
	}

	// if anything left in locals, add to the dir listing
	for _, l := range localMap {
		merged = append(merged, &dirList{local: l})
	}
	return
}

func summarizeChanges(changes []*Change, reduce bool) {
	for _, c := range changes {
		if c.Op() != OpNone {
			fmt.Println(c.Symbol(), c.Path)
		}
	}
	if reduce {
		opMap := map[int]sizeCounter{}

		for _, c := range changes {
			op := c.Op()
			counter := opMap[op]
			counter.count += 1
			if c.Src != nil {
				counter.src += c.Src.Size
			}
			if c.Dest != nil {
				counter.dest += c.Dest.Size
			}
			opMap[op] = counter
		}

		for op, counter := range opMap {
			if counter.count < 1 {
				continue
			}
			_, name := opToString(op)
			fmt.Printf("%s %s\n", name, counter.String())
		}
	}
}

func printChangeList(changes []*Change, noPrompt bool, noClobber bool) bool {
	if len(changes) == 0 {
		fmt.Println("Everything is up-to-date.")
		return false
	}

	summarizeChanges(changes, !noPrompt)

	if noPrompt {
		return true
	}

	input := "Y"
	fmt.Print("Proceed with the changes? [Y/n]: ")
	fmt.Scanln(&input)
	return strings.ToUpper(input) == "Y"
}
