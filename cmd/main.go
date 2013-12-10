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
	"log"
	"os"

	"github.com/rakyll/god"
	"github.com/rakyll/god/config"
)

func main() {
	// TODO: config.Initialize()
	// TODO: process level lock to protect pulls and pushes from themselves
	context := &config.Context{
		AbsPath:      "/Users/burcud/godtest",
		RefreshToken: "1/RqZ7kz24jGa5BE8DhqXRyCw2i2L50wvrnBiGvFlGzzk",
		ClientId:     "354790962074-uhtvp8nslh2334lk1krv4arpaqdm24jl.apps.googleusercontent.com",
		ClientSecret: "8glhKA6mkyvUWD4vC1kGsBiy",
	}

	g := god.New(context, &god.Options{
		Path:        "/",
		IsRecursive: true,
	})

	log.Println(g.Pull())
	return

	if len(os.Args) < 2 {
		help(os.Args)
		return
	}

	switch os.Args[1] {
	case "init":
	case "auth":
		log.Println("auth")
	case "pull":
		log.Println("pull")
	case "push":
		log.Println("push")
	case "stat":
		log.Println("stat")
	case "diff":
		log.Println("diff")
	default:
		help(os.Args)
	}
}

func help(args []string) {
	log.Println("print help")
}
