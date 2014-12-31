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
	"errors"
	"path"

	"github.com/cheggaaa/pb"
	"github.com/odeke-em/drive/config"
)

var (
	ErrNoContext = errors.New("not in a drive context")
)

type Options struct {
	// When set prevents overwriting of stale content
	NoClobber   bool
	Path        string
	IsNoPrompt  bool
	IsRecursive bool
	IsForce     bool
	// Exports contains the formats to export your Google Docs + Sheets to
	// e.g ["csv" "txt"]
	Exports []string
	// Hidden discovers hidden paths if set
	Hidden bool
	// Mounts is a list of all mountpoints
	// of paths that are not in the current drive context
	Mounts []*config.MountPoint
	// Sources is a of list all paths that are
	// within the scope/path of the current gd context
	Sources []string
}

type Commands struct {
	context *config.Context
	rem     *Remote
	opts    *Options

	progress *pb.ProgressBar
}

func New(context *config.Context, opts *Options) *Commands {
	var r *Remote
	if context != nil {
		r = NewRemoteContext(context)
	}
	if opts != nil {
		// should always start with /
		opts.Path = path.Clean(path.Join("/", opts.Path))
	}
	return &Commands{
		context: context,
		rem:     r,
		opts:    opts,
	}
}

func (g *Commands) taskStart(numOfTasks int) {
	if numOfTasks > 0 {
		g.progress = pb.StartNew(numOfTasks)
	}
}

func (g *Commands) taskDone() {
	if g.progress != nil {
		g.progress.Increment()
	}
}

func (g *Commands) taskFinish() {
	if g.progress != nil {
		g.progress.Finish()
	}
}
