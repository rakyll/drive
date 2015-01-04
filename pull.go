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
	"sort"
	"strings"
	"sync"
)

const (
	maxNumOfConcPullTasks  = 4
	maxChkSumHeuristicSize = 50 * 1000 * 1000
)

// Pull from remote if remote path exists and in a god context. If path is a
// directory, it recursively pulls from the remote if there are remote changes.
// It doesn't check if there are remote changes if isForce is set.
func (g *Commands) Pull() (err error) {
	var cl []*Change
	for _, relToRootPath := range g.opts.Sources {
		fsPath := g.context.AbsPathOf(relToRootPath)
		ccl, cErr := g.changeListResolve(relToRootPath, fsPath, false)
		if cErr == nil && len(ccl) > 0 {
			cl = append(cl, ccl...)
		}
	}

	ok := printChangeList(cl, g.opts.NoPrompt, g.opts.NoClobber)
	if ok {
		return g.playPullChangeList(cl, g.opts.Exports)
	}

	return
}

func (g *Commands) playPullChangeList(cl []*Change, exports []string) (err error) {
	var next []*Change
	g.taskStart(len(cl))

	// TODO: Only provide precedence ordering if all the other options are allowed
	// Currently noop on sorting by precedence
	if false && !g.opts.NoClobber {
		sort.Sort(ByPrecedence(cl))
	}

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
			switch c.CoercedOp(g.opts.NoClobber) {
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

	// Simple heuristic to avoid downloading all the
	// content yet it could just be a modTime difference
	if !chksumSizeEqual(change.Src, change.Dest) {
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
	destAbsDir := g.context.AbsPathOf(change.Parent)
	os.MkdirAll(destAbsDir, os.ModeDir|0755)

	if err != nil {
		return
	}
	if change.Src.IsDir {
		return os.Mkdir(destAbsPath, os.ModeDir|0755)
	}

	// download and create
	if err = g.download(change, exports); err != nil {
		return
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
		if err == nil && ef != nil {
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
		mimeType = mimeTypeFromExt(ext)
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
			defer func() {
				wg.Done()
			}()

			err := g.singleDownload(dest, id, url)
			if err == nil {
				manifest = append(manifest, dest)
			}
			return err
		}(&wg, pathName, f.Id, exportURL)
	}
	wg.Wait()
	return
}

func isLocalFile(f *File) bool {
	// TODO: Better check
	return f != nil && f.Etag == ""
}

func chksumSizeEqual(a, b *File) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Size != b.Size || a.IsDir != b.IsDir {
		return false
	}
	if a.IsDir {
		return false
	}

	if isLocalFile(a) && a.Size > maxChkSumHeuristicSize {
		return false
	}
	if isLocalFile(b) && b.Size > maxChkSumHeuristicSize {
		return false
	}
	return md5Checksum(a) == md5Checksum(b)
}

func (g *Commands) download(change *Change, exports []string) (err error) {
	if change.Src == nil {
		return fmt.Errorf("Tried to download nil change.Src")
	}

	destAbsPath := g.context.AbsPathOf(change.Path)
	if change.Src.BlobAt != "" {
		return g.singleDownload(destAbsPath, change.Src.Id, "")
	}

	// We need to touch the empty file to
	// ensure consistency during a push.
	if err = touchFile(destAbsPath); err != nil {
		return err
	}

	if len(exports) >= 1 && hasExportLinks(change.Src) {
		exportDirPath := destAbsPath
		if g.opts.ExportsDir != "" {
			exportDirPath = path.Join(g.opts.ExportsDir, change.Src.Name)
		}

		manifest, exportErr := g.export(change.Src, exportDirPath, exports)
		if exportErr == nil {
			for _, exportPath := range manifest {
				fmt.Printf("Exported '%s' to '%s'\n", destAbsPath, exportPath)
			}
		}
		return exportErr
	}
	return
}

func (g *Commands) singleDownload(p, id, exportURL string) (err error) {
	var fo *os.File
	fo, err = os.Create(p)
	if err != nil {
		fmt.Println("create", err)
		return
	}

	// close fo on exit and check for its returned error
	defer func() {
		fErr := fo.Close()
		if fErr != nil {
			fmt.Println("fErr", fErr)
			err = fErr
		}
	}()

	var blob io.ReadCloser
	defer func() {
		if blob != nil {
			blob.Close()
		}
	}()

	blob, err = g.rem.Download(id, exportURL)
	if err != nil {
		return err
	}
	_, err = io.Copy(fo, blob)
	return
}
