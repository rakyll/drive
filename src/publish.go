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
)

func (c *Commands) Publish() (err error) {
	for _, relToRoot := range c.opts.Sources {
		if pubErr := c.pub(relToRoot); pubErr != nil {
			fmt.Printf("\033[91mPub\033[00m %s:  %v\n", relToRoot, pubErr)
		}
	}
	return
}

func (c *Commands) pub(relToRoot string) (err error) {
	var file *File
	file, err = c.rem.FindByPath(relToRoot)
	if err != nil {
		return err
	}

	var link string
	link, err = c.rem.Publish(file.Id)
	if err != nil {
		return
	}
	fmt.Printf("%s Published on %s\n", relToRoot, link)
	return
}

func (c *Commands) Unpublish() error {
	for _, relToRoot := range c.opts.Sources {
		if unpubErr := c.unpub(relToRoot); unpubErr != nil {
			fmt.Printf("\033[91mUnpub\033[00m %s:  %v\n", relToRoot, unpubErr)
		}
	}
	return nil
}

func (c *Commands) unpub(relToRoot string) error {
	file, err := c.rem.FindByPath(relToRoot)
	if err != nil {
		return err
	}
	return c.rem.Unpublish(file.Id)
}
