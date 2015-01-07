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
)

var Version = "0.0.4b"

const (
	DescInit       = "inits a directory and authenticates user"
	DescPull       = "pulls remote changes from google drive"
	DescPush       = "push local changes to google drive"
	DescDiff       = "compares a local file with remote"
	DescEmptyTrash = "cleans out your trash"
	DescHelp       = "Get help for a topic"
	DescList       = "lists the contents of remote path"
	DescQuota      = "prints out the space information"
	DescPublish    = "publishes a file and prints its publicly available url"
	DescTrash      = "moves the file to trash"
	DescUntrash    = "restores the file from trash"
	DescUnpublish  = "revokes public access to a file"
	DescVersion    = "prints the version"
)

var shortToCmd = map[string][]string{
	"diff":       []string{DescDiff, "Accepts multiple paths for comparison"},
	"emptytrash": []string{DescEmptyTrash},
	"init":       []string{DescInit, "This is where you drive credentials will be placed"},
	"pull":       []string{DescPull, "Accepts multiple paths"},
	"push": []string{
		DescPush, "Accepts multiple paths", "Push comes in a couple of flavors",
		"\t* Ordinary push: `drive push path1 path2 path3`",
		"\t* Mounted push: `drive push -m path1 [path2 path3] drive_context_path`",
	},
	"list":    []string{DescList, "Accepts multiple paths"},
	"quota":   []string{DescQuota},
	"trash":   []string{DescTrash, "Accepts multiple paths"},
	"untrash": []string{DescUntrash, "Accepts multiple paths"},
	"pub":     []string{DescPublish, "Accepts multiple paths"},
	"unpub":   []string{DescUnpublish, "Accepts multiple paths"},
	"version": []string{DescVersion, fmt.Sprintf("current version is: %s", Version)},
}

func ShowDescription(topic string) {
	help, ok := shortToCmd[topic]
	if !ok {
		fmt.Printf("Unkown command '%s' type `drive help` for usage\n", topic)
	} else {
		description, documentation := help[0], help[1:]
		fmt.Printf("Name\n\t%s - %s\n", topic, description)
		if len(documentation) >= 1 {
			fmt.Println("Description")
			for _, line := range documentation {
				fmt.Printf("\t%s\n", line)
			}
		}
	}
}
