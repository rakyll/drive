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
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cheggaaa/pb"
	"github.com/odeke-em/drive/config"
	"github.com/odeke-em/log"
)

var (
	ErrNoContext = errors.New("not in a drive context")
)

const (
	DriveIgnoreSuffix = ".driveignore"
)

type Options struct {
	// Depth is the number of pages/ listing recursion depth
	Depth int
	// Exports contains the formats to export your Google Docs + Sheets to
	// e.g ["csv" "txt"]
	Exports []string
	// Directory to put the exported Google Docs + Sheets, if not
	// provided will export them to the same dir as the source files are.
	ExportsDir string
	// Force once set always converts NoChange into an Addition
	Force bool
	// Hidden discovers hidden paths if set
	Hidden       bool
	IgnoreRegexp *regexp.Regexp
	// IgnoreChecksum when set avoids the step
	// of comparing checksums as a final check.
	IgnoreChecksum bool
	// IgnoreConflict when set turns off the conflict resolution safety.
	IgnoreConflict bool
	// Allows listing of content in trash
	InTrash bool
	Meta    *map[string][]string
	Mount   *config.Mount
	// NoClobber when set prevents overwriting of stale content
	NoClobber bool
	// NoPrompt overwrites any prompt pauses
	NoPrompt bool
	Path     string
	// PageSize determines the number of results returned per API call
	PageSize  int64
	Recursive bool
	// Sources is a of list all paths that are
	// within the scope/path of the current gd context
	Sources []string
	// TypeMask contains the result of setting different type bits e.g
	// Folder to search only for folders etc.
	TypeMask int
	// Piped when set means to infer content to or from stdin
	Piped bool
}

type Commands struct {
	context *config.Context
	rem     *Remote
	opts    *Options
	log     *log.Logger

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

		if !opts.Force {
			ignoresPath := filepath.Join(context.AbsPath, DriveIgnoreSuffix)
			opts.IgnoreRegexp = readCommentedFileCompileRegexp(ignoresPath)
		}
	}
	return &Commands{
		context: context,
		rem:     r,
		opts:    opts,
		log:     log.New(os.Stdin, os.Stdout, os.Stderr),
	}
}

func readCommentedFileCompileRegexp(p string) *regexp.Regexp {
	clauses, err := readCommentedFile(p, "#")
	if err != nil {
		return nil
	}
	regExComp, regErr := regexp.Compile(strings.Join(clauses, "|"))
	if regErr != nil {
		return nil
	}
	return regExComp
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
