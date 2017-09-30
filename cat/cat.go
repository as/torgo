package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/as/mute"
)

const Prefix = "cat: "

var args struct {
	h, q bool
}
var f *flag.FlagSet

func init() {
	log.SetPrefix(Prefix)
	log.SetFlags(0)
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	err := mute.Parse(f, os.Args[1:])
	ck(err)
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
}

func main() {

	if len(f.Args()) == 0 {
		cat("-")
	} else {
		for _, f := range f.Args() {
			cat(f)
		}
	}
}

func cat(f string) {
	var (
		in  io.ReadCloser = os.Stdin
		err error
	)
	if f != "-" {
		in, err = os.OpenFile(f, os.O_RDONLY, 0666)
		if err != nil {
			ck(err)
			return
		}
		defer in.Close()
	}
	io.Copy(os.Stdout, bufio.NewReader(in))
}

func ck(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func usage() {
	fmt.Println(`
NAME
	cat - catenate files

SYNOPSIS
	cat [file...]

DESCRIPTION
	Cat writes the contents of each named file to stdout

EXAMPLE
	cat > file1
	cat file1 file2 > file3

BUGS
	This leaves you with an empty file1:
	cat file1 file2 > file1
`)
}
