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

	drive "github.com/odeke-em/google-api-go-client/drive/v2"
)

const (
	OpNone = iota
	OpAdd
	OpDelete
	OpMod
)

const (
	DifferNone = 1 << iota
	DifferDirType
	DifferMd5Checksum
	DifferModTime
	DifferSize
)

const (
	DriveFolderMimeType = "application/vnd.google-apps.folder"
)

// Arbitrary value. TODO: Get better definition of BigFileSize.
var BigFileSize = int64(1024 * 1024 * 400)

var opPrecedence = map[int]int{
	OpNone:   0,
	OpDelete: 1,
	OpAdd:    2,
	OpMod:    3,
}

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
	Etag        string
	Shared      bool
	// Copyable decides if the user has allowed for the file to be copied
	Copyable bool
	// UserPermission contains the permissions for the authenticated user on this file
	UserPermission *drive.Permission
	// CacheChecksum when set avoids recomputation of checksums
	CacheChecksum bool
	// Monotonically increasing version number for the file
	Version int64
	// The onwers of this file.
	OwnerNames []string
	// Permissions contains the overall permissions for this file
	Permissions []*drive.Permission
}

func NewRemoteFile(f *drive.File) *File {
	mtime, _ := time.Parse("2006-01-02T15:04:05.000Z", f.ModifiedDate)
	mtime = mtime.Round(time.Second)
	return &File{
		BlobAt:      f.DownloadUrl,
		Etag:        f.Etag,
		ExportLinks: f.ExportLinks,
		Id:          f.Id,
		IsDir:       f.MimeType == DriveFolderMimeType,
		Md5Checksum: f.Md5Checksum,
		MimeType:    f.MimeType,
		ModTime:     mtime,
		Copyable:    f.Copyable,
		// We must convert each title to match that on the FS.
		Name:           urlToPath(f.Title, true),
		Size:           f.FileSize,
		Shared:         f.Shared,
		UserPermission: f.UserPermission,
		Version:        f.Version,
		OwnerNames:     f.OwnerNames,
		Permissions:    f.Permissions,
	}
}

func DupFile(f *File) *File {
	return &File{
		BlobAt:      f.BlobAt,
		Etag:        f.Etag,
		ExportLinks: f.ExportLinks,
		Id:          f.Id,
		IsDir:       f.IsDir,
		Md5Checksum: f.Md5Checksum,
		MimeType:    f.MimeType,
		ModTime:     f.ModTime,
		Copyable:    f.Copyable,
		// We must convert each title to match that on the FS.
		Name:           f.Name,
		Size:           f.Size,
		Shared:         f.Shared,
		UserPermission: f.UserPermission,
		Version:        f.Version,
		OwnerNames:     f.OwnerNames,
		Permissions:    f.Permissions,
	}
}

func NewLocalFile(absPath string, f os.FileInfo) *File {
	return &File{
		Id:      "",
		Name:    f.Name(),
		ModTime: f.ModTime().Round(time.Second),
		IsDir:   f.IsDir(),
		Size:    f.Size(),
		BlobAt:  absPath,
		// TODO: Read the CacheChecksum toggle dynamically if set
		// by the requester ie if the file is rapidly changing.
		CacheChecksum: true,
	}
}

type Change struct {
	Dest           *File
	Parent         string
	Path           string
	Src            *File
	Force          bool
	NoClobber      bool
	IgnoreChecksum bool
}

type ByPrecedence []*Change

func (cl ByPrecedence) Less(i, j int) bool {
	if cl[i] == nil {
		return false
	}
	if cl[j] == nil {
		return true
	}

	rank1, rank2 := opPrecedence[cl[i].Op()], opPrecedence[cl[j].Op()]
	return rank1 < rank2
}

func (cl ByPrecedence) Len() int {
	return len(cl)
}

func (cl ByPrecedence) Swap(i, j int) {
	cl[i], cl[j] = cl[j], cl[i]
}

func (self *File) sameDirType(other *File) bool {
	return other != nil && self.IsDir == other.IsDir
}

func opToString(op int) (string, string) {
	switch op {
	case OpAdd:
		return "\033[32m+\033[0m", "Addition"
	case OpDelete:
		return "\033[31m-\033[0m", "Deletion"
	case OpMod:
		return "\033[33mM\033[0m", "Modification"
	default:
		return "", ""
	}
}

func (f *File) largeFile() bool {
	return f.Size > BigFileSize
}

func (c *Change) Symbol() string {
	symbol, _ := opToString(c.Op())
	return symbol
}

func md5Checksum(f *File) string {
	if f == nil || f.IsDir {
		return ""
	}
	if f.Md5Checksum != "" {
		return f.Md5Checksum
	}

	if f.largeFile() { // Just warn the user in case of impatience.
		// TODO: Only turn on warnings if verbosity is set.
		fmt.Printf("\033[91mmd5Checksum\033[00m: `%s` (%v)\nmight take time to checksum.\n",
			f.Name, prettyBytes(f.Size))
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
	checksum := fmt.Sprintf("%x", h.Sum(nil))
	if f.CacheChecksum {
		// fmt.Println("CACHING CHECKSUM", checksum, f.Name)
		f.Md5Checksum = checksum
	}
	return checksum
}

func checksumDiffers(mask int) bool {
	return (mask & DifferMd5Checksum) != 0
}

func dirTypeDiffers(mask int) bool {
	return (mask & DifferDirType) != 0
}

func modTimeDiffers(mask int) bool {
	return (mask & DifferModTime) != 0
}

func sizeDiffers(mask int) bool {
	return (mask & DifferSize) != 0
}

func fileDifferences(src, dest *File, ignoreChecksum bool) int {
	if src == nil || dest == nil {
		return DifferMd5Checksum | DifferSize | DifferModTime | DifferDirType
	}

	difference := DifferNone
	if src.Size != dest.Size {
		difference |= DifferSize
	}
	if !src.ModTime.Equal(dest.ModTime) {
		difference |= DifferModTime
	}
	if src.IsDir != dest.IsDir {
		difference |= DifferDirType
	}

	if !ignoreChecksum {
		// Only compute the checksum if the size is the same.
		if sizeDiffers(difference) || md5Checksum(src) != md5Checksum(dest) {
			difference |= DifferMd5Checksum
		}
	}
	return difference
}

// If the preliminary sameFile test passes,
// then perform an Md5 checksum comparison
func sameFileTillChecksum(src, dest *File, ignoreChecksum bool) bool {
	return fileDifferences(src, dest, ignoreChecksum) == DifferNone
}

func (c *Change) op() int {
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

	if !c.Src.IsDir && !sameFileTillChecksum(c.Src, c.Dest, c.IgnoreChecksum) {
		return OpMod
	}
	return OpNone
}

func (c *Change) Op() int {
	if c.Force {
		return OpAdd
	}
	op := c.op()
	if op != OpAdd && c.NoClobber {
		return OpNone
	}
	return op
}
