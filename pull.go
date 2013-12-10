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

package god

import (
	"fmt"
	"io"
	"os"
	gopath "path"
	"sync"

	"github.com/rakyll/god/types"
)

// Pull from remote if remote path exists and in a god context. If path is a
// directory, it recursively pulls from the remote if there are remote changes.
// It doesn't check if there are remote changes if isForce is set.
func (g *God) Pull() error {
	if g.context == nil {
		return ErrNoContext
	}
	// TODO: handle errors
	var cl []*types.Change
	cl, _ = g.createPullChangeListRecv(g.opts.Path, nil)
	// TODO: promt for approval
	return g.playPullChangeList(cl)
}

func (g *God) createPullChangeListRecv(path string, r *types.File) (cl []*types.Change, err error) {
	//defer wg.Done()
	if r == nil {
		if r, err = g.rem.FindByPath(path); err != nil {
			return nil, err
		}
	}

	var l *types.File
	absPath := g.context.AbsPathOf(path)
	localinfo, _ := os.Stat(absPath)
	if localinfo != nil {
		l = types.NewLocalFile(absPath, localinfo)
	}

	cl = []*types.Change{&types.Change{Path: path, Src: r, Dest: l}}
	if !g.opts.IsRecursive || !r.IsDir {
		return cl, nil
	}

	// look-up for children
	var children []*types.File
	children, err = g.rem.FindByParentId(r.Id)
	if err != nil {
		return
	}
	for _, child := range children {
		//go func(cl []*types.Change, wg *sync.WaitGroup, path string, child *types.File) {
		childChanges, _ := g.createPullChangeListRecv(gopath.Join(path, child.Name), child)
		cl = append(cl, childChanges...)
		//}(cl, wg, path, child)
	}
	return cl, nil
}

func (g *God) playPullChangeList(cl []*types.Change) (err error) {
	var wg sync.WaitGroup
	for _, c := range cl {
		switch c.Op() {
		case types.OpMod:
			wg.Add(1)
			go g.localMod(&wg, c)
		case types.OpAdd:
			wg.Add(1)
			go g.localAdd(&wg, c)
		case types.OpDelete:
			wg.Add(1)
			go g.localDelete(&wg, c)
		}
	}
	wg.Wait()
	return err
}

func (g *God) localMod(wg *sync.WaitGroup, change *types.Change) (err error) {
	defer wg.Done()
	fmt.Println("M", change.Path)
	destAbsPath := g.context.AbsPathOf(change.Path)
	if change.Src.BlobAt != "" {
		// download and replace
		if err = g.download(change); err != nil {
			return
		}
	}
	return os.Chtimes(destAbsPath, change.Src.ModTime, change.Src.ModTime)
}

func (g *God) localAdd(wg *sync.WaitGroup, change *types.Change) (err error) {
	defer wg.Done()
	fmt.Println("+", change.Path)
	destAbsPath := g.context.AbsPathOf(change.Path)
	if change.Src.IsDir {
		return os.Mkdir(destAbsPath, os.ModeDir|0755)
	}

	if change.Src.BlobAt != "" {
		// download and create
		if err = g.download(change); err != nil {
			return
		}
	}
	return os.Chtimes(destAbsPath, change.Src.ModTime, change.Src.ModTime)
}

// TODO: no one calls localdelete
func (g *God) localDelete(wg *sync.WaitGroup, change *types.Change) (err error) {
	defer wg.Done()
	fmt.Println("-", change.Path)
	return os.RemoveAll(change.Dest.BlobAt)
}

func (g *God) download(change *types.Change) (err error) {
	destAbsPath := g.context.AbsPathOf(change.Path)
	var fo *os.File
	fo, err = os.Create(destAbsPath)
	if err != nil {
		return
	}

	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			return
		}
	}()

	var blob io.ReadCloser
	defer func() {
		if blob != nil {
			blob.Close()
		}
	}()
	blob, err = g.rem.Download(change.Src.Id)
	if err != nil {
		return err
	}
	_, err = io.Copy(fo, blob)
	return
}
