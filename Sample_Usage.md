Welcome to the drive wiki!

**System pre-requisites for Drive:**
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


   $> go get github.com/rakyll/drive/cmd/drive

   Note: Running go install or go get github.com/rakyll/drive will not create an executable.


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


**Deleting files**
======

  + How do I delete a file on the cloud?

    The options here are:

    1) Using your browser login to Google Drive and delete that file.

       * The next pull that you do should clean up that file off your disk.

    2) Using your terminal, take that file out of its position and then perform a push on only that file.

       * The moving can be performed with a rename, move or delete (rm)

        ![drive unpub](https://github.com/odeke-em/wiki_content/blob/master/drive/push_to_trash.png)
