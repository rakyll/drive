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
	"strings"
	"sync"
)

func (g *Commands) Trash() (err error) {
	return g.reduce(g.opts.Sources, true)
}

func (g *Commands) Untrash() (err error) {
	return g.reduce(g.opts.Sources, false)
}

func (g *Commands) EmptyTrash() error {
	if !g.breadthFirst("", "", "", -1, 0, true) {
		return nil
	}

	if !g.opts.NoPrompt {
		fmt.Println("Empty trash: (Yn)? ")

		input := "Y"
		fmt.Print("Proceed with the changes? [Y/n]: ")
		fmt.Scanln(&input)

		if strings.ToUpper(input) != "Y" {
			fmt.Println("Aborted emptying trash")
			return nil
		}
	}

	err := g.rem.EmptyTrash()
	if err == nil {
		fmt.Println("Successfully emptied trash")
	}
	return err
}

func (g *Commands) trasher(relToRoot string, toTrash bool) (change *Change, err error) {
	var file *File
	if relToRoot == "/" && toTrash {
		return nil, fmt.Errorf("Will not try to trash root.")
	}
	if toTrash {
		file, err = g.rem.FindByPath(relToRoot)
	} else {
		file, err = g.rem.FindByPathTrashed(relToRoot)
	}

	if err != nil {
		return
	}

	change = &Change{Path: relToRoot}
	if toTrash {
		change.Dest = file
	} else {
		change.Src = file
	}
	return
}

func (g *Commands) reduce(args []string, toTrash bool) error {
	var cl []*Change
	for _, relToRoot := range args {
		c, cErr := g.trasher(relToRoot, toTrash)
		if cErr != nil {
			fmt.Printf("\033[91m'%s': %v\033[00m\n", relToRoot, cErr)
		} else if c != nil {
			cl = append(cl, c)
		}
	}

	ok := printChangeList(cl, g.opts.NoPrompt, false)
	if ok {
		return g.playTrashChangeList(cl, toTrash)
	}
	return nil
}

func (g *Commands) playTrashChangeList(cl []*Change, toTrash bool) (err error) {
	var next []*Change
	g.taskStart(len(cl))

	var f = g.remoteUntrash
	if toTrash {
		f = g.remoteDelete
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
