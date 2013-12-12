# gd

`gd` is a tiny program to pull or push files and directories from Google Drive. You need go tools installed in order to build the program.

    go get github.com/rakyll/gd

Use `gd help` for further reference.

* `$ gd init [--C <path>]`
* `$ gd auth [--C <path>]` // starts the authorization and authentication wizard
* `$ gd pull [--C <path>] [--recursive=true]` // pulls from remote
* `$ gd push [--C <path>] [--recursive=true]` // pushes to the remote
* `$ gd stat [--C <path>]` // shows current upload and download status
* `$ gd diff [--C <filepath>]` // outputs a diff of local and remote
