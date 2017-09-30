// Copyright 2015 "as".
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/as/mute"
)

const (
	Prefix = "cp: "
)

var args struct {
	h, q bool
	a    bool
	v    bool
	p    bool
	f    bool
}

var (
	wg     sync.WaitGroup
	ticket chan struct{}
)

func init() {
	ticket = make(chan struct{}, 1024)
}
func semaquire() {
	ticket <- struct{}{}
}
func semrelease() {
	<-ticket
}

func main() {
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
	a := f.Args() // Remaining non-flag args
	if len(a) == 0 {
		printerr("usage: echo src | cp [-a -v -p] dst")
		os.Exit(1)
	}
	cp(a[0])
}

func cp(dir string) {
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		suf := in.Text()
		switch {
		case suf == "":
		case suf[0:1] == "/":
			printerr(fmt.Sprintf("ignored: absolute path: %s\n", suf))
		default:
			src := clean(suf)
			var dst string
			if args.f {
				dst = clean(dir + "/" + filepath.Base(src))
			} else {
				dst = clean(dir + "/" + src)
			}
			wg.Add(1)
			semaquire()
			printerr("docp", dst, src)
			go docp(dst, src)
		}
	}
	if err := in.Err(); err != nil {
		printerr(err)
	}
	wg.Wait()
}
func docp(dst, src string) (n int64, err error) {
	defer semrelease()
	defer wg.Done()

	var buf [2 << 15]byte
	if args.v {
		fmt.Println(dst)
	}
	fds, err := os.Open(src)
	fatal(err)
	defer fds.Close()
	mkdir(dirof(dst))
	fdd, err := os.Create(dst)
	defer fdd.Close()
	fatal(err)
	return io.CopyBuffer(fdd, fds, buf[:])
}
func clean(dir string) string {
	dir = filepath.ToSlash(dir)
	dir = filepath.FromSlash(dir)
	return filepath.Clean(dir)
}
func dirof(file string) string {
	file = clean(file)
	return filepath.Dir(file)
}
func mkdir(dir string) error {
	return os.MkdirAll(clean(dir), 0777)
}
func readable(file string) bool {
	fd, err := os.Open(clean(file))
	defer fd.Close()
	if err != nil {
		printerr(err) //TODO
	}
	return true
}
func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}
func fatal(err error) {
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}
func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func usage() {
	fmt.Println(`
NAME
	cp - copy files

SYNOPSIS
	walk -f relativesrc | cp dest/

DESCRIPTION

	THIS IS NOT UNIX CP. YOU WILL DESTROY EVERYTHING.

	Cp reads a list of relative file names from stdin and
	copies each file to dest. Directories are created as
	needed and all files in dest are overwritten.

	-d  (NOT YET IMPLEMENTED) dont make directories
	-p	(NOT YET IMPLEMENTED) preserve existing files
	-v	show progress to stdout
	-f	flatten (dest will be a flat folder structure)

BUGS
	With -f, only one file with the same name can exist in
	the dest directory.

`)
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.a, "a", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.f, "f", false, "")
	f.BoolVar(&args.a, "p", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}
