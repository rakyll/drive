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
	"path"
	"path/filepath"
	"strings"
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
		return g.playPullChangeList(cl, g.opts.Exports)
	}
	return
}

func (g *Commands) playPullChangeList(cl []*Change, exports []string) (err error) {
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
				go g.localMod(&wg, c, exports)
			case OpAdd:
				go g.localAdd(&wg, c, exports)
			case OpDelete:
				go g.localDelete(&wg, c)
			}
		}
		wg.Wait()
	}

	g.taskFinish()
	return err
}

func (g *Commands) localMod(wg *sync.WaitGroup, change *Change, exports []string) (err error) {
	defer g.taskDone()
	defer wg.Done()
	destAbsPath := g.context.AbsPathOf(change.Path)

	if change.Src.BlobAt != "" || change.Src.ExportLinks != nil {
		// download and replace
		if err = g.download(change, exports); err != nil {
			return
		}
	}
	return os.Chtimes(destAbsPath, change.Src.ModTime, change.Src.ModTime)
}

func (g *Commands) localAdd(wg *sync.WaitGroup, change *Change, exports []string) (err error) {

	defer g.taskDone()
	defer wg.Done()
	destAbsPath := g.context.AbsPathOf(change.Path)
	// make parent's dir if not exists
	os.MkdirAll(filepath.Dir(destAbsPath), os.ModeDir|0755)
	if change.Src.IsDir {
		return os.Mkdir(destAbsPath, os.ModeDir|0755)
	}
	if change.Src.BlobAt != "" || change.Src.ExportLinks != nil {
		// download and create
		if err = g.download(change, exports); err != nil {
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

func touchFile(path string) (err error) {
	var ef *os.File
	defer func() {
		if err != nil && ef != nil {
			ef.Close()
		}
	}()
	ef, err = os.Create(path)
	return
}

func (g *Commands) export(f *File, destAbsPath string, exports []string) (manifest []string, err error) {
	if len(exports) < 1 || f == nil {
		return
	}

	dirPath := strings.Join([]string{destAbsPath, "exports"}, "_")
	if err = os.MkdirAll(dirPath, os.ModeDir|0755); err != nil {
		return
	}

	var ok bool
	var mimeType, exportURL string

	waitables := map[string]string{}
	for _, ext := range exports {
		mimeType, ok = docExportsMap[ext]
		if !ok {
			continue
		}
		exportURL, ok = f.ExportLinks[mimeType]
		if !ok {
			continue
		}
		exportPath := strings.Join([]string{filepath.Base(f.Name), ext}, ".")
		pathName := path.Join(dirPath, exportPath)
		waitables[pathName] = exportURL
	}

	var wg sync.WaitGroup
	wg.Add(len(waitables))

	for pathName, exportURL := range waitables {
		go func(wg *sync.WaitGroup, dest, id, url string) error {
			var fo *os.File
			var blob io.ReadCloser
			var fErr, dlErr error

			defer func() {
				if blob != nil {
					blob.Close()
				}
				if fo != nil {
					fo.Close()
				}
				wg.Done()
			}()

			fo, fErr = os.Create(dest)
			if fErr != nil {
				return fErr
			}

			blob, dlErr = g.rem.Download(id, url)
			if dlErr != nil {
				return dlErr
			}
			_, err = io.Copy(fo, blob)
			if err == nil {
				manifest = append(manifest, dest)
			}
			return err
		}(&wg, pathName, f.Id, exportURL)
	}
	wg.Wait()
	return
}

func (g *Commands) download(change *Change, exports []string) (err error) {
	baseName := change.Path
	destAbsPath := g.context.AbsPathOf(baseName)

	if hasExportLinks(change.Src) {
		// We need to touch the empty file to ensure
		// consistency during a push.
		emptyFilepath := g.context.AbsPathOf(baseName)
		if err = touchFile(emptyFilepath); err != nil {
			return err
		}
		manifest, exportErr := g.export(change.Src, destAbsPath, exports)
		if exportErr == nil {
			for i, exportPath := range manifest {
				fmt.Printf("# %d: %s\n", i+1, exportPath)
			}
		}
		return exportErr
	}

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
	blob, err = g.rem.Download(change.Src.Id, "")
	if err != nil {
		return err
	}
	_, err = io.Copy(fo, blob)
	return
}
