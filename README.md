# drive

[![Build Status](https://travis-ci.org/odeke-em/drive.png?branch=master)](https://travis-ci.org/odeke-em/drive)

`drive` is a tiny program to pull or push [Google Drive](https://drive.google.com) files.

`drive` was originally developed by [Burcu Dogan](https://github.com/rakyll) while working on the Google Drive team. This repository contains the latest version of the code, as she is no longer able to maintain it.

## Requirements

go 1.2 or higher is required. See [here](https://golang.org/doc/install) for installation instructions and platform installers.

## Installation

To install from the latest source, run:

```shell
$ go get -u github.com/odeke-em/drive/cmd/drive
```

Binary releases for tagged versions are also [available](https://github.com/odeke-em/drive/releases).

### Platform Packages

Platform | Author |
---------| -------|
[Arch Linux](https://aur.archlinux.org/packages/drive) | [Jonathan Jenkins](https://github.com/shaggytwodope)

## Configuration

Optionally set the `GOOGLE_API_CLIENT_ID` and `GOOGLE_API_CLIENT_SECRET` environment variables to use your own API keys.

## Usage

### Initializing

Before you can use `drive`, you need to mount your Google Drive directory on your local file system:

```shell
$ drive init ~/gdrive
$ cd ~/gdrive
```

### Pulling

The `pull` command downloads data from Google Drive that does not exist locally, and deletes local data that is not present on Google Drive. 

Run it without any arguments to pull all of the files from the current path:

```shell
$ drive pull
```

To pull specific files or directories, pass in one or more paths:

```shell
$ drive pull photos/img001.png docs
```

#### Exporting Docs

By default, the `pull` command will export Google Docs documents as PDF files. To specify other formats, use the `-export` option:

```shell
$ drive pull -export pdf,rtf,docx,txt
```

By default, the exported files will be placed in a new directory suffixed by `_exports` in the same path. To export the files to a different directory, use the `-export-dir` option:

```shell
$ drive pull -export pdf,rtf,docx,txt -export-dir ~/Desktop/exports
```

**Supported formats:**

* doc, docx
* jpeg, jpg
* gif
* html
* odt
* rtf
* pdf
* png
* ppt, pptx
* svg
* txt, text
* xls, xlsx

### Pushing

The `push` command uploads data to Google Drive to mirror data stored locally.

Like `pull`, you can run it without any arguments to push all of the files from the current path, or you can pass in one or more paths to push specific files or directories.

### Publishing

The `pub` command publishes a file or directory globally so that anyone can view it on the web using the link returned.

```shell
$ drive pub photos
```

### Unpublishing

The `unpub` command is the opposite of `pub`. It unpublishes a previously published file or directory.

```shell
$ drive unpub photos
```

### Trashing and Untrashing

Files can be trashed using the `trash` command:

```shell
$ drive trash Demo
```

Files that have been trashed can be restored using the `untrash` command:

```shell
$ drive untrash Demo
```

### Emptying the Trash

Emptying the trash will permanently delete all trashed files. They will be unrecoverable using `untrash` after running this command.

```shell
$ drive emptytrash
```

### Listing

The `list` command shows a paginated list of paths on the cloud.

Run it without arguments to list all files in the current directory:

```shell
$ drive list
```

Pass in a directory path to list files in that directory:

```shell
$ drive list photos
```

The `-trashed` option can be specified to show trashed files in the listing:

```shell
$ drive list -trashed photos
```

### Quota

The `quota` command prints information about your drive, such as the account type, bytes used/free, and the total amount of storage available.

```shell
$ drive quota
```

## Why another Google Drive client?

Background sync is not just hard, it is stupid. My technical and philosophical rants about why it is not worth to implement:

* Too racy. Data has been shared between your remote resource, local disk and sometimes in your sync daemon's in-memory struct. Any party could touch a file any time, hard to lock these actions. You end up working with multiple isolated copies of the same file and trying to determine which is the latest version and should be synced across different contexts.

* It requires great scheduling to perform best with your existing environmental constraints. On the other hand, file attributes has an impact on the sync strategy. Large files are blocking, you wouldn't like to sit on and wait for a VM image to get synced before you start to work on a tiny text file.

* It needs to read your mind to understand your priorities. Which file you need most? It needs to read your mind to foresee your future actions. I'm editing a file, and saving the changes time to time. Why not to wait until I feel confident enough to commit the changes to the remote resource?

`drive` is not a sync deamon, it provides:

* Upstreaming and downstreaming. Unlike a sync command, we provide pull and push actions. User has opportunity to decide what to do with their local copy and when. Do some changes, either push it to remote or revert it to the remote version. Perform these actions with user prompt.

	    $ echo "hello" > hello.txt
	    $ drive push # pushes hello.txt to Google Drive
	    $ echo "more text" >> hello.txt
	    $ drive pull # overwrites the local changes with the remote version

* Allowing to work with a specific file or directory, optionally not recursively. If you recently uploaded a large VM image to Google Drive, yet  only a few text files are required for you to work, simply only push/pull the file you want to work with.

	    $ echo "hello" > hello.txt
	    $ drive push hello.txt # pushes only the specified file
	    $ drive pull path/to/a/b # pulls the remote directory recursively

* Better I/O scheduling. One of the major goals is to provide better scheduling to improve upload/download times.

* Possibility to support multiple accounts. Pull from or push to multiple Google Drive remotes. Possibility to support multiple backends. Why not to push to Dropbox or Box as well?

## Known issues

* Probably, it doesn't work on Windows.
* Google Drive allows a directory to contain files/directories with the same name. Client doesn't handle these cases yet. We don't recommend you to use `drive` if you have such files/directories to avoid data loss.
* Racing conditions occur if remote is being modified while we're trying to update the file. Google Drive provides resource versioning with ETags, use Etags to avoid racy cases.

## LICENSE

Copyright 2013 Google Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


