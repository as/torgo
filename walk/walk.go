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
	h, q bool
	a, b, c, d, f, m bool
	t int64
}
var f *flag.FlagSet


var run struct {
	walk func(string, func(string))
	cond  []func(string) bool
	print func(string)
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.b, "b", false, "")
	f.BoolVar(&args.a, "a", false, "")
	f.BoolVar(&args.d, "d", false, "")
	f.BoolVar(&args.c, "c", false, "")
	f.BoolVar(&args.f, "f", false, "")
	f.BoolVar(&args.m, "m", false, "")
	f.Int64Var(&args.t, "t", 1024*1024*1024, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

type Dir struct {
	Name string
	Files []os.FileInfo
	Level int64
}

var (
	visit map[string]bool
	rwlock sync.RWMutex
	visitedfunc = visited
)

func isdir(f string) bool {
	fi, err := os.Stat(f)
	if err != nil {
		printerr(err); return false
	}
	return fi.IsDir()
}

func notdir(f string) bool {
	return !isdir(f)
}

func init() {
	visit = make(map[string]bool)
	if !args.m {
		visitedfunc = func(f string) (bool) { return false }
	}
}

func visited(f string) (yes bool) {
	if _, yes = visit[f]; !yes {
		rwlock.Lock()
		visit[f] = true
		rwlock.Unlock()
	}
	return
}

func print(f string)  {
	for _, fn := range run.cond {
		if !fn(f) {return}
	}
	fmt.Println(f)
}

func absprint(f string) bool{
	var err error
	if !filepath.IsAbs(f){
		if f, err = filepath.Abs(f); err != nil {
			printerr(err); return false
		}
	}
	print(f)
	return true
}

func main() {
	if args.h || args.q {
		usage(); os.Exit(0)
	}
	if args.d && args.f {
		printerr("bad args: -d and -f")
		os.Exit(1)
	}

	run.walk = dft
	if args.d { run.cond = append(run.cond, isdir)}
	if args.f { run.cond = append(run.cond, notdir)}
	if args.b { run.walk = bft }

	paths := f.Args()
	if len(paths) == 0 {
		paths = []string{"."} // Current working dir
		run.print = func(f string) {
			for _, fn := range run.cond {
				if !fn(f) {
					return
				}
			}
			if len(f) > 2 {
				fmt.Println(f[2:])
			}
		}
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
				printerr(err); return
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
				run.walk(v, run.print)
			} else {
				in := bufio.NewScanner(os.Stdin)
				for in.Scan() {
					run.walk(in.Text(), run.print)
				}
			}
			wg.Done()
		}(v)
	}
	wg.Wait()
}

// dft performs a depth-first traversal of the file f
func dft(name string, fn func(string)) {
	listch := make(chan string, NGo)
	var wg sync.WaitGroup
	wg.Add(1)
	go dft1(name, &wg, 0, listch)
	go func() {
		wg.Wait()
		close(listch)
	}()
	for d := range listch {
		run.print(d)
	}
}

func dft1(nm string, wg *sync.WaitGroup, deep int64, listch chan<- string) {
	defer wg.Done()
	d := dirs(nm, deep)
	if d == nil {
		return
	}
	for _, f := range d.Files {
		kid := filepath.Join(d.Name, f.Name())
		if visitedfunc(kid) {
			continue
		}
		if f.IsDir() && deep < args.t {
			wg.Add(1)
			go dft1(kid, wg, deep+1, listch)
		}
		listch <- kid
	}
}

// bfr performs a breadth-first traversal of the file f
func bft(name string, fn func(string)) {
	list := make(chan *Dir, NGo)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		list <- dirs(name, 1)
		wg.Wait()
		close(list)
	}()
	for dir := range list {
		if dir == nil {
			continue
		}
		gosem <- struct{}{}
		go func(d *Dir) {
			defer wg.Done()
			if d.Level > args.t {
				return
			}
			for _, f := range d.Files {
				kid := filepath.Join(d.Name, f.Name())
				run.print(kid)
				if f.IsDir() && !visitedfunc(kid) {
					wg.Add(1)
					list <- dirs(kid, d.Level+1)
				}
			}
			<- gosem 
		}(dir)
	}
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
	walk - traverse a list of files

SYNOPSIS
	walk [-b -a -m -t n] [-d|-f] [file ...]

DESCRIPTION
	Walk walks the named file list and prints each name
	to standard output. A directory in the file list is
	a file list. The file "-" names standard input as a
	file list of line-seperated file names.

	There are a number of options:

	-d    Print directories only 
	-f    Print files only 
	-a    Print absolute paths

	-b    Use breadth-first traversal
	-t n  n specifies the traversal limit
	-m    Memorize and ignore visited files 

	Walk will not follow symlinks

EXAMPLES
	Walk the first four levels down the directory
	tree, look for "mobius", and walk all files in
	the mobius directories.

	walk -d -t 4 | grep -i mobius | walk -f -

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
