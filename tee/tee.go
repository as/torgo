// Copyright 2015 "as". All rights reserved. The program and its corresponding
// gotools package is governed by an MIT license.
//
// Tee reads from stdin and writes to stdout, and any number of files. Tee
// can append its output to the files with [ -a ].

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

import (
	"github.com/as/mute"
)

const Prefix = "tee: "

var args struct {
	a, h, q bool
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.a, "a", false, "")

	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

func main() {
	if args.h || args.q {
		usage()
		os.Exit(0)
	}

	var err error

	nfiles := len(f.Args())
	writer := make([]io.Writer, nfiles+1)
	writer[0] = os.Stdout

	open := func(f string) (io.Writer, error) {
		if args.a {
			perm := os.O_WRONLY | os.O_APPEND | os.O_CREATE
			return os.OpenFile(f, perm, 0666)
		}
		return os.Create(f)
	}

	i := 1
	for _, fname := range f.Args() {
		writer[i], err = open(fname)
		if t, ok := writer[i].(io.WriteCloser); ok {
			defer t.Close()
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		i++
	}

	mw := io.MultiWriter(writer...)
	io.Copy(mw, os.Stdin)
}

func usage() {
	fmt.Println(`
NAME
	tee - read from stdin, write to stdout and given files

SYNOPSIS
	tee [-a] [file ...]

DESCRIPTION
	Tee reads from stdin and writes to stdout and given files.  

	-a	Append to the files instead of truncating them
`)
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}
