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

// Package contains the main entry point of god.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rakyll/command"
	"github.com/rakyll/gd"
	"github.com/rakyll/gd/config"
)

var (
	flagContextPath = flag.String("c", "", "gd context path. if none set, pwd")
)

var context *config.Context

func main() {
	command.On("init", &initCmd{})
	command.On("pull", &pullCmd{})
	command.On("push", &pushCmd{})
	command.Parse()

	contextPath := *flagContextPath
	if contextPath == "" {
		contextPath, _ = os.Getwd()
	}

	var err error
	context, err = config.Discover(contextPath)
	exitWithError(err)
	context.RefreshToken = "1/RqZ7kz24jGa5BE8DhqXRyCw2i2L50wvrnBiGvFlGzzk"
	context.ClientId = "354790962074-uhtvp8nslh2334lk1krv4arpaqdm24jl.apps.googleusercontent.com"
	context.ClientSecret = "8glhKA6mkyvUWD4vC1kGsBiy"

	// add auth, stat and diff
	command.Run()
}

type initCmd struct{}

func (cmd *initCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *initCmd) Run(args []string) {
}

type pullCmd struct {
	isRecursive *bool
	isNoPrompt  *bool
}

func (cmd *pullCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.isRecursive = fs.Bool("r", false, "perform the pull action recursively")
	cmd.isNoPrompt = fs.Bool("no-prompt", false, "no prompt before applying the pull action")
	return fs
}

func (cmd *pullCmd) Run(args []string) {
	path := ""
	if len(args) > 0 {
		path = args[0]
	}
	exitWithError(gd.New(context, &gd.Options{
		Path:        path, // TODO
		IsRecursive: *cmd.isRecursive,
		IsNoPrompt:  *cmd.isNoPrompt,
	}).Pull())
}

type pushCmd struct {
	isRecursive *bool
	isNoPrompt  *bool
}

func (cmd *pushCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.isRecursive = fs.Bool("r", false, "perform the push action recursively")
	cmd.isNoPrompt = fs.Bool("no-prompt", false, "no prompt before applying the push action")
	return fs
}

func (cmd *pushCmd) Run(args []string) {
	path := ""
	if len(args) > 0 {
		path = args[0]
	}
	exitWithError(gd.New(context, &gd.Options{
		Path:        path, // TODO
		IsRecursive: *cmd.isRecursive,
		IsNoPrompt:  *cmd.isNoPrompt,
	}).Push())
}

func exitWithError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
