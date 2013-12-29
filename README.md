# drive

[![Build Status](https://travis-ci.org/rakyll/drive.png?branch=master)](https://travis-ci.org/rakyll/drive)

`drive` is a tiny program to pull or push files and directories from Google Drive. You need go tools installed in order to build the program.

    go install github.com/rakyll/drive

Use `drive help` for further reference.

* `$ drive init [path]`
* `$ drive pull [-r -no-prompt path]` // pulls from remote
* `$ drive push [-r -no-prompt path]` // pushes to the remote
* `$ drive diff [path]` // outputs a diff of local and remote
