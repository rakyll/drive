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
	"io/ioutil"
	"os"
	gopath "path"
	"strings"

	"github.com/rakyll/drive/config"
)

// Pushes to remote if local path exists and in a god context. If path is a
// directory, it recursively pushes to the remote if there are local changes.
// It doesn't check if there are local changes if isForce is set.
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

func (g *Commands) Push() (err error) {
	var cl []*Change
	fmt.Println("Resolving...")

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
	return g.resolve(cl, nil)
}

func (g *Commands) resolve(cl []*Change, lastErr error) (err error) {
	if lastErr != nil {
		return lastErr
	}
	if ok := printChangeList(cl, g.opts.IsNoPrompt); ok {
		return g.playPushChangeList(cl)
	}
	g.clearMountPoints()
	return
}

func (g *Commands) clearMountPoints() {
	for _, mtpt := range g.opts.Mounts {
		mtpt.Unmount()
	}
}

func (g *Commands) playPushChangeList(cl []*Change) (err error) {
	defer g.clearMountPoints()
	g.taskStart(len(cl))
	for _, c := range cl {
		switch c.Op() {
		case OpMod:
			g.remoteMod(c)
		case OpAdd:
			g.remoteAdd(c)
		case OpDelete:
			g.remoteDelete(c)
		}
	}
	g.taskFinish()
	return err
}

func (g *Commands) remoteMod(change *Change) (err error) {
	defer g.taskDone()
	absPath := g.context.AbsPathOf(change.Path)
	var updated, parent *File
	if change.Dest != nil {
		change.Src.Id = change.Dest.Id // TODO: bad hack
	}

	p := strings.Split(change.Path, "/")
	p = append([]string{"/"}, p[:len(p)-1]...)
	if parent, err = g.rem.FindByPath(gopath.Join(p...)); err != nil {
		return
	}

	var body *os.File
	if !change.Src.IsDir {
		body, err = os.Open(absPath)
		if err != nil {
			return err
		}
	}
	if updated, err = g.rem.Upsert(parent.Id, change.Src, body); err != nil {
		return
	}
	return os.Chtimes(absPath, updated.ModTime, updated.ModTime)
}

func (g *Commands) remoteAdd(change *Change) (err error) {
	return g.remoteMod(change)
}

func (g *Commands) remoteDelete(change *Change) (err error) {
	defer g.taskDone()
	return g.rem.Trash(change.Dest.Id)
}

func list(context *config.Context, path string, hidden bool) (files []*File, err error) {
	absPath := context.AbsPathOf(path)
	var f []os.FileInfo
	if f, err = ioutil.ReadDir(absPath); err != nil {
		return
	}
	for _, file := range f {
		if hidden || !strings.HasPrefix(file.Name(), ".") {
			files = append(files, NewLocalFile(gopath.Join(absPath, file.Name()), file))
		}
	}
	return
}
