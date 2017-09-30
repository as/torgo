// Copyright 2015 "as". All rights reserved. The program and its corresponding
// gotools package is governed by an MIT license.
//
// Wc counts lines, words, bytes, and runes

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"sync"
	"unicode/utf8"
)

import (
	"github.com/as/argfile"
	"github.com/as/mute"
)

const (
	Prefix     = "wc: "
	BufferSize = 2 ^ 16
)

var args struct {
	lines, words, runes, chars, x bool
	h, q                          bool
}

func (t tally) String() string {
	return t.name
}

type tally struct {
	lines, words, runes, chars, x uint64
	name                          string
}

// add adds tally2 to tally t
func (t *tally) add(t2 *tally) {
	t.lines += t2.lines
	t.words += t2.words
	t.runes += t2.runes
	t.chars += t2.chars
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.lines, "l", false, "")
	f.BoolVar(&args.words, "w", false, "")
	f.BoolVar(&args.chars, "c", false, "")
	f.BoolVar(&args.runes, "r", false, "")
	f.BoolVar(&args.x, "x", false, "")

	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
	if args.x {
		return
	}
	if !args.lines {
		if !args.words {
			if !args.chars {
				if !args.runes {
					args.lines = true
					args.words = true
					args.chars = true
				}
			}
		}
	}

}

func main() {
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
	var (
		wg      sync.WaitGroup
		nfiles  int
		totals  = &tally{name: "totals"}
		countfn = count
		donec   = make(chan bool)
	)
	if args.x {
		countfn = countx
	}
	report := make(chan *tally, 1)
	go func() {
		for t := range report {
			if args.x {
				fmt.Printf("%10d", t.x)
				fmt.Printf(" %s\n", t)
				return
			}
			if args.lines {
				fmt.Printf("%10d", t.lines)
			}
			if args.words {
				fmt.Printf("%10d", t.words)
			}
			if args.chars {
				fmt.Printf("%10d", t.chars)
			}
			if args.runes {
				fmt.Printf("%10d", t.runes)
			}
			fmt.Printf(" %s\n", t)
		}
		donec <- true
	}()
	for v := range argfile.Next(f.Args()...) {
		wg.Add(1)
		nfiles++
		go func(f *argfile.File) {
			defer wg.Done()
			t, err := countfn(f)
			f.Close()
			if err != nil {
				printerr(err)
				return
			}
			totals.add(t) // update totals
			t.name = f.Name
			report <- t
		}(v)
	}
	wg.Wait()
	if nfiles > 1 {
		report <- totals
	}
	close(report)
	<-donec
}

func count(in io.Reader) (*tally, error) {
	t := new(tally)
	r := bufio.NewScanner(in)
	var spaces = []byte{' ', '	'}
	//splitfn := func(data []byte, atEOF bool)(advance int, token []byte, err error)
	for r.Scan() {
		l := r.Bytes()
		t.chars += uint64(len(l))
		t.words += uint64(bytes.Count(l, spaces))
		t.runes += uint64(utf8.RuneCount(l))
		t.lines++
	}
	return t, nil
}

func countx(in io.Reader) (*tally, error) {
	var err error
	megabuffer := make([]byte, 1024*1024)
	t := new(tally)
	for err == nil {
		_, err = in.Read(megabuffer)
		t.x++
	}
	return t, nil
}

func efatal(err error) {
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

func eprint(err error) bool {
	if err != nil {
		printerr(err)
		return true
	}
	return false
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func usage() {
	fmt.Println(`
NAME
	wc - word count

SYNOPSIS
	wc [ -lwrc | -x ] [ file ... ]

DESCRIPTION
	Wc counts lines, words, runes and characters in the
	file list provided. An empty file list implies stdin.

	The default behavior is equal to: wc -lwc

	The experimental -x flag counts the number of calls
	to read, relying on an assumption that the each read
	is a meaningful iota of input split with xo.
	
	If - is named as a file, standard input is treated as a
	list of files. If no files are named, stdin is treated
	as the file.

NOMENCLATURE
	A Rune is a UTF-8 character. UTF-8 is the de-facto
	text encoding on Plan9 and all modern operating systems
	except Microsoft Windows.

EXAMPLE
	Count runes and characters in /etc/hosts
	wc -r -c /etc/hosts

	Count lines, words and chars in file1 and file2:
	wc file1 file2

BUGS
	Doesn't count broken runes
`)
}
