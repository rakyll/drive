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
)

func (g *Commands) Trash() (err error) {
	return g.reduce(g.opts.Sources, true)
}

func (g *Commands) Untrash() (err error) {
	return g.reduce(g.opts.Sources, false)
}

func (g *Commands) EmptyTrash() error {
	rootFile, err := g.rem.FindByPath("/")
	if err != nil {
		return err
	}

	spin := newPlayable(10)
	spin.play()
	defer spin.stop()

	if !g.breadthFirst(rootFile, "", "", -1, 0, true, spin) {
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

	err = g.rem.EmptyTrash()
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

func (g *Commands) trashByMatch(inTrash bool) error {
	matches, err := g.rem.FindMatches(g.opts.Path, g.opts.Sources, inTrash)
	if err != nil {
		return err
	}
	var cl []*Change
	p := g.opts.Path
	if p == "/" {
		p = ""
	}
	for match := range matches {
		if match == nil {
			continue
		}
		ch := &Change{Path: p + "/" + match.Name}
		if inTrash {
			ch.Src = match
		} else {
			ch.Dest = match
		}
		cl = append(cl, ch)
	}

	if len(cl) < 1 {
		return fmt.Errorf("no matches found!")
	}

	toTrash := !inTrash
	ok := printChangeList(cl, g.opts.NoPrompt, false)
	if ok {
		return g.playTrashChangeList(cl, toTrash)
	}
	return nil
}

func (g *Commands) TrashByMatch() error {
	return g.trashByMatch(false)
}

func (g *Commands) UntrashByMatch() error {
	return g.trashByMatch(true)
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
	g.taskStart(len(cl))

	var f = g.remoteUntrash
	if toTrash {
		f = g.remoteDelete
	}

	for _, c := range cl {
		if c.Op() == OpNone {
			continue
		}

		cErr := f(c)
		if cErr != nil {
			fmt.Println(cErr)
		}
	}

	g.taskFinish()
	return err
}
