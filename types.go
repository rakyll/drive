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
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"time"

	drive "code.google.com/p/google-api-go-client/drive/v2"
)

const (
	OpNone = iota
	OpAdd
	OpDelete
	OpMod
)

type File struct {
	BlobAt      string
	ExportLinks map[string]string
	Id          string
	IsDir       bool
	Md5Checksum string
	MimeType    string
	ModTime     time.Time
	Name        string
	Size        int64
}

func NewRemoteFile(f *drive.File) *File {
	mtime, _ := time.Parse("2006-01-02T15:04:05.000Z", f.ModifiedDate)
	mtime = mtime.Round(time.Second)
	return &File{
		BlobAt:      f.DownloadUrl,
		ExportLinks: f.ExportLinks,
		Id:          f.Id,
		IsDir:       f.MimeType == "application/vnd.google-apps.folder",
		Md5Checksum: f.Md5Checksum,
		MimeType:    f.MimeType,
		ModTime:     mtime,
		Name:        f.Title,
		Size:        f.FileSize,
	}
}

func NewLocalFile(absPath string, f os.FileInfo) *File {
	return &File{
		Id:      "",
		Name:    f.Name(),
		ModTime: f.ModTime(),
		IsDir:   f.IsDir(),
		Size:    f.Size(),
		BlobAt:  absPath,
	}
}

type Change struct {
	Path string
	Src  *File
	Dest *File
}

func (c *Change) Symbol() string {
	op := c.Op()
	switch op {
	case OpAdd:
		return "\x1b[32m+\x1b[0m"
	case OpDelete:
		return "\x1b[31m-\x1b[0m"
	case OpMod:
		return "\x1b[33mM\x1b[0m"
	default:
		return ""
	}
}

func md5Checksum(f *File) string {
	if f == nil || f.IsDir {
		return ""
	}
	if f.Md5Checksum != "" {
		return f.Md5Checksum
	}

	fh, err := os.Open(f.BlobAt)

	if err != nil {
		return ""
	}
	defer fh.Close()

	h := md5.New()
	_, err = io.Copy(h, fh)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// if it's a regular file, see it it's modified.
// If the first test passes, then do an Md5 checksum comparison
func isSameFile(src, dest *File) bool {
	if src.Size != dest.Size || !src.ModTime.Equal(dest.ModTime) {
		return false
	}

	ssum := md5Checksum(src)
	dsum := md5Checksum(dest)

	if dsum != ssum {
		return false
	}
	return true
}

// Will turn any other op but an Addition into a noop
func (c *Change) CoercedOp(noClobber bool) int {
	op := c.Op()
	if op != OpAdd && noClobber {
		return OpNone
	}
	return op
}

func (c *Change) Op() int {
	if c.Src == nil && c.Dest == nil {
		return OpNone
	}
	if c.Src != nil && c.Dest == nil {
		return OpAdd
	}
	if c.Src == nil && c.Dest != nil {
		return OpDelete
	}
	if c.Src.IsDir != c.Dest.IsDir {
		return OpMod
	}

	if !c.Src.IsDir {
		// if it's a regular file, see it it's modified.
		// If the first test passes then do an Md5 checksum comparison

		if c.Src.Size != c.Dest.Size || !c.Src.ModTime.Equal(c.Dest.ModTime) {
			return OpMod
		}

		ssum := md5Checksum(c.Src)
		dsum := md5Checksum(c.Dest)

		if dsum != ssum {
			return OpMod
		}
	}
	return OpNone
}
