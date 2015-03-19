// Copyright 2015 Google Inc. All Rights Reserved.
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
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	MimeTypeJoiner = "-"
)

type desktopEntry struct {
	name string
	url  string
	icon string
}

func sepJoin(sep string, args ...string) string {
	return strings.Join(args, sep)
}

func (f *File) toDesktopEntry(urlMExt *urlMimeTypeExt) *desktopEntry {
	name := f.Name
	if urlMExt.ext != "" {
		name = sepJoin("-", f.Name, urlMExt.ext)
	}
	return &desktopEntry{
		name: name,
		url:  urlMExt.url,
		icon: urlMExt.mimeType,
	}
}

func (f *File) serializeAsDesktopEntry(destPath string, urlMExt *urlMimeTypeExt) (int, error) {
	deskEnt := f.toDesktopEntry(urlMExt)
	handle, err := os.Create(destPath)
	if err != nil {
		return 0, err
	}
	defer handle.Close()
	icon := strings.Replace(deskEnt.icon, UnescapedPathSep, MimeTypeJoiner, -1)

	return fmt.Fprintf(handle, "[Desktop Entry]\nIcon=%s\nName=%s\nType=%s\nURL=%s\n",
		icon, deskEnt.name, LinkKey, deskEnt.url)
}

func remotePathSplit(p string) (dir, base string) {
	// Avoiding use of filepath.Split because of bug with trailing "/" not being stripped
	sp := strings.Split(p, "/")
	spl := len(sp)
	dirL, baseL := sp[:spl-1], sp[spl-1:]
	dir = strings.Join(dirL, "/")
	base = strings.Join(baseL, "/")
	return
}

func commonPrefix(values ...string) string {
	vLen := len(values)
	if vLen < 1 {
		return ""
	}
	minIndex := 0
	min := values[0]
	minLen := len(min)

	for i := 1; i < vLen; i += 1 {
		st := values[i]
		if st == "" {
			return ""
		}
		lst := len(st)
		if lst < minLen {
			min = st
			minLen = lst
			minIndex = i + 0
		}
	}

	prefix := make([]byte, minLen)
	matchOn := true
	for i := 0; i < minLen; i += 1 {
		for j, other := range values {
			if minIndex == j {
				continue
			}
			if other[i] != min[i] {
				matchOn = false
				break
			}
		}
		if !matchOn {
			break
		}
		prefix[i] = min[i]
	}
	return string(prefix)
}

func readCommentedFile(p, comment string) (clauses []string, err error) {
	f, fErr := os.Open(p)
	if fErr != nil || f == nil {
		err = fErr
		return
	}

	defer f.Close()
	scanner := bufio.NewScanner(f)

	for {
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		line = strings.Trim(line, " ")
		line = strings.Trim(line, "\n")
		if strings.HasPrefix(line, comment) || len(line) < 1 {
			continue
		}
		clauses = append(clauses, line)
	}
	return
}
