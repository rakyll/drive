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
	"os"
	"strings"
	"time"

	"github.com/aryann/difflib"
)

func readIntoBuffer(abspath string) (buf string, err error) {
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
	n := 0
	n, err = fh.Read(bbuf)
	if err != nil {
		return
	}
	buf = string(bbuf[:n])
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

	// Next step will be to a temp file with an obscure name unlikely to clash.
	tmpName := strings.Join([]string {
		".",
		fmt.Sprintf("tmp%vtmp", time.Now()),
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
	rBuffer, err = readIntoBuffer(frTmp.Name())
	if err != nil {
		return
	}
	lBuffer, err = readIntoBuffer(l.BlobAt)
	if err != nil {
		return
	}

	diff := difflib.Diff([]string{lBuffer}, []string{rBuffer})

	// In an ideal world, they should already be merged.
	// Otherwise we could use the clause below.
	ldiff, rdiff := diff[0], diff[1]
	if ldiff.Payload != rdiff.Payload {
		fmt.Println(ldiff, rdiff)
	}
    
	/*
	// Expecting one array with two elements, local & remote
	localChanges := diff[0]
	remoteChanges := diff[1]
	var i, j uint64
	llen, rlen := len(localChanges), len(remoteChanges)
	for i = j = 0; i < llen; i++ {
		lline, rline := "", ""
		if j < rlen {
			rline = remoteChanges[j]
			j++
		}
		lline = localChanges[i]
		if lline != rline {
			fmt.Println(lline)
			fmt.Println(rline)
		}
	}

	for i = i; i < llen; i++ {
		fmt.Print(localChanges[i])
	}
	for j = j; ir < rlen; j++ {
		fmt.Print(remoteChanges[j])
	}
	*/

	return
}
