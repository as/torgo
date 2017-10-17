# Torgo
Torgo is a set UNIX and Plan9-like command line tools. Target platforms are Windows, Plan 9, Linux, and possibly OSX. The project's technical goal is to provide command line utilities that are accessible, homogeneous, and interoperable with each other. Non-goals are compatibility with the existing syntax and semantics of similar toolkits, such as GNU coreutils.

# Warning
Use this pre-alpha toolkit at your own risk. Many of the tools will do not work the way you expect them to on Linux. 

# Installation

```
go get github.com/as/torgo/...
go install github.com/as/torgo/...
```

# Example

Print a list of duplicate files in the current working directory tree, prefixed by the number of duplicates and the sha1 sum of the file contents

`walk -f | hash sha1 - | uniq -d -l -c -x "[0-9a-f]{16}"  | sort -n`

See the wiki for more examples

# License
The license shall be identical to that of the Go programming language at the time of 2017/01/01. (BSD-like). 
