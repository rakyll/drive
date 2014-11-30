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
	if r, err = g.rem.FindByPath(relPath); err != nil {
		// We cannot diff a non-existant remote
		fmt.Println("Local is a new file")
        return
	}
	localinfo, err  := os.Stat(absPath)
	if localinfo != nil {
		l = NewLocalFile(absPath, localinfo)
	}
	if isSameFile(r, l) {
		fmt.Println("Everything is up-to-date.")
		return nil
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
	tmpName := strings.Join([]string {
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

	diffo := diffmp.New()
	patches := diffo.PatchMake(lBuffer, rBuffer)
	fmt.Println("diffp", diffo.PatchToText(patches))

	return
}
