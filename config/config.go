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
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
)

type Context struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	AbsPath      string `json:"-"`
}

func (c *Context) AbsPathOf(fileOrDirPath string) string {
	return path.Join(c.AbsPath, fileOrDirPath)
}

func (c *Context) Read() (err error) {
	var data []byte
	if data, err = ioutil.ReadFile(credentialsPath(c.AbsPath)); err != nil {
		return
	}
	err = json.Unmarshal(data, c)
	return
}

func (c *Context) Write() (err error) {
	var data []byte
	if data, err = json.Marshal(c); err != nil {
		return
	}
	return ioutil.WriteFile(credentialsPath(c.AbsPath), data, 0600)
}

// Discovers the gd directory, if no gd directory or credentials
// could be found for the path, returns ErrNoContext.
func Discover(currentAbsPath string) (context *Context, err error) {
	p := currentAbsPath
	found := false
	for {
		info, e := os.Stat(gdPath(p))
		if e == nil && info.IsDir() {
			found = true
			break
		}
		newPath := path.Join(p, "..")
		if p == newPath {
			break
		}
		p = newPath
	}

	if !found {
		return nil, errors.New("no gd context is found; use gd init")
	}
	context = &Context{AbsPath: p}
	err = context.Read()
	return
}

func Initialize(absPath string) (c *Context, err error) {
	p := gdPath(absPath)
	if err = os.MkdirAll(p, 0755); err != nil {
		return
	}
	c = &Context{AbsPath: absPath}
	err = c.Write()
	return
}

func gdPath(absPath string) string {
	return path.Join(absPath, ".gd")
}

func credentialsPath(absPath string) string {
	return path.Join(gdPath(absPath), "credentials.json")
}
