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
	var file *File
	var link string
	if file, err = c.rem.FindByPath(c.opts.Path); err != nil {
		return
	}
	if link, err = c.rem.Publish(file.Id); err != nil {
		return
	}
	fmt.Println("Published on", link)
	return
}

func (c *Commands) Unpublish() error {
	file, err := c.rem.FindByPath(c.opts.Path)
	if err != nil {
		return err
	}
	return c.rem.Unpublish(file.Id)
}
