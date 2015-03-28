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
	"runtime"
	"sort"
	"sync"
)

const (
	maxNumOfConcPullTasks = 4
)

type urlMimeTypeExt struct {
	ext      string
	mimeType string
	url      string
}

// Pull from remote if remote path exists and in a god context. If path is a
// directory, it recursively pulls from the remote if there are remote changes.
// It doesn't check if there are remote changes if isForce is set.
func (g *Commands) Pull() (err error) {
	var cl []*Change

	g.log.Logln("Resolving...")

	spin := g.playabler()
	spin.play()

	for _, relToRootPath := range g.opts.Sources {
		fsPath := g.context.AbsPathOf(relToRootPath)
		ccl, cErr := g.changeListResolve(relToRootPath, fsPath, false)
		if cErr != nil {
			return cErr
		}
		if len(ccl) > 0 {
			cl = append(cl, ccl...)
		}
	}

	spin.stop()

	nonConflictsPtr, conflictsPtr := g.resolveConflicts(cl, false)
	if conflictsPtr != nil {
		warnConflictsPersist(g.log, *conflictsPtr)
		return fmt.Errorf("conflicts have prevented a pull operation")
	}

	nonConflicts := *nonConflictsPtr

	ok := printChangeList(g.log, nonConflicts, !g.opts.canPrompt(), g.opts.NoClobber)
	if !ok {
		return
	}

	return g.playPullChangeList(nonConflicts, g.opts.Exports)
}

func (g *Commands) PullPiped() (err error) {
	// Cannot pull asynchronously because the pull order must be maintained
	for _, relToRootPath := range g.opts.Sources {
		rem, err := g.rem.FindByPath(relToRootPath)
		if err != nil {
			return fmt.Errorf("%s: %v", relToRootPath, err)
		}
		if rem == nil {
			continue
		}

		if hasExportLinks(rem) {
			g.log.LogErrf("'%s' is a GoogleDoc/Sheet document cannot be pulled from raw, only exported.\n", relToRootPath)
			continue
		}
		blobHandle, dlErr := g.rem.Download(rem.Id, "")
		if dlErr != nil {
			return dlErr
		}
		if blobHandle == nil {
			continue
		}
		_, err = io.Copy(os.Stdout, blobHandle)
		blobHandle.Close()
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
			switch c.Op() {
			case OpMod:
				go g.localMod(&wg, c, exports)
			case OpModConflict:
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
	defer func() {
		if err == nil {
			src := change.Src
			index := src.ToIndex()
			wErr := g.context.SerializeIndex(index, g.context.AbsPathOf(""))

			// TODO: Should indexing errors be reported?
			if wErr != nil {
				g.log.LogErrf("serializeIndex %s: %v\n", src.Name, wErr)
			}
		}
		g.taskDone()
		wg.Done()
	}()

	destAbsPath := g.context.AbsPathOf(change.Path)

	// Simple heuristic to avoid downloading all the
	// content yet it could just be a modTime difference
	mask := fileDifferences(change.Src, change.Dest, change.IgnoreChecksum)
	if checksumDiffers(mask) {
		// download and replace
		if err = g.download(change, exports); err != nil {
			return
		}
	}
	err = os.Chtimes(destAbsPath, change.Src.ModTime, change.Src.ModTime)
	return
}

func (g *Commands) localAdd(wg *sync.WaitGroup, change *Change, exports []string) (err error) {
	defer func() {
		if err == nil {
			src := change.Src
			index := src.ToIndex()
			sErr := g.context.SerializeIndex(index, g.context.AbsPathOf(""))

			// TODO: Should indexing errors be reported?
			if sErr != nil {
				g.log.LogErrf("serializeIndex %s: %v\n", src.Name, sErr)
			}
		}
		g.taskDone()
		wg.Done()
	}()

	destAbsPath := g.context.AbsPathOf(change.Path)

	// make parent's dir if not exists
	destAbsDir := g.context.AbsPathOf(change.Parent)

	if destAbsDir != destAbsPath {
		err = os.MkdirAll(destAbsDir, os.ModeDir|0755)
		if err != nil {
			return err
		}
	}

	if change.Src.IsDir {
		return os.Mkdir(destAbsPath, os.ModeDir|0755)
	}

	// download and create
	if err = g.download(change, exports); err != nil {
		return
	}

	err = os.Chtimes(destAbsPath, change.Src.ModTime, change.Src.ModTime)
	return
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

	dirPath := sepJoin("_", destAbsPath, "exports")
	if err = os.MkdirAll(dirPath, os.ModeDir|0755); err != nil {
		return
	}

	var ok bool
	var mimeType, exportURL string

	waitables := []*urlMimeTypeExt{}

	for _, ext := range exports {
		mimeType = mimeTypeFromExt(ext)
		exportURL, ok = f.ExportLinks[mimeType]
		if !ok {
			continue
		}

		waitables = append(waitables, &urlMimeTypeExt{
			mimeType: mimeType,
			url:      exportURL,
			ext:      ext,
		})
	}

	var wg sync.WaitGroup
	wg.Add(len(waitables))

	basePath := filepath.Base(f.Name)
	baseDir := path.Join(dirPath, basePath)

	for _, exportee := range waitables {
		go func(wg *sync.WaitGroup, baseDirPath, id string, urlMExt *urlMimeTypeExt) error {
			defer func() {
				wg.Done()
			}()

			exportPath := sepJoin(".", baseDirPath, urlMExt.ext)

			// TODO: Decide if users should get to make *.desktop users even for exports
			if runtime.GOOS == OSLinuxKey && false {
				desktopEntryPath := sepJoin(".", exportPath, "desktop")

				_, dentErr := f.serializeAsDesktopEntry(desktopEntryPath, urlMExt)
				if dentErr != nil {
					g.log.LogErrf("desktopEntry: %s %v\n", desktopEntryPath, dentErr)
				}
			}

			err := g.singleDownload(exportPath, id, urlMExt.url)
			if err == nil {
				manifest = append(manifest, exportPath)
			}
			return err
		}(&wg, baseDir, f.Id, exportee)
	}
	wg.Wait()
	return
}

func isLocalFile(f *File) bool {
	// TODO: Better check
	return f != nil && f.Etag == ""
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
	if runtime.GOOS != OSLinuxKey {
		err = touchFile(destAbsPath)
		if err != nil {
			return err
		}
	} else {
		// For those our Linux kin that need .desktop files
		dirPath := g.opts.ExportsDir
		if dirPath == "" {
			dirPath = filepath.Dir(destAbsPath)
		}

		f := change.Src

		urlMExt := urlMimeTypeExt{
			url:      f.AlternateLink,
			ext:      "",
			mimeType: f.MimeType,
		}

		dirPath = filepath.Join(dirPath, f.Name)
		desktopEntryPath := sepJoin(".", dirPath)

		_, dentErr := f.serializeAsDesktopEntry(desktopEntryPath, &urlMExt)
		if dentErr != nil {
			g.log.LogErrf("desktopEntry: %s %v\n", desktopEntryPath, dentErr)
		}
	}

	if len(exports) >= 1 && hasExportLinks(change.Src) {
		exportDirPath := destAbsPath
		if g.opts.ExportsDir != "" {
			exportDirPath = path.Join(g.opts.ExportsDir, change.Src.Name)
		}

		manifest, exportErr := g.export(change.Src, exportDirPath, exports)
		if exportErr == nil {
			for _, exportPath := range manifest {
				g.log.Logf("Exported '%s' to '%s'\n", destAbsPath, exportPath)
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
		g.log.LogErrf("create: %s %v\n", p, err)
		return
	}

	// close fo on exit and check for its returned error
	defer func() {
		fErr := fo.Close()
		if fErr != nil {
			g.log.LogErrf("fErr", fErr)
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
