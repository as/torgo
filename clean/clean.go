// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
//
// Walk traverses directory trees, printing visited files

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

import (
	"github.com/as/mute"
)

const Prefix = "walk: "

// sem is for dirreads, gosem is for
// goroutines. optimal number undetermined.

var (
	NGo   = 1024
	sem   = make(chan struct{}, NGo)
	gosem = make(chan struct{}, NGo)
)

var args struct {
	h, q             bool
	a, b, d bool
}
var f *flag.FlagSet

// run controls the runtime behavior during the traversal
var run struct {
	walk  func(string, func(string)) // walk function
	print func(string)               // print function
	cond  []func(string) bool        // print condition functions
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.a, "a", false, "")
	f.BoolVar(&args.b, "b", false, "")
	f.BoolVar(&args.d, "d", false, "")

	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

type Dir struct {
	Name  string
	Files []os.FileInfo
	Level int64
}

var (
	visit       map[string]bool
	rwlock      sync.RWMutex
	visitedfunc = visited
)

func isdir(f string) bool {
	fi, err := os.Stat(f)
	if err != nil {
		printerr(err)
		return false
	}
	return fi.IsDir()
}

func notdir(f string) bool {
	return !isdir(f)
}

func visited(f string) (yes bool) {
	if _, yes = visit[f]; !yes {
		rwlock.Lock()
		visit[f] = true
		rwlock.Unlock()
	}
	return
}

func print(f string) {
	for _, fn := range run.cond {
		if !fn(f) {
			return
		}
	}
	fmt.Println(f)
}

func absprint(f string) bool {
	var err error
	if !filepath.IsAbs(f) {
		if f, err = filepath.Abs(f); err != nil {
			printerr(err)
			return false
		}
	}
	print(f)
	return true
}

func main() {
	paths := f.Args()
	if args.h || args.q || len(paths) == 0 {
		usage()
		os.Exit(0)
	}

	if args.d {
		run.cond = append(run.cond, isdir)
	}

	if args.a {
		run.print = func(f string) {
			var err error
			for _, fn := range run.cond {
				if !fn(f) {
					return
				}
			}
			if f, err = filepath.Abs(f); err != nil {
				printerr(err)
				return
			}
			fmt.Println(f)
		}
	} else {
		run.print = print
	}
	var wg sync.WaitGroup
	for _, v := range paths {
		wg.Add(1)
		go func(v string) {
			if v != "-" {
				fmt.Println(clean(v))
			} else {
				in := bufio.NewScanner(os.Stdin)
				for in.Scan() {
					fmt.Println(clean(in.Text()))
				}
			}
			wg.Done()
		}(v)
	}
	wg.Wait()
}

func dirname(path string) string{
	p, err := filepath.Abs(path)
	if err != nil{
	}
	return filepath.Dir(p)
}

func base(path string) string {
	p, err := filepath.Abs(path)
	if err != nil{
	}
	return filepath.Base(p)
}
func clean(path string) string {
	p, err := filepath.Abs(path)
	if err != nil{
	}
	return filepath.Clean(p)
}

func dirs(n string, level int64) (dir *Dir) {
	f, err := ioutil.ReadDir(n)
	if err != nil || f == nil {
		printerr(err)
		return nil
	}
	return &Dir{n, f, level}
}

func usage() {
	fmt.Println(`
NAME
	clean - clean a file or directory path

SYNOPSIS
	clean [-a -m -t n] [-d|-b] [file ...]

DESCRIPTION

	There are a number of options:

	-d    Print the base directory
	-f    Print the file name
	-a    Print absolutes

EXAMPLES


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
