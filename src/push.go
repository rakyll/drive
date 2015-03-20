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
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/odeke-em/drive/config"
)

// Pushes to remote if local path exists and in a gd context. If path is a
// directory, it recursively pushes to the remote if there are local changes.
// It doesn't check if there are local changes if isForce is set.
func (g *Commands) Push() (err error) {
	defer g.clearMountPoints()

	root := g.context.AbsPathOf("")
	var cl []*Change

	fmt.Println("Resolving...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	spin := newPlayable(10)
	spin.play()

	// To Ensure mount points are cleared in the event of external exceptios
	go func() {
		_ = <-c
		spin.stop()
		g.clearMountPoints()
		os.Exit(1)
	}()

	for _, relToRootPath := range g.opts.Sources {
		fsPath := g.context.AbsPathOf(relToRootPath)
		ccl, cErr := g.changeListResolve(relToRootPath, fsPath, true)
		if cErr != nil {
			spin.stop()
			return cErr
		}
		if len(ccl) > 0 {
			cl = append(cl, ccl...)
		}
	}

	mount := g.opts.Mount
	if mount != nil {
		for _, mt := range mount.Points {
			ccl, cerr := lonePush(g, root, mt.Name, mt.MountPath)
			if cerr == nil {
				cl = append(cl, ccl...)
			}
		}
	}

	spin.stop()

	nonConflictsPtr, conflictsPtr := g.resolveConflicts(cl)
	if conflictsPtr != nil {
		warnConflictsPersist(*conflictsPtr)
		return fmt.Errorf("conflicts have prevented a push")
	}

	nonConflicts := *nonConflictsPtr

	ok := printChangeList(nonConflicts, g.opts.NoPrompt, g.opts.NoClobber)
	if !ok {
		return
	}

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
	return g.playPushChangeList(nonConflicts)
}

func (g *Commands) resolveConflicts(cl []*Change) (*[]*Change, *[]*Change) {
	if g.opts.IgnoreConflict {
		return &cl, nil
	}

	nonConflicts, conflicts := sift(cl)
	resolved, unresolved := resolveConflicts(conflicts, true, g.deserializeIndex)
	if conflictsPersist(unresolved) {
		return &resolved, &unresolved
	}

	for _, ch := range unresolved {
		resolved = append(resolved, ch)
	}

	for _, ch := range resolved {
		nonConflicts = append(nonConflicts, ch)
	}
	return &nonConflicts, nil
}

func (g *Commands) PushPiped() (err error) {
	// Cannot push asynchronously because the push order must be maintained
	for _, relToRootPath := range g.opts.Sources {
		rem, resErr := g.rem.FindByPath(relToRootPath)
		if resErr != nil && resErr != ErrPathNotExists {
			return resErr
		}
		if rem != nil && !g.opts.Force {
			return fmt.Errorf("%s already exists remotely, use `%s` to override this behaviour.\n", relToRootPath, ForceKey)
		}

		if hasExportLinks(rem) {
			return fmt.Errorf("'%s' is a GoogleDoc/Sheet document cannot be pushed to raw.\n", relToRootPath)
		}

		base := filepath.Base(relToRootPath)
		local := fauxLocalFile(base)
		if rem == nil {
			rem = local
		}

		parentPath := g.parentPather(relToRootPath)
		parent, pErr := g.rem.FindByPath(parentPath)
		if pErr != nil {
			spin := newPlayable(10)
			spin.play()
			parent, pErr = g.remoteMkdirAll(parentPath)
			spin.stop()
			if pErr != nil || parent == nil {
				fmt.Printf("%s: %v", relToRootPath, pErr)
				return
			}
		}

		args := upsertOpt{
			parentId:       parent.Id,
			fsAbsPath:      relToRootPath,
			src:            rem,
			dest:           rem,
			mask:           g.opts.TypeMask,
			nonStatable:    true,
			ignoreChecksum: g.opts.IgnoreChecksum,
		}

		rem, rErr := g.rem.upsertByComparison(os.Stdin, &args)
		if rErr != nil {
			fmt.Printf("%s: %v\n", relToRootPath, rErr)
			return rErr
		}

		if rem == nil {
			continue
		}

		index := rem.ToIndex()
		wErr := g.context.SerializeIndex(index, g.context.AbsPathOf(""))

		// TODO: Should indexing errors be reported?
		if wErr != nil {
			fmt.Printf("serializeIndex %s: %v\n", rem.Name, wErr)
		}
	}
	return
}

func (g *Commands) deserializeIndex(identifier string) *config.Index {
	index, err := g.context.DeserializeIndex(g.context.AbsPathOf(""), identifier)
	if err != nil {
		return nil
	}
	return index
}

func (g *Commands) playPushChangeList(cl []*Change) (err error) {
	g.taskStart(len(cl))

	// TODO: Only provide precedence ordering if all the other options are allowed
	// Currently noop on sorting by precedence
	if false && !g.opts.NoClobber {
		sort.Sort(ByPrecedence(cl))
	}

	for _, c := range cl {
		switch c.Op() {
		case OpMod:
			g.remoteMod(c)
		case OpModConflict:
			g.remoteMod(c)
		case OpAdd:
			g.remoteAdd(c)
		case OpDelete:
			g.remoteDelete(c)
		}
	}

	// Time to organize them according branching
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

func (g *Commands) pathSplitter(absPath string) (dir, base string) {
	p := strings.Split(absPath, "/")
	pLen := len(p)
	base = p[pLen-1]
	p = append([]string{"/"}, p[:pLen-1]...)
	dir = gopath.Join(p...)
	return
}

func (g *Commands) parentPather(absPath string) string {
	dir, _ := g.pathSplitter(absPath)
	return dir
}

func (g *Commands) remoteMod(change *Change) (err error) {
	defer g.taskDone()

	if change.Dest == nil && change.Src == nil {
		err = fmt.Errorf("bug on: both dest and src cannot be nil")
		fmt.Println(err)
		return err
	}

	absPath := g.context.AbsPathOf(change.Path)
	var parent *File
	if change.Dest != nil && change.Src != nil {
		change.Src.Id = change.Dest.Id // TODO: bad hack
	}

	parentPath := g.parentPather(change.Path)
	parent, err = g.rem.FindByPath(parentPath)

	if err != nil {
		return err
	}

	args := upsertOpt{
		parentId:       parent.Id,
		fsAbsPath:      absPath,
		src:            change.Src,
		dest:           change.Dest,
		mask:           g.opts.TypeMask,
		ignoreChecksum: g.opts.IgnoreChecksum,
	}

	rem, err := g.rem.UpsertByComparison(&args)
	if err != nil {
		fmt.Printf("%s: %v\n", change.Path, err)
		return
	}
	if rem == nil {
		return
	}
	index := rem.ToIndex()
	wErr := g.context.SerializeIndex(index, g.context.AbsPathOf(""))

	// TODO: Should indexing errors be reported?
	if wErr != nil {
		fmt.Printf("serializeIndex %s: %v\n", rem.Name, wErr)
	}
	return
}

func (g *Commands) remoteAdd(change *Change) (err error) {
	return g.remoteMod(change)
}

func (g *Commands) indexAbsPath(fileId string) string {
	return config.IndicesAbsPath(g.context.AbsPathOf(""), fileId)
}

func (g *Commands) remoteUntrash(change *Change) (err error) {
	defer g.taskDone()

	return g.rem.Untrash(change.Src.Id)
}

func (g *Commands) remoteDelete(change *Change) (err error) {
	defer g.taskDone()

	err = g.rem.Trash(change.Dest.Id)
	if err != nil {
		return
	}

	indexPath := g.indexAbsPath(change.Dest.Id)
	if rmErr := os.Remove(indexPath); rmErr != nil {
		fmt.Printf("%s \"%s\": remove indexfile %v\n", change.Path, change.Dest.Id, rmErr)
	}
	return
}

func (g *Commands) remoteMkdirAll(d string) (file *File, err error) {
	// Try the lookup one last time in case a coroutine raced us to it.
	retrFile, retryErr := g.rem.FindByPath(d)
	if retryErr == nil && retrFile != nil {
		return retrFile, nil
	}

	rest, last := remotePathSplit(d)

	parent, parentErr := g.rem.FindByPath(rest)
	if parentErr != nil && parentErr != ErrPathNotExists {
		return parent, parentErr
	}

	if parent == nil {
		parent, parentErr = g.remoteMkdirAll(rest)
		if parentErr != nil || parent == nil {
			return parent, parentErr
		}
	}

	remoteFile := &File{
		IsDir:   true,
		Name:    last,
		ModTime: time.Now(),
	}

	args := upsertOpt{
		parentId: parent.Id,
		src:      remoteFile,
	}
	parent, parentErr = g.rem.UpsertByComparison(&args)
	if parentErr == nil && parent != nil {
		index := parent.ToIndex()
		wErr := g.context.SerializeIndex(index, g.context.AbsPathOf(""))

		// TODO: Should indexing errors be reported?
		if wErr != nil {
			fmt.Printf("serializeIndex %s: %v\n", parent.Name, wErr)
		}
	}
	return parent, parentErr
}

func list(context *config.Context, p string, hidden bool, ignore *regexp.Regexp) (fileChan chan *File, err error) {
	absPath := context.AbsPathOf(p)
	var f []os.FileInfo
	f, err = ioutil.ReadDir(absPath)
	fileChan = make(chan *File)
	if err != nil {
		close(fileChan)
		return
	}

	go func() {
		for _, file := range f {
			if file.Name() == config.GDDirSuffix {
				continue
			}
			if ignore != nil && ignore.Match([]byte(file.Name())) {
				continue
			}
			if !isHidden(file.Name(), hidden) {
				fileChan <- NewLocalFile(gopath.Join(absPath, file.Name()), file)
			}

			symlink := (file.Mode() & os.ModeSymlink) != 0
			if symlink {
				symAbsPath := gopath.Join(absPath, file.Name())
				var symResolvPath string
				symResolvPath, err = filepath.EvalSymlinks(symAbsPath)
				if err != nil {
					continue
				}

				var symInfo os.FileInfo
				symInfo, err = os.Stat(symResolvPath)
				if err != nil {
					continue
				}
				fileChan <- NewLocalFile(symAbsPath, symInfo)
			}
		}
		close(fileChan)
	}()
	return
}
