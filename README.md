# gd

[![Build Status](https://travis-ci.org/rakyll/gd.png?branch=master)](https://travis-ci.org/rakyll/gd)

`gd` is a tiny program to pull or push files and directories from Google Drive. You need go tools installed in order to build the program.

    go install github.com/rakyll/gd

Use `gd help` for further reference.

* `$ gd init [path]`
* `$ gd pull [-r -no-prompt path]` // pulls from remote
* `$ gd push [-r -no-prompt path]` // pushes to the remote
* `$ gd diff [path]` // outputs a diff of local and remote
