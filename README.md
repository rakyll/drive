# drive

[![Build Status](https://travis-ci.org/odeke-em/drive.png?branch=master)](https://travis-ci.org/odeke-em/drive)

`drive` is a tiny program to pull or push [Google Drive](https://drive.google.com) files. You need at least go1.2 installed in order to build the program.

 `drive` was originally developed by [Burcu Dogan](https://github.com/rakyll) while working on the Google Drive team.

However, she no longer has the time to work on it so I am its new maintainer.

## Installation

   $> go get -u -d github.com/odeke-em/drive/cmd/drive # To force the latest download of the latest code from the Github repo

   $> go get github.com/odeke-em/drive/cmd/drive # To install the code locally checked out on your machine.


Use `drive help` for further reference.

	$ drive version
	$ drive init [path]
	$ drive pull [-r -no-prompt path1 path2 path3 ...] # pulls from remotes
	$ drive pull [-r -no-prompt -hidden path] # pulls even hidden paths from remote
	$ drive pull [-r -no-prompt -force path] # forces addition to local of all pulls from remote
	$ drive pull [-r -no-prompt -export ext1,ext2,ext3 -export-dir <export-dir> path] # pulls from remote and exports Docs + Sheets to one of its export formats.
    e.g:
	$ drive pull [-r -no-prompt -export pdf,docx,rtf,html ReportII.txt] # pull ReportII.txt from
	 remote and export it to pdf, docx, rtf and html.
	$ drive pull [-r -no-prompt -export pdf,docx,rtf,html -export-dir ~/Desktop/exports ReportII.txt] # pull ReportII.txt from
	 remote and export it to pdf, docx, rtf and html and save it in ~/Desktop/exports
        
	$ drive push [-r -no-prompt path1 path2 path3 ...] # pushes to the remotes
	$ drive push [-r -no-prompt -force path] # pushes all files as additions to the remote
	$ drive push [-r -hidden path] # pushes also hidden directories and paths to the remote
	# To push from a location not within the drive:
	$ drive push -m $LOCATION .
    e.g
        `drive push -m /mnt/media`
	$ drive diff [path1 path2 ...] # outputs diffs of multiple paths.
	$ drive pub [path1 path2 ...] # publishes the files, outputs URL.
	$ drive unpub [path1 path2 ...] # revokes public access to the specified files.
	$ drive list [-d 2 path1 path2 path3 ...] # list contents of paths on remote to a recursion depth of 2.

	# Note using the no-clobber option for push or pull ensures that only ADDITIONS are made
	# any modifications or deletions are ignored and those files are safe.
	$ drive pull -no-clobber
	$ drive push -no-clobber

	$ drive trash [path1 path2 ...] # Sends the remote specified files to the trash.
	$ drive untrash [path1 path2 ...] # Restore the specified remote files from trash.
	$ drive emptytrash
	$ drive emptytrash [-no-prompt] # No prompt is presented before emptying out your trash.

	$ drive quota # To return quota information

## Configuration

If you would like to use your own client ID/client secret pair with `drive`, set the `GOOGLE_API_CLIENT_ID` and `GOOGLE_API_CLIENT_SECRET` variables in your environment


## Platform Packages
    Get drive on your platform from the list below
   + [Arch Linux](https://aur.archlinux.org/packages/drive) prepared by [Johnathan Jenkins](https://github.com/shaggytwodope)


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


## Known issues
* Probably, it doesn't work on Windows.
* Google Drive allows a directory to contain files/directories with the same name. Client doesn't handle these cases yet. We don't recommend you to use `drive` if you have such files/directories to avoid data loss.
* Racing conditions occur if remote is being modified while we're trying to update the file. Google Drive provides resource versioning with ETags, use Etags to avoid racy cases.


## Sample Walk-through starting from installation.
**System pre-requisites:**
======


   + An installation of Go with version >= 1.2

     To check your go version:

     `$> go version`

   + Your gopath should have been set in your e.g

      Sample set up:

      `vi ~/.bashrc or vi ~/.bash_profile`

      `export GOPATH=~/gopath`

      `export PATH=$GOPATH:$GOPATH/bin:${PATH}`


**Installing drive, the program:**
=======

   $> go get github.com/odeke-em/drive/cmd/drive # To install the code locally checked out on your machine.

   $> go get -u -d github.com/odeke-em/drive/cmd/drive # To force the latest download of the latest code from the Github repo

   Note: Running go install or go get github.com/odeke-em/drive will not create an executable.


**Initializing a drive:**
=====


  To setup/mount your Google Drive directory to path ~/GDRIVE
  
  ![drive init](https://github.com/odeke-em/wiki_content/blob/master/drive/init.png)


**Pull:**
====

 + What does a pull operation do?

  A pull operation gets the manifest of content from your Google drive and tries to mirror the Google Drive's

  content to your drive. This will entail downloading content from the cloud as well as deleting content that

 is present on your local drive folder but not on your Google Drive.

 + How is a pull operation performed?

  ** Performing a pull with no arguments will pull all the respective content in from the current path

  ![drive pull](https://github.com/odeke-em/wiki_content/blob/master/drive/pull_all.png)

  ** You can also pull from a specific path
  ![drive pull](https://github.com/odeke-em/wiki_content/blob/master/drive/pull_specific.png)

 + Can I export documents?
 
  Yes, by default Google Docs + Sheets cannot be downloaded raw but only exported. To export

  your documents pass in: -export and a list of desired exports e.g:
  
  * After creating a new document:
  
  ![drive newdoc](https://github.com/odeke-em/wiki_content/blob/master/drive/testDocument1.png)
  
  * Now the exported pull:
  ![drive exported pull](https://github.com/odeke-em/wiki_content/blob/master/drive/export_usage.png)

**Push:**
====


+ What does "push" do?

  push uploads/updates content to your Google Drive mirroring its directory structure locally.
 
+ How is a push operation performed?

  ![drive push](https://github.com/odeke-em/wiki_content/blob/master/drive/pushing.png)

**Publish "pub"**
====

  + What does pub do?

   "pub" publishes a file globally so that anyone with a link to it can read the file.

  + How do I publish a file?

   ![drive pub](https://github.com/odeke-em/wiki_content/blob/master/drive/pub.png)

  + What happens if I publish a file that doesn't yet exist on the my Google Drive?

   ![drive pub non-existant](https://github.com/odeke-em/wiki_content/blob/master/drive/pub_unexistant.png)


**Unpublish "unpub"**
=========


  + What does unpub do?

    "unpub" revokes public read access to a file.

  + How do I unpublish a file?

  ![drive unpub](https://github.com/odeke-em/wiki_content/blob/master/drive/unpub.png)


**Trashing and Untrashing files**
======

  + How do I trash/untrash a file on the cloud?

    The options here are:

    1) Using your browser login to Google Drive and delete that file.

       * The next pull that you do should clean up that file off your disk.

    2) 

   ![drive trash/untrash](https://github.com/odeke-em/wiki_content/blob/master/drive/trash-untrash.png)


**Listing on the remote**
======

  + What does list do ?
    
    list performs a paginated list of paths on the cloud.
 
  + How is it done ?

   ![drive list](https://github.com/odeke-em/wiki_content/blob/master/drive/list.png)

   ![drive trash-untrash-list](https://github.com/odeke-em/wiki_content/blob/master/drive/trash-untrash-list.png)

**Emptying the trash**
=====

    + What does this do ?
        Cleans out your trash permanently.
    
    + How is it done ?

   ![drive emptytrash](https://github.com/odeke-em/wiki_content/blob/master/drive/emptytrash.png)

**Quota Information**
====
    + What does `quota` do ?
        Returns you information about your drive.

    + How is it done ?
    
    `drive quota`
        `Bytes Used: 967MB`
        `Bytes Free: 15GB`
        `Total Bytes: 16GB`
        `Account type: LIMITED`


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
