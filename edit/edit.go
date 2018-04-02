package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/as/argfile"
	"github.com/as/edit"
	"github.com/as/text"
)

var (
	h       = flag.Bool("?", false, "help")
	quest   = flag.Bool("h", false, "help")
	inplace = flag.Bool("l", false, "apply in-place chages to a list of files followed by an optional address")
	m       = flag.Bool("m", false, "print the names of modified files to stdout")
	u       = flag.Bool("u", false, "print the names of unmodified files to stdout")
)

func argparse() {
	log.SetFlags(0)
	log.SetPrefix("edit:")
	flag.Parse()
	if *h || *quest {
		Usage()
		os.Exit(0)
	}
}

var (
	err0      error
	nmodified int
)

func main() {
	argparse()
	if *inplace {
		multiflight()
	} else {
		singleflight()
	}

	if err0 != nil || nmodified == 0 {
		os.Exit(1)
	}

}

func multiflight() {
	if len(flag.Args()) < 2 {
		log.Fatalln("usage: edit [-l ] command [files ...]")
	}
	cmd := edit.MustCompile(flag.Args()[0])
	buf := new(bytes.Buffer)

	var wg sync.WaitGroup
	defer wg.Wait()
	for fd := range argfile.Next(flag.Args()[1:]...) {
		fd := fd
		wg.Add(1)
		var err error
		func() {
			defer wg.Done()
			defer fd.Close()
			buf.Reset()
			must(buf.ReadFrom(fd))
			ed, _ := text.Open(text.BufferFrom(buf.Bytes()))
			if err = cmd.Run(ed); err != nil {
				return
			}
			if cmd.Modified() {
				fd.Close()
				var fi os.FileInfo
				fi, err = os.Stat(fd.Name)
				if err != nil {
					return
				}
				err = ioutil.WriteFile(fd.Name, ed.Bytes(), fi.Mode().Perm())
				if err != nil {
					return
				}

				nmodified++
				if *m {
					fmt.Println(fd.Name)
				}
			}
			if !cmd.Modified() {
				if *u {
					if *m {
						fmt.Print("\t")
					}
					fmt.Println(fd.Name)
				}
			}
		}()
		if err != nil {
			log.Printf("%q: %s", fd.Name, err)
		}
		if err0 != nil {
			err0 = err
		}
	}
}

func singleflight() {
	if len(flag.Args()) > 2 {
		log.Println("too many arguments: use -l to specify multiple files")
		log.Fatalln("usage: edit [-l ] command [files ...]")
	}
	if len(flag.Args()) < 1 {
		log.Fatalln("usage: edit [-l ] command [files ...]")
	}
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	cmd := edit.MustCompile(strings.Join(flag.Args()[0:], " "))
	data, err := ioutil.ReadAll(in)
	if err != nil {
		log.Fatal(err)
	}

	buf, _ := text.Open(text.BufferFrom(data))
	if err = cmd.Run(buf); err != nil {
		log.Fatal(err)
	}

	io.Copy(out, bytes.NewReader(buf.Bytes()))
	out.Flush()
	if cmd.Modified() {
		nmodified++
	}
}

func must(_ int64, err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func init() {
	flag.Usage = Usage
}

func Usage() {
	fmt.Println(`
NAME
   Edit - execute structural regular expression command on file

SYNOPSIS
   edit [-l ] command [files ...]

DESCRIPTION
   Edit executes a Sam structural regular expression command
   on stdin, or optionaly (when -l is set), a list of files from the
   command line.
	
   The -l flag enables in place modification of the files in list. A
   dash reads a list of files from stdin (see walk -h).

   The exit status is zero if at least one files was modified and
   no errors occured during the command's execution

FLAGS
   -l   enables in place modification of the files in list
   -m   print modified file names to stdout
   -u   print unmodifed file names to stdout

EXAMPLE
   echo can i buy a vowel | edit ,x,vowel,x,wel,c,le,d

   echo apple > apple.txt
   echo apple > pear.txt
   edit -l ,x,apple,c,rare, apple.txt pear.txt

   walk -f | edit -l ",x,(apple|pear),c,fruit," -

BUGS
   Edit should integrate with output from xo -l and select
   a file along with an optional address as input to an 
   additional command chain defined in the Edit command line

`)
}
