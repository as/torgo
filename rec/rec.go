// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/as/ms/screen"
	"github.com/as/mute"
	"golang.org/x/image/bmp"
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
	p    int
	f    int
}
var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.StringVar(&args.r, "r", "", "")
	f.IntVar(&args.w, "w", 0, "")
	f.IntVar(&args.p, "p", 0, "")
	f.IntVar(&args.f, "f", 0, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}
}
func mustatoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalln(err)
	}
	return i
}
func main() {
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
	r := image.Rect(0, 0, 1024, 768)
	if args.r != "" {
		x := strings.Split(args.r, ",")
		if len(x) != 4 {
			log.Fatalln("-r must be in the form x0,y0,x1,y1")
		}
		r.Min.X = mustatoi(x[0])
		r.Min.Y = mustatoi(x[1])
		r.Max.X = mustatoi(x[2])
		r.Max.Y = mustatoi(x[3])
	} else {
		if args.w != 0 {
			r = rect(args.w)
		}
		log.Println(r)
	}
	if r == image.ZR {
		log.Fatalln("zero rectangle")
	}
	bytec := make(chan *bytes.Buffer, 8)
	imgc := make(chan image.Image, 8)
	donec := make(chan bool)
	go func() {
		for b := range bytec {
			io.Copy(os.Stdout, b)
		}
		donec <- true
	}()
	go func() {
		for img := range imgc {
			b := new(bytes.Buffer)
			bmp.Encode(b, img)
			bytec <- b
		}
		close(bytec)
	}()
	ir := time.Duration(0)
	if args.f != 0 {
		ir = time.Second / time.Duration(args.f)
	}
	tc := time.NewTimer(ir)
	for tc != nil {
		<-tc.C
		if args.f == 0 {
			tc = nil
		} else {
			tc = time.NewTimer(ir)
		}
		img, err := screen.Capture(args.w, args.w, r)
		if err != nil {
			log.Fatalln(err)
		}
		imgc <- img
		if args.f == 0 {
			close(imgc)
		}
	}
	<-donec
}

func usage() {
	fmt.Println(`
NAME
	rec - capture the screen or window

SYNOPSIS
	rec [-r x0,y0,x1,y1] [-w wid0 ... widN] [-p pid0 ... pidN]

DESCRIPTION
	Rec captures a window(s) with the clipping
	rectangle bounded by x0,x1,y0,y1. If no wids or pids
	are given, the entire screen is captured with the clipping
	rectangle given by -r. The resulting bitmap is written to
	standard output.

	The -f option emits a bitmap stream to stdout with the
	specified bitrate

	Options:

	-r       Clipping rectangle bounds
	-w wid   Window handle
	-p pid   Process ID
	-f       Frame rate (default 0/one image)

EXAMPLES
    Capture a 640x480 screenshot starting at (0,0)
      rec -r 0,0,640,480 > x.bmp

    Capture a 640x480 RGB bitmap stream at 30fps and display it
      rec -r 0,0,640,480 -f 30 | page

BUGS
	The video stream is an uncompressed rgb stream instead of
	compressed Y'CrCb
`)
}

var (
	u32            = syscall.MustLoadDLL("user32.dll")
	k32            = syscall.MustLoadDLL("kernel32.dll")
	pGetWindowRect = u32.MustFindProc("GetWindowRect")
)

// Because it's 32-bit
type Point struct {
	x, y int32
}
type Rect struct {
	min, max Point
}

func rect(wid int) image.Rectangle {
	var r Rect
	rp := uintptr(unsafe.Pointer(&r))
	e, _, _ := pGetWindowRect.Call(uintptr(uint32(wid)), rp)
	if e == 0 {
		return image.ZR
	}
	goodr := image.Rect(int(r.min.x), int(r.min.y), int(r.max.x), int(r.max.y))
	return goodr
}
