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
	"path"
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

func (g *Commands) resolveChangeListRecv(
	isPush bool, d, p string, r *File, l *File) (cl []*Change, err error) {
	var change *Change
	// fmt.Printf("Parent: %s Path: %s\n", d, p)
	if isPush {
		change = &Change{Path: p, Src: l, Dest: r, Parent: d}
	} else {
		change = &Change{Path: p, Src: r, Dest: l, Parent: d}
	}
	if change.Op() != OpNone {
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
		fullPath := urlToPath(path.Join(d, p), false)
		// fmt.Printf("d: %s p: %s fmp: %s\n", d, p, fullPath)
		localChildren, err = list(g.context, fullPath, g.opts.Hidden, isPush)
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
	// m := p
	// p = path.Join(d, p)
	// fmt.Printf("pax: %s p: %s d: %s\n", p, m, d)
	for _, l := range dirlist {
		go func(wg *sync.WaitGroup, isPush bool, cl *[]*Change, p string, l *dirList) {
			defer wg.Done()
			// Note that using path.Join converts an '//'* to '/' so better to use strings.Join
			joined := strings.Join([]string{p, l.Name()}, "/")
			// fmt.Printf("p: %s v: %s joined  now: %s\n", p, l.Name(), joined)
			childChanges, _ := g.resolveChangeListRecv(isPush, p, joined, l.remote, l.local)
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

func printChangeList(changes []*Change, isNoPrompt bool) bool {
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
