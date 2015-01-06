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
	"os/signal"
	gopath "path"
	"sort"
	"strings"

	"github.com/odeke-em/drive/config"
)

// Pushes to remote if local path exists and in a gd context. If path is a
// directory, it recursively pushes to the remote if there are local changes.
// It doesn't check if there are local changes if isForce is set.
func (g *Commands) Push() (err error) {
	defer g.clearMountPoints()

	root := g.context.AbsPathOf("")
	var cl []*Change

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// To Ensure mount points are cleared in the event of external exceptios
	go func() {
		_ = <-c
		g.clearMountPoints()
		os.Exit(1)
	}()

	for _, relToRootPath := range g.opts.Sources {
		fsPath := g.context.AbsPathOf(relToRootPath)
		ccl, cErr := g.changeListResolve(relToRootPath, fsPath, true)
		if cErr == nil && len(ccl) > 0 {
			cl = append(cl, ccl...)
		}
	}

	for _, mt := range g.opts.Mounts {
		ccl, cerr := lonePush(g, root, mt.Name, mt.MountPath)
		if cerr == nil {
			cl = append(cl, ccl...)
		}
	}

	ok := printChangeList(cl, g.opts.NoPrompt, g.opts.NoClobber)
	if ok {
		pushSize := reduceToSize(cl, true)

		quotaStatus, qErr := g.QuotaStatus(pushSize)
		if qErr != nil {
			return qErr
		}
		unSafe := false
		switch quotaStatus {
		case AlmostExceeded:
			fmt.Println("\033[92mAlmost exceeding your drive quota\033[00m")
		case Exceeded:
			fmt.Println("\033[91mThis change will exceed your drive quota\033[00m")
			unSafe = true
		}
		if unSafe {
			fmt.Printf(" projected size: %d (%d)\n", pushSize, prettyBytes(pushSize))
			if !promptForChanges() {
				return
			}
		}
		return g.playPushChangeList(cl)
	}
	return
}

func (g *Commands) playPushChangeList(cl []*Change) (err error) {
	g.taskStart(len(cl))

	// TODO: Only provide precedence ordering if all the other options are allowed
	// Currently noop on sorting by precedence
	if false && !g.opts.NoClobber {
		sort.Sort(ByPrecedence(cl))
	}

	for _, c := range cl {
		if c.Src == nil {
			// fmt.Println("Push: BUG ON", c.Path, c.Symbol())
			continue
		}
		switch c.CoercedOp(g.opts.NoClobber) {
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

func lonePush(g *Commands, parent, absPath, path string) (cl []*Change, err error) {
	r, err := g.rem.FindByPath(absPath)
	if err != nil && err != ErrPathNotExists {
		return
	}

	var l *File
	localinfo, _ := os.Stat(path)
	if localinfo != nil {
		l = NewLocalFile(path, localinfo)
	}

	return g.resolveChangeListRecv(true, parent, absPath, r, l)
}

func (g *Commands) remoteMod(change *Change) (err error) {
	defer g.taskDone()
	absPath := g.context.AbsPathOf(change.Path)
	var parent *File
	if change.Dest != nil {
		change.Src.Id = change.Dest.Id // TODO: bad hack
	}

	p := strings.Split(change.Path, "/")
	p = append([]string{"/"}, p[:len(p)-1]...)
	parent, err = g.rem.FindByPath(gopath.Join(p...))
	if err != nil {
		fmt.Println(parent, err)
		return
	}

	var body *os.File
	if !change.Src.IsDir {
		body, err = os.Open(absPath)
		if err != nil {
			return err
		}
	}
	_, err = g.rem.Upsert(parent.Id, change.Src, body)
	return err
}

func (g *Commands) remoteAdd(change *Change) (err error) {
	return g.remoteMod(change)
}

func (g *Commands) remoteUntrash(change *Change) (err error) {
	defer g.taskDone()
	return g.rem.Untrash(change.Src.Id)
}

func (g *Commands) remoteDelete(change *Change) (err error) {
	defer g.taskDone()
	return g.rem.Trash(change.Dest.Id)
}

func list(context *config.Context, p string, hidden bool) (files []*File, err error) {
	absPath := context.AbsPathOf(p)
	var f []os.FileInfo
	f, err = ioutil.ReadDir(absPath)
	if err != nil {
		return
	}
	for _, file := range f {
		if hidden || !strings.HasPrefix(file.Name(), ".") {
			files = append(files, NewLocalFile(gopath.Join(absPath, file.Name()), file))
		}
	}
	return
}
