# drive

[![Build Status](https://travis-ci.org/odeke-em/drive.png?branch=master)](https://travis-ci.org/odeke-em/drive)

`drive` is a tiny program to pull or push [Google Drive](https://drive.google.com) files. You need at least go1.2 installed in order to build the program.

## Installation

    go get github.com/odeke-em/drive/cmd/drive

Use `drive help` for further reference.

	$ drive init [path]
	$ drive pull [-r -no-prompt path] # pulls from remote
	$ drive pull [-r -no-prompt -export ext1,ext2,ext3 path] # pulls from remote and exports Docs + Sheets to one of its export formats.
    e.g:
	$ drive pull [-r -no-prompt -export pdf,docx,rtf,html ReportII.txt] # pull ReportII.txt from
	 remote and export it to pdf, docx, rtf and html.
        
	$ drive push [-r -no-prompt path] # pushes to the remote
	$ drive push [-r -hidden path] # pushes also hidden directories and paths to the remote
	# To push from a location not within the drive:
	$ drive push -m $LOCATION .
    e.g
        `drive push -m /mnt/media`
	$ drive diff [path] # outputs a diff of local and remote
	$ drive pub [path] # publishes a file, outputs URL
	$ drive unpub [path] # revokes public access to the file

	# Note using the no-clobber option for push or pull ensures that only ADDITIONS are made
	# any modifications or deletions are ignored and those files are safe.
	$ drive pull -no-clobber
	$ drive push -no-clobber

## Configuration

If you would like to use your own client ID/client secret pair with `drive`, set the `GOOGLE_API_CLIENT_ID` and `GOOGLE_API_CLIENT_SECRET` variables in your environment

## Why another Google Drive client?
Background sync is not just hard, it's stupid. My technical and philosophical rants about why it is not worth to implement:

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

## Notes:
* Google Docs cannot be directly downloaded but only
exported to different forms e.g docx, xlsx, csv etc.
When doing a pull remember to include option `-export ext1,ext2,ext3`
where ext1, ext2, ... could be:
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

The exported files will be placed in a directory in the same path
as the source Doc but affixed with '\_exports' e.g
drive pull -export gif,jpg,svg logo
if successful will create a directory logo\_exports which will look like:
|- logo\_exports
                |- logo.gif
                |- logo.png
                |- logo.svg

## [Sample usage](https://github.com/odeke-em/drive/blob/master/SampleWalkThrough.md):

## Known issues
* Probably, it doesn't work on Windows.
* Google Drive allows a directory to contain files/directories with the same name. Client doesn't handle these cases yet. We don't recommend you to use `drive` if you have such files/directories to avoid data loss.
* Racing conditions occur if remote is being modified while we're trying to update the file. Google Drive provides resource versioning with ETags, use Etags to avoid racy cases.

## License
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
