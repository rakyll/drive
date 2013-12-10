# god

`god` is a tiny program to pull or push files and directories from Google Drive. You need go tools installed in order to build the program.

    go get github.com/rakyll/god
    

Use `god --help` for further reference.

* `$ god init [--C <path>]`
* `$ god auth [--C <path>]` // starts the authorization and authentication wizard
* `$ god pull [--C <path>] [--recursive=true]` // pulls from remote
* `$ god push [--C <path>] [--recursive=true]` // pushes to the remote
* `$ god stat [--C <path>]` // shows current upload and download status
* `$ god diff [--C <filepath>]` // outputs a diff of local and remote
