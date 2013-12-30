# drive

[![Build Status](https://travis-ci.org/rakyll/drive.png?branch=master)](https://travis-ci.org/rakyll/drive)

`drive` is a tiny program to pull or push files and directories from [Google Drive](https://drive.google.com). You need go 1.2 installed, in order to build the program.

## Installation

    go install github.com/rakyll/drive

Use `drive help` for further reference.

	$ drive init [path]
	$ drive pull [-r -no-prompt path] # pulls from remote
	$ drive push [-r -no-prompt path] # pushes to the remote
	$ drive diff [path] # outputs a diff of local and remote


## Why another Google Drive client?
`drive` is not a sync deamon, it provides:

* Upstreaming and downstreaming unlike sync clients. User has full control what to do with their local copy and when. Do some changes, either push it to remote or revert it to the remote version. Perform these actions with user prompt. 

	    $ echo "hello" > hello.txt
	    $ drive push # pushes hello.txt to Google Drive
	    $ echo "more text" >> hello.txt
	    $ drive pull # overwrites the local changes with the remote version

* Allowing to work with a specific file or directory, optionally not recursively. If you recently uploaded a large VM image to Google Drive, yet  only a few text files are required for you to work, simply only push/pull the file you want to work with.

	    $ echo "hello" > hello.txt
	    $ drive push hello.txt # pushes only the specified file
	    $ drive pull path_to/a/b # pulls the remote directory recursively

* Better I/O scheduling. One of the major goals is to provide better scheduling to fasten your daily interaction with Google Drive backend.

## Known issues
* Probably it doesn't work on Windows.
* Google Drive allows a directory to contain files/directories with the same name. Client doesn't handle these cases yet. We don't recommend you to use `drive` if you have such files/directories to avoid data loss.