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
	"fmt"
	"sync"
	"time"

	drive "github.com/odeke-em/google-api-go-client/drive/v2"
	"github.com/odeke-em/log"
)

type keyValue struct {
	key   string
	value interface{}
}

func (g *Commands) Stat() error {
	channelMap := make(map[int]chan *keyValue)
	var wg sync.WaitGroup
	wg.Add(len(g.opts.Sources))
	for i, relToRootPath := range g.opts.Sources {
		go func(id int, p string, chanMap *map[int]chan *keyValue, wgg *sync.WaitGroup) {
			defer wgg.Done()
			chMap := *chanMap

			file, err := g.rem.FindByPath(p)
			if err == nil {
				chMap[id] = g.stat(p, file)
				return
			}

			g.log.LogErrf("%s: %v\n", p, err)
			childChan := make(chan *keyValue)
			close(childChan)
			chMap[id] = childChan
			return
		}(i, relToRootPath, &channelMap, &wg)
	}
	wg.Wait()

	throttle := time.Tick(1e9 / 10)
	// Spin until all the channels are drained
	for {
		if len(channelMap) < 1 {
			break
		}

		for key, childChan := range channelMap {
			select {
			case v := <-childChan:
				if v == nil { // Closed
					delete(channelMap, key)
				} else if v.value != nil {
					err := v.value.(error)
					if err != nil {
						g.log.LogErrf("v: %s err: %v\n", v.key, err)
					}
				}
			default:
			}
		}

		// Pause for a bit
		<-throttle
	}
	return nil
}

func prettyPermission(logf log.Loggerf, perm *drive.Permission) {
	logf("\n*\nName: %v <%s>\n", perm.Name, perm.EmailAddress)
	kvList := []*keyValue{
		&keyValue{"Role", perm.Role},
		&keyValue{"AccountType", perm.Type},
	}
	for _, kv := range kvList {
		logf("%-20s %-30v\n", kv.key, kv.value.(string))
	}
	logf("*\n")
}

func prettyFileStat(logf log.Loggerf, relToRootPath string, file *File) {
	dirType := "file"
	if file.IsDir {
		dirType = "folder"
	}

	logf("\n\033[92m%s\033[00m\n", relToRootPath)
	kvList := []*keyValue{
		&keyValue{"FileId", file.Id},
		&keyValue{"Bytes", fmt.Sprintf("%v", file.Size)},
		&keyValue{"Size", prettyBytes(file.Size)},
		&keyValue{"DirType", dirType},
		&keyValue{"MimeType", file.MimeType},
		&keyValue{"Etag", file.Etag},
		&keyValue{"ModTime", fmt.Sprintf("%v", file.ModTime)},
	}
	if !file.IsDir {
		kvList = append(kvList, &keyValue{"Md5Checksum", file.Md5Checksum})
	}
	for _, kv := range kvList {
		logf("%-20s %-30v\n", kv.key, kv.value.(string))
	}
}

func (g *Commands) stat(relToRootPath string, file *File) chan *keyValue {
	statChan := make(chan *keyValue)

	// Arbitrary value for throttle pause duration
	throttle := time.Tick(1e9 / 5)
	go func() {
		kv := &keyValue{
			key: relToRootPath,
		}

		defer func() {
			statChan <- kv
			statChan <- nil
			close(statChan)
		}()

		prettyFileStat(g.log.Logf, relToRootPath, file)
		perms, permErr := g.rem.listPermissions(file.Id)
		if permErr != nil {
			kv.value = permErr
			return
		}

		for _, perm := range perms {
			prettyPermission(g.log.Logf, perm)
		}
		if !file.IsDir || !g.opts.Recursive {
			return
		}

		remoteChildren := g.rem.FindByParentId(file.Id, g.opts.Hidden)
		channelMap := make(map[int]chan *keyValue)
		i := 0
		for child := range remoteChildren {
			childChan := g.stat(relToRootPath+"/"+child.Name, child)
			<-throttle
			channelMap[i] = childChan
			i += 1
		}

		for {
			if len(channelMap) < 1 {
				break
			}

			for key, childChan := range channelMap {
				select {
				case v := <-childChan:
					if v == nil { // Closed
						delete(channelMap, key)
					} else {
						statChan <- v
					}
				default:
					<-throttle
				}
			}
			<-throttle
		}
	}()
	return statChan
}
