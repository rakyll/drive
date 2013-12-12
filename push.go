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

package gd

import (
	"fmt"
	"io/ioutil"
	"os"
	gopath "path"
	"strings"

	"github.com/rakyll/gd/config"
	"github.com/rakyll/gd/remote"
	"github.com/rakyll/gd/third_party/github.com/cheggaaa/pb"
	"github.com/rakyll/gd/types"
)

// Pushes to remote if local path exists and in a god context. If path is a
// directory, it recursively pushes to the remote if there are local changes.
// It doesn't check if there are local changes if isForce is set.
func (g *Gd) Push() (err error) {
	if g.context == nil {
		return ErrNoContext
	}

	absPath := g.context.AbsPathOf(g.opts.Path)
	r, err := g.rem.FindByPath(g.opts.Path)
	if err != nil && err != remote.ErrPathNotExists {
		return err
	}

	var l *types.File
	localinfo, _ := os.Stat(absPath)
	if localinfo != nil {
		l = types.NewLocalFile(absPath, localinfo)
	}

	fmt.Println("Resolving...")
	var cl []*types.Change
	if cl, err = g.resolveChangeListRecv(true, g.opts.Path, r, l); err != nil {
		return err
	}

	if ok := printChangeList(cl); !ok {
		return
	}
	return g.playPushChangeList(cl)
}

func (g *Gd) playPushChangeList(cl []*types.Change) (err error) {
	if len(cl) > 0 {
		g.progress = pb.New(len(cl))
		g.progress.Start()
	}
	for _, c := range cl {
		switch c.Op() {
		case types.OpMod:
			g.remoteMod(c)
		case types.OpAdd:
			g.remoteAdd(c)
		case types.OpDelete:
			g.remoteDelete(c)
		}
	}
	if g.progress != nil {
		defer g.progress.Finish()
	}
	return err
}

func (g *Gd) remoteMod(change *types.Change) (err error) {
	if g.progress != nil {
		defer g.progress.Increment()
	}

	absPath := g.context.AbsPathOf(change.Path)
	var updated, parent *types.File
	if change.Dest != nil {
		change.Src.Id = change.Dest.Id // TODO: bad hack
	}

	p := strings.Split(change.Path, "/")
	p = append([]string{"/"}, p[:len(p)-1]...)
	if parent, err = g.rem.FindByPath(gopath.Join(p...)); err != nil {
		return
	}

	// TODO: mimetype detection
	var body *os.File
	if !change.Src.IsDir {
		// TODO: handle errors, read more efficiently for large files
		body, _ = os.Open(absPath)
	}
	// TODO: append blob
	if updated, err = g.rem.Upsert(parent.Id, change.Src, body); err != nil {
		return
	}
	return os.Chtimes(absPath, updated.ModTime, updated.ModTime)
}

func (g *Gd) remoteAdd(change *types.Change) (err error) {
	return g.remoteMod(change)
}

// TODO: no one calls localdelete
func (g *Gd) remoteDelete(change *types.Change) (err error) {
	if g.progress != nil {
		defer g.progress.Increment()
	}
	return g.rem.Trash(change.Dest.Id)
}

func list(context *config.Context, path string) (files []*types.File, err error) {
	absPath := context.AbsPathOf(path)
	var f []os.FileInfo
	if f, err = ioutil.ReadDir(absPath); err != nil {
		return
	}
	for _, file := range f {
		// ignore hidden files
		if !strings.HasPrefix(file.Name(), ".") {
			files = append(files, types.NewLocalFile(gopath.Join(absPath, file.Name()), file))
		}
	}
	return
}
