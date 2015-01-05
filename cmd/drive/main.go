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

// Package contains the main entry point of gd.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/odeke-em/drive"
	"github.com/odeke-em/drive/config"
	"github.com/rakyll/command"
)

var context *config.Context

const Version = "0.0.3"

const (
	descInit       = "inits a directory and authenticates user"
	descPull       = "pulls remote changes from google drive"
	descPush       = "push local changes to google drive"
	descDiff       = "compares a local file with remote"
	descEmptyTrash = "cleans out your trash"
	descList       = "lists the contents of remote path"
	descPublish    = "publishes a file and prints its publicly available url"
	descTrash      = "moves the file to trash"
	descUntrash    = "restores the file from trash"
	descUnpublish  = "revokes public access to a file"
	descVersion    = "prints the version"
)

func main() {
	command.On("diff", descDiff, &diffCmd{}, []string{})
	command.On("init", descInit, &initCmd{}, []string{})
	command.On("list", descList, &listCmd{}, []string{})
	command.On("pull", descPull, &pullCmd{}, []string{})
	command.On("push", descPush, &pushCmd{}, []string{})
	command.On("pub", descPublish, &publishCmd{}, []string{})
	command.On("emptytrash", descEmptyTrash, &emptyTrashCmd{}, []string{})
	command.On("trash", descTrash, &trashCmd{}, []string{})
	command.On("untrash", descUntrash, &untrashCmd{}, []string{})
	command.On("unpub", descUnpublish, &unpublishCmd{}, []string{})
	command.On("version", descVersion, &versionCmd{}, []string{})
	command.ParseAndRun()
}

type versionCmd struct{}

func (cmd *versionCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *versionCmd) Run(args []string) {
	fmt.Printf("drive version %s\n", Version)
	exitWithError(nil)
}

type initCmd struct{}

func (cmd *initCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	return fs
}

func (cmd *initCmd) Run(args []string) {
	exitWithError(drive.New(initContext(args), nil).Init())
}

type listCmd struct {
	hidden    *bool
	pageCount *int
	recursive *bool
	depth     *int
	inTrash   *bool
}

func (cmd *listCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.depth = fs.Int("d", 1, "maximum recursion depth")
	cmd.hidden = fs.Bool("a", false, "list all paths even hidden ones")
	cmd.pageCount = fs.Int("p", -1, "number of results per pagination")
	cmd.inTrash = fs.Bool("trashed", false, "list content in the trash")
	cmd.recursive = fs.Bool("r", false, "recursively list subdirectories")

	return fs
}

func (cmd *listCmd) Run(args []string) {
	cwd, err := os.Getwd()
	exitWithError(err)

	context, path := discoverContext([]string{cwd})
	uniqArgv := uniqOrderedStr(args)
	if len(uniqArgv) < 1 {
		uniqArgv = append(uniqArgv, "")
	}

	exitWithError(drive.New(context, &drive.Options{
		Depth:     *cmd.depth,
		Hidden:    *cmd.hidden,
		Path:      path,
		Recursive: *cmd.recursive,
		Sources:   uniqArgv,
		InTrash:   *cmd.inTrash,
	}).List())
}

type pullCmd struct {
	exportsDir *string
	export     *string
	force      *bool
	hidden     *bool
	noPrompt   *bool
	noClobber  *bool
	recursive  *bool
}

func (cmd *pullCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.noClobber = fs.Bool("no-clobber", false, "prevents overwriting of old content")
	cmd.export = fs.String(
		"export", "", "comma separated list of formats to export your docs + sheets files")
	cmd.recursive = fs.Bool("r", true, "performs the pull action recursively")
	cmd.noPrompt = fs.Bool("no-prompt", false, "shows no prompt before applying the pull action")
	cmd.hidden = fs.Bool("hidden", false, "allows pulling of hidden paths")
	cmd.force = fs.Bool("force", false, "forces a pull even if no changes present")
	cmd.exportsDir = fs.String("export-dir", "", "directory to place exports")

	return fs
}

func nonEmptyStrings(v []string) (splits []string) {
	for _, elem := range v {
		if elem != "" {
			splits = append(splits, elem)
		}
	}
	return
}

func (cmd *pullCmd) Run(args []string) {
	sources, context, path := preprocessArgs(args)

	// Filter out empty strings.
	exports := nonEmptyStrings(strings.Split(*cmd.export, ","))

	exitWithError(drive.New(context, &drive.Options{
		Exports:    uniqOrderedStr(exports),
		ExportsDir: strings.Trim(*cmd.exportsDir, " "),
		Force:      *cmd.force,
		Hidden:     *cmd.hidden,
		NoPrompt:   *cmd.noPrompt,
		NoClobber:  *cmd.noClobber,
		Path:       path,
		Recursive:  *cmd.recursive,
		Sources:    sources,
	}).Pull())
}

type pushCmd struct {
	noClobber   *bool
	hidden      *bool
	force       *bool
	noPrompt    *bool
	recursive   *bool
	mountedPush *bool
}

func (cmd *pushCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.noClobber = fs.Bool("no-clobber", false, "allows overwriting of old content")
	cmd.hidden = fs.Bool("hidden", false, "allows pushing of hidden paths")
	cmd.recursive = fs.Bool("r", true, "performs the push action recursively")
	cmd.noPrompt = fs.Bool("no-prompt", false, "shows no prompt before applying the push action")
	cmd.force = fs.Bool("force", false, "forces a push even if no changes present")
	cmd.mountedPush = fs.Bool("m", false, "allows pushing of mounted paths")
	return fs
}

func preprocessArgs(args []string) ([]string, *config.Context, string) {
	var relPaths []string
	context, path := discoverContext(args)
	root := context.AbsPathOf("")

	if len(args) < 1 {
		args = []string{"."}
	}

	var err error
	for _, p := range args {
		p, err = filepath.Abs(p)
		if err != nil {
			fmt.Println(err)
			continue
		}

		relPath, err := filepath.Rel(root, p)
		if relPath == "." {
			relPath = ""
		}

		exitWithError(err)

		relPath = "/" + relPath
		relPaths = append(relPaths, relPath)
	}

	return uniqOrderedStr(relPaths), context, path
}

func (cmd *pushCmd) Run(args []string) {
	if *cmd.mountedPush {
		pushMounted(cmd, args)
	} else {
		sources, context, path := preprocessArgs(args)
		exitWithError(drive.New(context, &drive.Options{
			Force:     *cmd.force,
			Hidden:    *cmd.hidden,
			NoClobber: *cmd.noClobber,
			NoPrompt:  *cmd.noPrompt,
			Path:      path,
			Recursive: *cmd.recursive,
			Sources:   sources,
		}).Push())
	}
}

func pushMounted(cmd *pushCmd, args []string) {
	argc := len(args)

	var contextArgs, rest, sources []string

	if !*cmd.mountedPush {
		contextArgs = args
	} else {
		// Expectation is that at least one path has to be passed
		if argc < 2 {
			cwd, cerr := os.Getwd()
			if cerr != nil {
				contextArgs = []string{cwd}
			}
			rest = args
		} else {
			rest = args[:argc-1]
			contextArgs = args[argc-1:]
		}
	}

	rest = nonEmptyStrings(rest)
	context, path := discoverContext(contextArgs)
	contextAbsPath, err := filepath.Abs(path)
	if path == "." {
		path = ""
	}
	exitWithError(err)

	mountPoints, auxSrcs := config.MountPoints(path, contextAbsPath, rest, *cmd.hidden)
	sources = append(sources, auxSrcs...)

	exitWithError(drive.New(context, &drive.Options{
		Hidden:    *cmd.hidden,
		NoPrompt:  *cmd.noPrompt,
		Recursive: *cmd.recursive,
		Mounts:    mountPoints,
		NoClobber: *cmd.noClobber,
		Path:      path,
		Sources:   sources,
	}).Push())
}

type diffCmd struct {
	hidden *bool
}

func (cmd *diffCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool("hidden", false, "allows pulling of hidden paths")
	return fs
}

func (cmd *diffCmd) Run(args []string) {
	sources, context, path := preprocessArgs(args)
	exitWithError(drive.New(context, &drive.Options{
		Recursive: true,
		Path:      path,
		Hidden:    *cmd.hidden,
		Sources:   sources,
	}).Diff())
}

type publishCmd struct {
	hidden *bool
}

type unpublishCmd struct {
	hidden *bool
}

func (cmd *unpublishCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool("hidden", false, "allows pulling of hidden paths")
	return fs
}

func (cmd *unpublishCmd) Run(args []string) {
	sources, context, path := preprocessArgs(args)
	exitWithError(drive.New(context, &drive.Options{
		Path:    path,
		Sources: sources,
	}).Unpublish())
}

type emptyTrashCmd struct {
	noPrompt *bool
}

func (cmd *emptyTrashCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.noPrompt = fs.Bool("no-prompt", false, "shows no prompt before emptying the trash")
	return fs
}

func (cmd *emptyTrashCmd) Run(args []string) {
	_, context, _ := preprocessArgs(args)
	exitWithError(drive.New(context, &drive.Options{
		NoPrompt: *cmd.noPrompt,
	}).EmptyTrash())
}

type trashCmd struct {
	hidden *bool
}

func (cmd *trashCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool("hidden", false, "allows trashing hidden paths")
	return fs
}

func (cmd *trashCmd) Run(args []string) {
	sources, context, path := preprocessArgs(args)
	exitWithError(drive.New(context, &drive.Options{
		Path:    path,
		Sources: sources,
	}).Trash())
}

type untrashCmd struct {
	hidden *bool
}

func (cmd *untrashCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool("hidden", false, "allows untrashing hidden paths")
	return fs
}

func (cmd *untrashCmd) Run(args []string) {
	sources, context, path := preprocessArgs(args)
	exitWithError(drive.New(context, &drive.Options{
		Path:    path,
		Sources: sources,
	}).Untrash())
}

func (cmd *publishCmd) Flags(fs *flag.FlagSet) *flag.FlagSet {
	cmd.hidden = fs.Bool("hidden", false, "allows publishing of hidden paths")
	return fs
}

func (cmd *publishCmd) Run(args []string) {
	sources, context, path := preprocessArgs(args)
	exitWithError(drive.New(context, &drive.Options{
		Path:    path,
		Sources: sources,
	}).Publish())
}

func initContext(args []string) *config.Context {
	var err error
	var gdPath string
	var firstInit bool

	gdPath, firstInit, context, err = config.Initialize(getContextPath(args))

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// The signal handler should clean up the .gd path if this is the first time
	go func() {
		_ = <-c
		if firstInit {
			os.RemoveAll(gdPath)
		}
		os.Exit(1)
	}()

	exitWithError(err)
	return context
}

func discoverContext(args []string) (*config.Context, string) {
	var err error
	context, err = config.Discover(getContextPath(args))
	exitWithError(err)
	relPath := ""
	if len(args) > 0 {
		var headAbsArg string
		headAbsArg, err = filepath.Abs(args[0])
		if err == nil {
			relPath, err = filepath.Rel(context.AbsPath, headAbsArg)
		}
	}

	exitWithError(err)

	// relPath = strings.Join([]string{"", relPath}, "/")
	return context, relPath
}

func getContextPath(args []string) (contextPath string) {
	if len(args) > 0 {
		contextPath, _ = filepath.Abs(args[0])
	}
	if contextPath == "" {
		contextPath, _ = os.Getwd()
	}
	return
}

func uniqOrderedStr(sources []string) []string {
	cache := map[string]bool{}
	var uniqPaths []string
	for _, p := range sources {
		ok := cache[p]
		if ok {
			continue
		}
		uniqPaths = append(uniqPaths, p)
		cache[p] = true
	}
	return uniqPaths
}

func exitWithError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
