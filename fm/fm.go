// Copyright 2015 "as". All rights reserved. The program and its corresponding
// gotools package is governed by an MIT license.
/*
	for (GOOS in '' windows) go build whatever
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/as/mute"
)

const (
	Prefix    = "fm: "
	MaxBuffer = 65536
	Debug     = false // true false
)

var args struct {
	h, q bool
	r    bool
	k    string
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)

	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.StringVar(&args.k, "k", "", "")
	f.BoolVar(&args.r, "r", false, "")
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
	a := f.Args() // Remaining non-flag args
	sc := bufio.NewScanner(os.Stdin)
	format := strings.Join(a, " ")

	for sc.Scan() {
		ln := sc.Text()
		fmt.Printf(format, ln)
		if !args.r {
			fmt.Println()
		}
	}
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func debugerr(v ...interface{}) {
	if Debug {
		printerr(v)
	}
}

func usage() {
	fmt.Println(`
NAME
	fm - format

SYNOPSIS
	fm [verb...]

DESCRIPTION
	Fm reads from standard input and prints each
	line in the format described by verb

	-r	Raw: don't insert newline after each read

EXAMPLE
	echo world | fm hello %s
	echo google | fm ping %s.com
	echo 3 | fm %05d

BUGS

`)
}
