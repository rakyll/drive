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
)

func (g *Commands) Trash() (err error) {
	for _, relToRoot := range g.opts.Sources {
		tErr := g.trash(relToRoot)
		if tErr != nil {
			fmt.Printf("\033[91mFailed to trash %s: %v\033[00m\n", relToRoot, tErr)
		} else {
			fmt.Printf("%s successfully trashed\n", relToRoot)
		}
	}
	return
}

func (g *Commands) trash(relToRoot string) error {
	file, err := g.rem.FindByPath(relToRoot)
	if err != nil {
		return err
	}
	return g.rem.Trash(file.Id)
}

func (g *Commands) Untrash() (err error) {
	for _, relToRoot := range g.opts.Sources {
		uErr := g.untrash(relToRoot)
		if uErr != nil {
			fmt.Printf("\033[91mFailed to untrash: '%s': %v\033[00m\n", relToRoot, uErr)
		} else {
			fmt.Printf("\033[92mSuccessfully untrashed: '%s'\033[00m\n", relToRoot)
		}
	}
	return
}

func (g *Commands) untrash(relToRoot string) error {
	file, err := g.rem.FindByPathTrashed(relToRoot)
	if err != nil {
		return err
	}
	return g.rem.Untrash(file.Id)
}

func (g *Commands) playTrashChangeList(cl []*Change, trashed bool) (err error) {
	var next []*Change
	g.taskStart(len(cl))

	var f = g.remoteDelete
	if trashed {
		f = g.remoteUntrash
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
			if c.Op() != OpNone {
				go func() error {
					defer wg.Done()
					return f(c)
				}()
			}
		}
		wg.Wait()
	}

	g.taskFinish()
	return err
}
