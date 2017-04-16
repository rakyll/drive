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
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	diffmp "github.com/sergi/go-diff/diffmatchpatch"
)

// MaxFileSize is the max number of bytes we
// can accept for diffing (Arbitrary value)
const MaxFileSize = 70 * 1024 * 1024

func fsFileToString(abspath string) (buf string, err error) {
	var finfo os.FileInfo

	finfo, err = os.Stat(abspath)
	if err != nil {
		return
	}

	var fh *os.File
	fh, err = os.Open(abspath)
	defer func() {
		if fh != nil {
			fh.Close()
		}
	}()
	if err != nil {
		return
	}
	bbuf := make([]byte, finfo.Size())
	_, err = fh.Read(bbuf)
	if err != nil {
		return
	}
	buf = string(bbuf)
	return
}

func (g *Commands) Diff() (err error) {
	var relPath, absPath string
	relPath, absPath, err = g.pathResolve()
	if err != nil {
		return
	}
	var r, l *File

	r, err = g.rem.FindByPath(relPath)
	if err != nil || r == nil {
		return
	}
	var localinfo os.FileInfo
	localinfo, err = os.Stat(absPath)
	if err != nil || localinfo == nil {
		return
	}
	if localinfo != nil {
		l = NewLocalFile(absPath, localinfo)
	}

	// Pre-screening phase
	if r.IsDir {
		if l.IsDir {
			fmt.Println("Both local and remote are directories")
		} else {
			fmt.Println("Remote is a directory while local is an ordinary file")
		}
		return
	}
	if l.IsDir {
		if r.IsDir {
			fmt.Println("Both local and remote are directories")
		} else {
			fmt.Println("Local is a directory while remote is an ordinary file")
		}
		return
	}

	if r.BlobAt == "" {
		return fmt.Errorf("Could not find remote: '%v'", r.Name)
	}
	if isSameFile(r, l) {
		// No output when "no changes found"
		return nil
	}

	if r.Size > MaxFileSize {
		return fmt.Errorf(
			"Remote too large for display \033[94m[%v bytes]\033[00m", r.Size)
	}
	if l.Size > MaxFileSize {
		return fmt.Errorf(
			"Local too large for display \033[92m[%v bytes]\033[00m", l.Size)
	}

	var frTmp, fl *os.File
	var blob io.ReadCloser

	// Clean-up
	defer func() {
		if frTmp != nil {
			os.RemoveAll(frTmp.Name())
		}
		if fl != nil {
			fl.Close()
		}
		if blob != nil {
			blob.Close()
		}
	}()

	blob, err = g.rem.Download(r.Id)
	if err != nil {
		return err
	}

	// Next step: Create a temp file with an obscure name unlikely to clash.
	tmpName := strings.Join([]string{
		".",
		fmt.Sprintf("tmp%v.tmp", rand.Int()),
	}, "x")

	frTmp, err = ioutil.TempFile(".", tmpName)
	if err != nil {
		return
	}
	_, err = io.Copy(frTmp, blob)
	if err != nil {
		return
	}
	var rBuffer, lBuffer string
	rBuffer, err = fsFileToString(frTmp.Name())
	if err != nil {
		return
	}
	lBuffer, err = fsFileToString(l.BlobAt)
	if err != nil {
		return
	}

	dmp := diffmp.New()
	patches := dmp.PatchMake(lBuffer, rBuffer)
	fmt.Print(dmp.PatchToText(patches))
	return
}
