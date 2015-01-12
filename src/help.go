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

var Version = "0.0.4d"

const (
	AllKey        = "all"
	DiffKey       = "diff"
	EmptyTrashKey = "emptytrash"
	FeaturesKey   = "features"
	InitKey       = "init"
	ListKey       = "list"
	PullKey       = "pull"
	PushKey       = "push"
	PubKey        = "pub"
	HelpKey       = "help"
	QuotaKey      = "quota"
	TouchKey      = "touch"
	TrashKey      = "trash"
	UntrashKey    = "untrash"
	UnpubKey      = "unpub"
	VersionKey    = "version"
)

const (
	DescAll        = "print out the entire help section"
	DescDiff       = "compares local files with their remote equivalent"
	DescEmptyTrash = "permanently cleans out your trash"
	DescFeatures   = "returns information about the features of your drive"
	DescHelp       = "Get help for a topic"
	DescInit       = "initializes a directory and authenticates user"
	DescList       = "lists the contents of remote path"
	DescQuota      = "prints out information related to your quota space"
	DescPublish    = "publishes a file and prints its publicly available url"
	DescPull       = "pulls remote changes from Google Drive"
	DescPush       = "push local changes to Google Drive"
	DescTouch      = "updates a remote file's modification time to that currently on the server"
	DescTrash      = "moves files to trash"
	DescUntrash    = "restores files from trash to their original locations"
	DescUnpublish  = "revokes public access to a file"
	DescVersion    = "prints the version"
)

var docMap = map[string][]string{
	DiffKey: []string{
		DescDiff, "Accepts multiple remote paths for line by line comparison",
	},
	EmptyTrashKey: []string{
		DescEmptyTrash,
	},
	FeaturesKey: []string{
		DescFeatures,
	},
	InitKey: []string{
		DescInit, "Requests for access to your Google Drive",
		"Creating a folder that contains your credentials",
		"Note: init in an already initialized drive will erase the old credentials",
	},
	PullKey: []string{
		DescPull, "Downloads content from the remote drive or modifies",
		" local content to match that on your Google Drive",
	},
	PushKey: []string{
		DescPush, "Uploads content to your Google Drive from your local path",
		"Push comes in a couple of flavors",
		"\t* Ordinary push: `drive push path1 path2 path3`",
		"\t* Mounted push: `drive push -m path1 [path2 path3] drive_context_path`",
	},
	ListKey: []string{
		DescList,
		"List the information related a remote path not necessarily present locally",
		"Allows printing of long options and by default does minimal printing",
	},
	PubKey:     []string{DescPublish, "Accepts multiple paths"},
	QuotaKey:   []string{DescQuota},
	TouchKey:   []string{DescTouch},
	TrashKey:   []string{DescTrash, "Accepts multiple paths"},
	UntrashKey: []string{DescUntrash, "Accepts multiple paths"},
	UnpubKey:   []string{DescUnpublish, "Accepts multiple paths"},
	VersionKey: []string{
		DescVersion, fmt.Sprintf("current version is: %s", Version),
	},
}

func ShowAllDescriptions() {
	for key, _ := range docMap {
		ShowDescription(key)
		fmt.Println()
	}
}

func ShowDescription(topic string) {
	if topic == AllKey {
		ShowAllDescriptions()
		return
	}
	help, ok := docMap[topic]
	if !ok {
		fmt.Printf("Unkown command '%s' type `drive help all` for entire usage documentation\n", topic)
		ShowAllDescriptions()
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
