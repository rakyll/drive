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

package config

import (
	"errors"
	"os"
	"path"
	"time"
)

type Context struct {
	ClientId           string
	ClientSecret       string
	RefreshToken       string
	AccessTokenExpires *time.Time
	AbsPath            string
}

// Discovers the gd directory, if no gd directory or credentials
// could be found for the path, returns ErrNoContext.
func Discover(currentAbsPath string) (context *Context, err error) {
	gdPath := currentAbsPath
	found := false
	for {
		info, e := os.Stat(path.Join(gdPath, ".gd"))
		if e == nil && info.IsDir() {
			found = true
			break
		}
		newPath := path.Join(gdPath, "..")
		if gdPath == newPath {
			break
		}
		gdPath = newPath
	}

	if !found {
		return nil, errors.New("no gd context is found; use gd init")
	}
	// TODO: read credentials
	return &Context{AbsPath: gdPath}, nil
}

func (c *Context) AbsPathOf(fileOrDirPath string) string {
	return path.Join(c.AbsPath, fileOrDirPath)
}
