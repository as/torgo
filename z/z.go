package main

import (
	"compress/zlib"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/as/mute"
)

func main() {
	if args.d {
		deflate()
	} else {
		flate()
	}
}

func flate() {
	zw := zlib.NewWriter(os.Stdout)
	_, err := io.Copy(zw, os.Stdin)
	ck(err)
}

func deflate() {
	zr, err := zlib.NewReader(os.Stdin)
	ck(err)
	_, err = io.Copy(os.Stdout, zr)
	ck(err)
}

var args struct {
	h, q bool
	d    bool
}
var f *flag.FlagSet

func init() {
	log.SetFlags(0)
	log.SetPrefix("z: ")
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.d, "d", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
}
func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Println(`
NAME
	z - compress or decompress io

SYNOPSIS
	z [-d]

DESCRIPTION
	Z compresses or decompresses the input stream
	and writes the results to standard output

	There is one option:

	-d    Decompress

EXAMPLES
    # prints hello
	echo hello | z | z -d

`)
}
