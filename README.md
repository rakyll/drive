# drive

[![Build Status](https://travis-ci.org/odeke-em/drive.png?branch=master)](https://travis-ci.org/odeke-em/drive)

`drive` is a tiny program to pull or push [Google Drive](https://drive.google.com) files.

`drive` was originally developed by [Burcu Dogan](https://github.com/rakyll) while working on the Google Drive team. This repository contains the latest version of the code, as she is no longer able to maintain it.

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
  - [Platform Packages](#platform-packages)
- [Configuration](#configuration)
- [Usage](#usage)
  - [Initializing](#initializing)
  - [Pulling](#pulling)
    - [Exporting Docs](#exporting-docs)
  - [Pushing](#pushing)
  - [Publishing](#publishing)
  - [Unpublishing](#unpublishing)
  - [Sharing and Emailing](#sharing-and-emailing)
  - [Unsharing](#unsharing)
  - [Touching](#touch)
  - [Trashing and Untrashing](#trashing-and-untrashing)
  - [Emptying the Trash](#emptying-the-trash)
  - [Listing Files](#listing-files)
  - [Stating Files](#stating-files)
  - [Quota](#quota)
  - [Features](#features)
  - [About](#about)
  - [Help](#help)
  - [DriveIgnore](#driveignore)
- [Why another Google Drive client?](#why-another-google-drive-client)
- [Known issues](#known-issues)
- [LICENSE](#license)

## Requirements

go 1.2 or higher is required. See [here](https://golang.org/doc/install) for installation instructions and platform installers.

## Installation

To install from the latest source, run:

```shell
$ go get -u github.com/odeke-em/drive/cmd/drive
```

Binary releases for tagged versions are also [available](https://github.com/odeke-em/drive/releases).

### Platform Packages

For curated packages on your favorite platform, please see file [Platform Packages.md](https://github.com/odeke-em/drive/blob/master/platform_packages.md).

Is your platform missing a package? Feel free to prepare / contribute an installation package and then submit a PR to add it in.


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

To force download from paths that otherwise would be marked with no-changes

```shell
$ drive pull -force
```

+ Note:
  In relation to [#57](https://github.com/odeke-em/drive/issues/57) and [@rakyll's #49](https://github.com/rakyll/drive/issues/49).
   A couple of scenarios in which data was getting totally clobbered and unrecoverable, drive now tries to play it safe and warn you if your data could potentially be lost e.g during a to-disk clobber for which you have no backup. At least with a push you have the luxury of untrashing content. To disable this safety, run drive with flag `-ignore-conflict` e.g:

```shell
$ drive pull -ignore-conflict collaboration_documents
```

Playing the safety card even more, if you want to get changes that are non clobberable ie only additions
run drive with flag `-no-clobber` e.g:

```shell
$ drive pull -no-clobber Makefile
```

To pull specific files or directories, pass in one or more paths:

```shell
$ drive pull photos/img001.png docs
```

Note: To ignore checksum verification during a pull:

```shell
$ drive pull -ignore-checksum
```

drive also supports piping pulled content to stdout which can be accomplished by:

```shell
$ drive pull -piped path1 path2
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

Note: To ignore checksum verification during a push:

```shell
$ drive push -ignore-checksum
```

For safety with non clobberable changes i.e only additions:

```shell
$ drive push -no-clobber
```

drive also supports pushing content piped from stdin which can be accomplished by:

```shell
$ drive push -piped path
```


+ Due to the reasons explained in the pull section, drive should be able to warn you in case of total clobbers on data. To turn off this behaviour/safety, pass in the `-ignore-conflict` flag i.e:

```shell
$ drive push -force sure_of_content
```


To get Google Drive to convert a file to its native Google Docs format

```shell
$ drive push -convert
```
Extra features: to make Google Drive attempt Optical Character Recognition (OCR) for png, gif, pdf and jpg files.

```shell
$ drive push -ocr
```
Note: To use OCR, your account should have this feature. You can find out if your account has OCR allowed.

```shell
$ drive features
```

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

### Sharing and Emailing

The `share` command enables you to share a set of files with specific users and assign them specific roles as well as specific generic access to the files. It also allows for email notifications on share.

```shell
$ drive share -emails odeke@ualberta.ca,odeke.ex@gmail.com -message "This is the substring file I told you about" -role writer -type group mnt/substringfinder.c projects/kmp.c
```

For example to share a file with users of a mailing list and a custom message

```shell
$ drive share -emails drive-mailing-list@gmail.com -message "Here is the drive code" -role group mnt/drive
```

### Unsharing

The `unshare` command revokes access of a specific accountType to a set of files.

```shell
$ drive unshare -type group mnt/drive
```

### Touching

Files that exist remotely can be touched i.e their modification time updated to that on the remote server using the `touch` command:

```shell
$ drive touch Photos/img001.png logs/log9907.txt
```

For example to touch all files that begin with digits 0  to 9:

```shell
$ drive touch -matches $(seq 0 9)
```

### Trashing and Untrashing

Files can be trashed using the `trash` command:

```shell
$ drive trash Demo
```

To trash files that contain a prefix match e.g all files that begin with Untitled, or Make

Note: This option uses the current working directory as the parent that the paths belong to.

```shell
$ drive trash -matches Untitled Make
```

Files that have been trashed can be restored using the `untrash` command:

```shell
$ drive untrash Demo
```

To untrash files that match a certain prefix pattern

```shell
$ drive untrash -matches pQueue photos Untitled
```


### Emptying the Trash

Emptying the trash will permanently delete all trashed files. They will be unrecoverable using `untrash` after running this command.

```shell
$ drive emptytrash
```

### Listing Files

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

To get detailed information about the listings e.g owner information and the version number of all listed files:

```shell
$ drive list -owners -l -version
```

### Stating Files

The `stat` commands show detailed file information for example people with whom it is shared, their roles and accountTypes, and
fileId etc. It is useful to help determine whom and what you want to be set when performing share/unshare

```shell
$ drive stat mnt
```

By default `stat` won't recursively stat a directory, to enable recursive stating:

```shell
$ drive stat -r mnt
```

### Quota

The `quota` command prints information about your drive, such as the account type, bytes used/free, and the total amount of storage available.

```shell
$ drive quota
```

### Features

The `features` command provides information about the features present on the
drive being queried and the request limit in queries per second

```shell
$ drive features
```

### About

The `about` command provides information about the program as well as that about
your Google Drive. Think of it as a hybrid between the `features` and `quota` commands.
```shell
$ drive about
```

OR for detailed information
```shell
$ drive about -features -quota
```

### Help

Run the `help` command without any arguments to see information about the commands that are available:

```shell
$ drive help
```

Pass in the name of a command to get information about that specific command and the options that can be passed to it.

```shell
$ drive help push
```

To get help for all the commands
```shell
$ drive help all
```

### Copying

drive allows you to copy content remotely without having to explicitly download and then reupload.

```shell
$ drive copy -r blobStore.py mnt flagging
```

```shell
$ drive copy blobStore.py blobStoreDuplicated.py
```

### DriveIgnore

drive allows you to specify a '.driveignore' file similar to your .gitignore, in the root
directory of the mounted drive. Blank lines and those prefixed by '#' are considered as comments and skipped.

For example:

```shell
$ cat << $ >> .driveignore
> # My drive ignore file
> .gd
> .so
> .swp
> $
```

Note: Pattern matching and suffixes are done by regular expression matching so make sure to use a valid regular expression suffix.


## Why another Google Drive client?

Background sync is not just hard, it is stupid. My technical and philosophical rants about why it is not worth to implement:

* Too racy. Data has been shared between your remote resource, local disk and sometimes in your sync daemon's in-memory struct. Any party could touch a file any time, hard to lock these actions. You end up working with multiple isolated copies of the same file and trying to determine which is the latest version and should be synced across different contexts.

* It requires great scheduling to perform best with your existing environmental constraints. On the other hand, file attributes has an impact on the sync strategy. Large files are blocking, you wouldn't like to sit on and wait for a VM image to get synced before you start to work on a tiny text file.

* It needs to read your mind to understand your priorities. Which file you need most? It needs to read your mind to foresee your future actions. I'm editing a file, and saving the changes time to time. Why not to wait until I feel confident enough to commit the changes to the remote resource?

`drive` is not a sync daemon, it provides:

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

