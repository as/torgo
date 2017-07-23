// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
//
// Walk traverses directory trees, printing visited files

package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"bytes"
	"golang.org/x/image/bmp"
	"github.com/as/mute"
	"github.com/as/screen"
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("rec: ")
}

// sem is for dirreads, gosem is for
// goroutines. optimal number undetermined.

var args struct {
	h, q bool
	r    string
	w    int
}
var f *flag.FlagSet

var run struct {
	walk  func(string, func(string))
	cond  []func(string) bool
	print func(string)
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.StringVar(&args.r, "r", "", "")
	f.IntVar(&args.w, "w", 0, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}
}
func main() {
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
	r := image.Rect(0, 0, 1080, 1080)
	if args.r != "" {
	}
	img, err := screen.Capture(args.w, args.w, r)
	if err != nil {
		log.Fatalln(err)
	}
	b := new(bytes.Buffer)
	bmp.Encode(b, img)
	io.Copy(os.Stdout, b)
}

func usage() {
	fmt.Println(`
NAME
	rec - capture the screen or window

SYNOPSIS
	rec [-r x0,y0,x1,y1] [-w window]

DESCRIPTION
	Rec captures a window with the clipping
	rectangle bounded by x0,x1,y0,y1. If window
	is zero, the entire screen is captured. The
	resulting bitmap is written to standard output

	Options:

	-r    Clipping rectangle bounds
	-w    Window handle

EXAMPLES
	

`)
}
