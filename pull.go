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
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	maxNumOfConcPullTasks = 4
)

// Pull from remote if remote path exists and in a god context. If path is a
// directory, it recursively pulls from the remote if there are remote changes.
// It doesn't check if there are remote changes if isForce is set.
func (g *Commands) Pull() (err error) {
	var r, l *File
	if r, err = g.rem.FindByPath(g.opts.Path); err != nil {
		return
	}
	absPath := g.context.AbsPathOf(g.opts.Path)
	localinfo, _ := os.Stat(absPath)
	if localinfo != nil {
		l = NewLocalFile(absPath, localinfo)
	}

	var cl []*Change
	fmt.Println("Resolving...")
	if cl, err = g.resolveChangeListRecv(false, g.opts.Path, r, l); err != nil {
		return
	}

	if ok := printChangeList(cl, g.opts.IsNoPrompt); ok {
		return g.playPullChangeList(cl)
	}
	return
}

func (g *Commands) playPullChangeList(cl []*Change) (err error) {
	var next []*Change
	g.taskStart(len(cl))

	for {
		if len(cl) > maxNumOfConcPullTasks {
			next, cl = cl[:maxNumOfConcPullTasks], cl[maxNumOfConcPullTasks:len(cl)]
		} else {
			next, cl = cl, []*Change{}
		}
		if len(next) == 0 {
			break
		}
		var wg sync.WaitGroup
		wg.Add(len(next))
		// play the changes
		// TODO: add timeouts
		for _, c := range next {
			switch c.Op() {
			case OpMod:
				go g.localMod(&wg, c)
			case OpAdd:
				go g.localAdd(&wg, c)
			case OpDelete:
				go g.localDelete(&wg, c)
			}
		}
		wg.Wait()
	}

	g.taskFinish()
	return err
}

func (g *Commands) localMod(wg *sync.WaitGroup, change *Change) (err error) {
	defer g.taskDone()
	defer wg.Done()
	destAbsPath := g.context.AbsPathOf(change.Path)
	if change.Src.BlobAt != "" {
		// download and replace
		if err = g.download(change); err != nil {
			return
		}
	}
	return os.Chtimes(destAbsPath, change.Src.ModTime, change.Src.ModTime)
}

func (g *Commands) localAdd(wg *sync.WaitGroup, change *Change) (err error) {
	defer g.taskDone()
	defer wg.Done()
	destAbsPath := g.context.AbsPathOf(change.Path)
	// make parent's dir if not exists
	os.MkdirAll(filepath.Dir(destAbsPath), os.ModeDir|0755)
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

func (g *Commands) localDelete(wg *sync.WaitGroup, change *Change) (err error) {
	defer g.taskDone()
	defer wg.Done()
	return os.RemoveAll(change.Dest.BlobAt)
}

func (g *Commands) download(change *Change) (err error) {
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
