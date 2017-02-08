package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/as/mute"
)

func main() {
	trmap := make(map[byte]byte)
	a := f.Args()

	var src, dst string
	switch n := len(a); n {
	case 0:
		// No arguments provided
	case 1:
		src = a[0]
		for i := range src {
			trmap[src[i]] = src[i]
		}
	default:
		src, dst = a[0], a[1]
		for i := range src {
			trmap[src[i]] = dst[i%len(dst)]
		}
	}

	i := bufio.NewReader(os.Stdin)
	o := bufio.NewWriter(os.Stdout)
	for {
		src, err := i.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		dst, ok := trmap[src]
		if !ok {
			o.WriteByte(src)
			continue
		}
		if args.d {
			continue
		}
		o.WriteByte(dst)
	}
	o.Flush()
}

func remove(p []byte, i int) []byte {
	copy(p[i:], p[i+1:])
	return p[:len(p)-1]
}

const Prefix = "tr: "

var args struct {
	h, q bool
	i    bool
	s    bool
	c    bool
	d    bool
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.i, "i", false, "")
	f.BoolVar(&args.s, "s", false, "")
	f.BoolVar(&args.c, "c", false, "")
	f.BoolVar(&args.d, "d", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
}

var f *flag.FlagSet

func usage() {
	fmt.Println(`
NAME
	tr - translate character set

SYNOPSIS
	tr [-d] set1 [set2]

DESCRIPTION
	Tr reads from stdin and maps bytes in set1 to bytes in set2.
	The -d flag causes bytes found in set1 to be deleted from the
	input instead of being remapped.

FLAGS
	-d,   Delete bytes in set1 from input

EXAMPLE

	# Type like a Microsoft Win32 kernel developer
	echo MyVeryOwnBufferClass | tr -d aeiouAEIOU

	# Make good english from include
	echo '#include <yolo>' | tr -d '#<\/>'

BUGS
	Missing 

`)
}

func no(err error) {
	if err != nil {
		log.Fatalln(err)
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

var debug = false

func debugerr(v ...interface{}) {
	if debug {
		printerr(v)
	}
}
