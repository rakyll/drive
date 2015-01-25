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
)

type keyValue struct {
	path        string
	file        *File
	permissions []*drive.Permission
	err         error
}

func (g *Commands) Stat() error {
	for _, relToRootPath := range g.opts.Sources {
		res := g.stat(relToRootPath)
		if res.err != nil {
			fmt.Println(res.err)
			continue
		}
		p, file, perms := res.path, res.file, res.permissions
		if file == nil {
			continue
		}
		fmt.Println(p)
		for _, perm := range perms {
			prettyPermission(perm)
		}
	}
	return nil
}

func prettyPermission(perm *drive.Permission) {
	fmt.Printf("Name: %v <%s>\nRole: %v\nAccountType: %v\nValue: %v\n", perm.Name, perm.EmailAddress, perm.Role, perm.Type, perm.Value)
}

func (g *Commands) stat(relToRootPath string) *keyValue {
	file, err := g.rem.FindByPath(relToRootPath)
	if err != nil {
		return &keyValue{
			err: err,
		}
	}

	perms, permErr := g.rem.listPermissions(file.Id)
	return &keyValue{
		path:        relToRootPath,
		file:        file,
		permissions: perms,
		err:         permErr,
	}
}
