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

package gd

import (
	"path"

	"github.com/rakyll/gd/config"
	"github.com/rakyll/gd/remote"

	"github.com/rakyll/gd/third_party/github.com/cheggaaa/pb"
)

type Options struct {
	Path        string
	IsNoPrompt  bool
	IsRecursive bool
	IsForce     bool
}

type Gd struct {
	context *config.Context
	rem     *remote.Remote
	opts    *Options

	progress *pb.ProgressBar
}

func New(context *config.Context, opts *Options) *Gd {
	var r *remote.Remote
	if context != nil {
		r = remote.New(context)
	}
	// TODO: should always start with /
	opts.Path = path.Clean(opts.Path)
	return &Gd{
		context: context,
		rem:     r,
		opts:    opts,
	}
}

func (g *Gd) taskStart(numOfTasks int) {
	if numOfTasks > 0 {
		g.progress = pb.StartNew(numOfTasks)
	}
}

func (g *Gd) taskDone() {
	if g.progress != nil {
		g.progress.Increment()
	}
}

func (g *Gd) taskFinish() {
	if g.progress != nil {
		g.progress.Finish()
	}
}
