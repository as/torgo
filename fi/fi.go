// Copyright 2016 "as".

package main

/*
	for (GOOS in '' windows) go build whatever
	go build xo.go
*/

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/as/mute"
)

const (
	Prefix = "fi: "
	Debug  = false // true false
)

var args struct {
	h, q, v    bool
	s, m, p, c bool
	d          bool
}

var count struct {
	match, unmatch int64
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.c, "c", false, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.d, "d", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.s, "s", false, "")
	f.BoolVar(&args.m, "m", false, "")
	f.BoolVar(&args.p, "p", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		os.Exit(1)
	}
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
}

/*
	CMD ARG
	m + 2006.01.02 15:04:05
	m - 2006.01.02 15:04:05
	m + duration
	m - duration
	m 2006.01.02 15:04:05+duration
	m 2006.01.02 15:04:05-duration
	m 2006.01.02 15:04:05,2006.01.02 15:04:05
	m 2006,2007
*/

func main() {
	sc := bufio.NewScanner(os.Stdin)
	var sprintfn = make([]func(s string, fi os.FileInfo) string, 0)
	sprintfn = append(sprintfn, func(s string, fi os.FileInfo) string {
		return fmt.Sprintf("%s\n", s)
	})
	if args.s {
		sprintfn = append(sprintfn, func(s string, fi os.FileInfo) string {
			return fmt.Sprintf("%d\t", fi.Size())
		})
	}
	if args.d {
		sprintfn = append(sprintfn, func(s string, fi os.FileInfo) string {
			return fmt.Sprintf("%s\t", time.Now().Sub(fi.ModTime())/time.Second*time.Second)
		})
	}
	if args.m {
		sprintfn = append(sprintfn, func(s string, fi os.FileInfo) string {
			return fmt.Sprintf("%s\t", fi.ModTime().Format("2006.01.02 15:04:05"))
		})
	}
	if args.p {
		sprintfn = append(sprintfn, func(s string, fi os.FileInfo) string {
			return fmt.Sprintf("%s\t", fi.Mode())
		})
	}
	sum := int64(0)
	for sc.Scan() {
		fi, err := os.Stat(sc.Text())
		if err != nil && args.v {
			printerr(err)
		}

		s := ""
		for i := len(sprintfn) - 1; i >= 0; i-- {
			s += sprintfn[i](sc.Text(), fi)
			sum += fi.Size()
		}
		fmt.Print(s)
	}
	if args.s && args.c {
		fmt.Printf("%d total\n", sum)
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
	fi - print file information

SYNOPSIS
	walk -f | fi [-s -c -f -d -p] 

DESCRIPTION
	fi reads filenames from stdin and prints file info

    There are a number of options:

 	-s  Print file size
	-c  Print cumulative statistics
	-f  Print modified time
	-d  Print duration since last modified
	-p  Print permissions

EXAMPLES
	Display the size of each file under the directory,
	then print the total size of all files

	  walk -f mink/ | fi -s -c
`)
}
