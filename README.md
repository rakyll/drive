# gd

`gd` is a tiny program to pull or push files and directories from Google Drive. You need go tools installed in order to build the program.

    go get github.com/rakyll/gd

Use `gd help` for further reference.

* `$ gd [-c=<gdDir>] init`
* `$ gd [-c=<gdDir>] pull [-r=false] [-no-prompt=false] <path>` // pulls from remote
* `$ gd [-c=<gdDir>] push [-r=false] [-no-prompt=false] <path>` // pushes to the remote
* `$ gd [-c=<gdDir>] diff <path>` // outputs a diff of local and remote
