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
	drive "github.com/google/google-api-go-client/drive/v2"
	"sync"
	"time"
)

type keyValue struct {
	path string
	err  error
}

func (g *Commands) Stat() error {
	channelMap := make(map[int]chan *keyValue)
	var wg sync.WaitGroup
	wg.Add(len(g.opts.Sources))
	for i, relToRootPath := range g.opts.Sources {
		go func(id int, p string, chanMap *map[int]chan *keyValue, wgg *sync.WaitGroup) {
			defer wgg.Done()
			chMap := *chanMap

			file, err := g.rem.FindByPath(relToRootPath)
			if err == nil {
				chMap[id] = g.stat(p, file)
				return
			}

			fmt.Printf("%s: %v\n", p, err)
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
				} else {
					if v.err != nil {
						fmt.Printf("v: %s err: %v\n", v.path, v.err)
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

func prettyPermission(perm *drive.Permission) {
	fmt.Printf("\n*\nName: %v <%s>\nRole: %v\nAccountType: %v\n*\n", perm.Name, perm.EmailAddress, perm.Role, perm.Type)
}

func (g *Commands) stat(relToRootPath string, file *File) chan *keyValue {
	statChan := make(chan *keyValue)

	// Arbitrary value for throttle pause duration
	throttle := time.Tick(1e9 / 5)
	go func() {
		kv := &keyValue{
			path: relToRootPath,
		}

		defer func() {
			statChan <- kv
			statChan <- nil
			close(statChan)
		}()

		perms, permErr := g.rem.listPermissions(file.Id)
		if permErr != nil {
			kv.err = permErr
			return
		}

		fmt.Printf("\n\033[92m%s\033[00m\nFileId: %s\nSize: %v\n", relToRootPath, file.Id, prettyBytes(file.Size))
		for _, perm := range perms {
			prettyPermission(perm)
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
