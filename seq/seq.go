// Copyright 2015 "as"

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/as/mute"
)

var f *flag.FlagSet
var args struct {
	h, q bool
	f    string
}

func init() {
	log.SetFlags(0)
	log.SetPrefix("seq: ")
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.StringVar(&args.f, "f", `%d\n`, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}
	if args.h || args.q || len(f.Args()) < 1 {
		usage()
		if len(f.Args()) < 1 {
			os.Exit(1)
		}
		os.Exit(0)
	}
}

func main() {
	a := f.Args()
	i := atoi(a[0])
	j := 0
	if len(a) > 1 {
		j = atoi(a[1])
	} else {
		j = i
		i = 0
	}
	k := 1
	if len(a) > 2 {
		k = atoi(a[2])
	}
	fm, err := strconv.Unquote(`"` + args.f + `"`)
	if err != nil {
		log.Fatalln("unquote:", err)
	}

	// generate the sequence {i,i+k,...,j}
	// using format string
	for ; i <= j; i += k {
		fmt.Printf(fm, i)
	}
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		log.Fatalln(err)
	}
	return i
}
func usage() {
	fmt.Println(`
NAME
    seq - generate numeric sequence

SYNOPSIS
    seq [-f fmt] i j [k]

DESCRIPTION
    Seq outputs the sequence {i, i+k, ..., j} according
    to the output format (default "%d\n"). An empty k sets k=1,
    printing a line seperated list of the numbers i-j inclusive.
	
FORMATS
    Types
	
    %d     Integer
    %f     Float (64 bits)
    %q     Quoted number
    %x     Hexadecimal number
    %c     A byte's value
    
    Widths (example only shows %d)
    
    %8d    Int, 8 leading spaces
    %08d   Int, 8 leading zeroes
    %-8d   Int, 8 spaces, left justified 

EXAMPLE
    Print 0 to 100, skipping by 3
      seq 0 100 3
    
    The first 10 bytes of ASCII table
      seq -f "%q\n" 0 9

    The binary values of the ASCII table
      seq -f "%c" 0 9
`)
}
