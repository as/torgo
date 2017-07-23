// Copyright 2016 "as". All rights reserved. The program and its corresponding
// gotools package is governed by an MIT license.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/as/mute"
	"github.com/as/xo"
)

const (
	Prefix = "sort: "
)

type Dot []byte

type Keys struct {
	Dots   []Dot
	lessfn func(i, j int) bool
	o      *regexp.Regexp
}

func (d Dot) String() string               { return string(d) }
func (d Dot) O(re *regexp.Regexp) [][]byte { return re.FindAll([]byte(d), -1) }

func (k Keys) Len() int {
	return len(k.Dots)
}
func (k *Keys) Less(i, j int) bool {
	if k.lessfn == nil {
		return k.LessDefault(i, j)
	}
	return k.lessfn(i, j)
}
func (k *Keys) Swap(i, j int) {
	k.Dots[i], k.Dots[j] = k.Dots[j], k.Dots[i]
}

func (k *Keys) LessO(i, j int) bool {
	a, b := k.Dots[i], k.Dots[j]
	ao, bo := a.O(k.o), b.O(k.o)
	if len(ao) == 0 {
		return false
	}
	if len(bo) == 0 {
		return true
	}
	a0 := ao[0]
	b0 := bo[0]

	if args.i {
		a0, b0 = bytes.ToLower(a0), bytes.ToLower(b0)
	}
	return cmpfn(a0, b0) == -1
}

func (k *Keys) LessDefault(i, j int) bool {
	a0, b0 := k.Dots[i], k.Dots[j]
	if args.i {
		a0, b0 = bytes.ToLower(a0), bytes.ToLower(b0)
	}
	return cmpfn(a0, b0) == -1
}

var cmpfn = bytes.Compare

func ncmp(a, b []byte) int {
	fa := strings.Fields(string(a))
	if len(fa) == 0 {
		return 1
	}
	fb := strings.Fields(string(b))
	if len(fb) == 0 {
		return -1
	}
	A, err := strconv.Atoi(string(fa[0]))
	if err != nil {
		return 1
	}
	B, err := strconv.Atoi(string(fb[0]))
	if err != nil {
		return -1
	}
	if A < B {
		return -1
	}
	if A == B {
		return 0
	}
	return 1
}

func main() {
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
	if args.n {
		cmpfn = ncmp
	}
	if args.y != "" {
		args.x = args.y
	}
	r, err := xo.NewReaderString(os.Stdin, "", args.x)
	no(err)

	K := &Keys{o: regexp.MustCompile(args.k)}

	fn := r.X
	if args.y != "" {
		fn = r.Y
	}
	for {
		_, _, err = r.Structure()
		if err != nil && err != io.EOF {
			log.Fatalln(err)
		}
		data := append([]byte{}, fn()...)
		K.Dots = append(K.Dots, Dot(data))
		if err != nil {
			break
		}
	}
	if args.k != "" {
		K.lessfn = K.LessO
	}

	if args.r {
		sort.Stable(sort.Reverse(K))
	} else {
		sort.Stable(K)
	}
	for _, v := range K.Dots {
		fmt.Printf("%s", v)
	}
	a := f.Args() // Remaining non-flag args
	a = a
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

var args struct {
	h, q bool
	r    bool
	i    bool
	n    bool
	k    string
	x    string
	y    string
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)

	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.i, "i", false, "")
	f.BoolVar(&args.n, "n", false, "")
	f.StringVar(&args.k, "k", "", "")
	f.StringVar(&args.x, "x", "/./,/(\n|$)/", "")
	f.StringVar(&args.y, "y", "", "")
	f.BoolVar(&args.r, "r", false, "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}

}

var f *flag.FlagSet

func usage() {
	fmt.Println(`
NAME
	Sort - sort selections

SYNOPSIS
	sort [ -x op/re/ ...] [-k re] [-i -r]

DESCRIPTION
	Sort reads from standard input and sorts selections
	according to -k re. Re is a regexp defaulting to the
	entire selection.

	The -x option controls what is selected. It uses xo's
	grammar to select an arbitrary string of text. The
	selections are sorted after all the input is read.

	Sort is guaranteed to choose a stable sorting algorithm

FLAGS
	-k re      Use 're' as a regexp to define the sort key
	-i         Insensitive case
	-r         Reverse the sense of comparisons
	-n         Sort numerically

    -x op/re/  Select structure with xo grammar instead of a line
    -y op/re/  Inverse selection with respect to -x op/re/

EXAMPLE
   Sort this manual by heading names:
      sort -h | sort -x "/[A-Z]/,/\n[A-Z]/-/./" -k "^[A-Z]+"

   Sort Go functions by name
      sort -x '/func/,/\n\}\n/" < sort.go

   Sort a string containing the first few letters of the alphabet
      echo dAcbaZe | sort -i -x /./ -k .

   Sort a title-case phrase
      echo WhoAmI  | sort  -x "/[A-Z][a-z]*/"  -k .

BUGS
	When -x is used, there should be two -k keys, one for what is
	selected and another for what is missed (as in xo -y, which
	isn't implemented yet).

`)
}
