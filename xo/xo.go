// Copyright 2016 "as".
// This is the 'new' version of xo

package main

/*
	for (GOOS in '' windows) go build whatever
	go build xo.go
*/

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/as/argfile"
	"github.com/as/mute"
	"github.com/as/xo"
)

const (
	Prefix = "xo: "
)

var args struct {
	h, H, q bool
	r       bool
	o       bool
	i       bool
	verb    bool
	l       bool
	p       bool
	v       bool
	n, N    bool
	f       string
	x       string
	y       string
}

var count struct {
	match, unmatch int64
}

var f *flag.FlagSet

func init() {
	log.SetFlags(0)
	log.SetPrefix(Prefix)
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.verb, "verb", false, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.H, "?", false, "")
	f.BoolVar(&args.q, "q", false, "")
	f.StringVar(&args.f, "f", "", "")
	f.BoolVar(&args.i, "i", false, "")
	f.BoolVar(&args.l, "l", false, "")
	f.BoolVar(&args.o, "o", false, "")
	f.BoolVar(&args.p, "p", false, "")
	f.BoolVar(&args.n, "n", false, "")
	f.BoolVar(&args.N, "N", false, "")
	f.StringVar(&args.x, "x", `/.*\n?/`, "")
	f.StringVar(&args.y, "y", "", "")
	f.BoolVar(&args.r, "r", false, "")
	f.BoolVar(&args.v, "v", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		//printerr(err)//todo
		os.Exit(1)
	}
	if args.n && args.N {
		log.Fatalln("user error: -n and -N set")
	}
	if args.n || args.N {
		args.l = false
		args.o = false
	}
}

func xoxo(re *regexp.Regexp, in *argfile.File) {
	var buf []byte
	//
	// What is a line? sregexp will tell us.
	linedef := args.x
	if args.y != "" {
		linedef = args.y
	}

	r, err := xo.NewReaderString(in, "", linedef)
	var el, sl int

	matchfn := r.X
	if args.y != "" {
		matchfn = r.Y
	}

	nmatched := int64(0)
	for err == nil && r.Err() == nil {
		_, _, err = r.Structure()
		if err != nil && err != io.EOF {
			break
		}
		buf = matchfn()
		switch matched := re.Match(buf); {
		case !args.v && !matched:
			continue // bad: no match
		case args.v && matched:
			continue // bad: unwanted match
		}
		nmatched++
		if args.n || args.N {
			continue
		}
		if args.l {
			if r.Line1 == r.Line0 {
				fmt.Printf("%s:%d:	", in.Name, r.Line0)
			} else {
				fmt.Printf("%s:%d,%d:	", in.Name, r.Line0, r.Line1)
			}
		}
		if args.o {
			el += r.Last[1]
			sl = el - len(buf)
			fmt.Printf("%s:#%d,#%d:	", in.Name, sl, el)
		}
		if args.q {
			fmt.Printf("%q\n", string(buf))
		} else {
			fmt.Print(string(buf))
		}

		if args.p {
			fmt.Println()
		}
	}
	if args.n || args.N {
		if args.N && nmatched == 0 {
			fmt.Printf("%s", in.Name)
			fmt.Println()
		} else if args.n && nmatched != 0 {
			fmt.Printf("%s", in.Name)
			fmt.Println()
		}
	}
	count.match += nmatched
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		log.Println(err)
	}
}

func main() {
	var (
		a  = f.Args() // Remaining non-flag args
		re string     // Regexp to match against
	)
	switch nargs := len(a); {
	case args.h || args.H:
		usage()
		os.Exit(0)
	case args.f != "":
		re := "("
		sep := ""
		fd, err := os.Open(args.f)
		if err != nil {
			os.Exit(1)
		}
		defer fd.Close()
		sc := bufio.NewScanner(fd)
		for sc.Scan() {
			re += sep + sc.Text()
			sep = "|"
		}
		re += ")"
	case nargs < 1:
		usage()
		os.Exit(1)
	default:
		re = a[0]
		a = a[1:]
	}

	if !strings.Contains(re, NL) {
		// replace newline with low-precedence OR
		// AB\nC -> (BC)|(C)
		re = fmt.Sprintf("(%s)", strings.Replace(re, NL, ")|(", -1))
	}
	flags := "s"
	if args.i {
		flags += "i"
	}
	prog := regexp.MustCompile(fmt.Sprintf("(?%s)%s", flags, re))
	for fd := range argfile.Next(a...) {
		xoxo(prog, fd)
		fd.Close()
	}
	if count.match == 0 {
		os.Exit(1)
	}
}

var NL = func() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}()

func usage() {
	fmt.Println(`
NAME
	xo - Search for patterns in arbitrary structures

SYNOPSIS
	xo [flags] [-x linedef] regexp [file ...]

DESCRIPTION
	Xo scans files for pattern using regexp. By default xo
	applies regexp to each line and prints matching lines found.
	This default behavior is identical to Plan 9 grep.

	The notion of a line can be redefined by setting -x. This
	provides the ability to capture arbitrary text not
	delimited by newlines.

	The following 4 operations define text selection on the regular
	expression (re):

	   xo -x ,/re/
	   xo -x +/re/
	   xo -x -/re/
	   xo -x ;/re/

	The line definition can chain an arbitrary quantity of operations

	   xo -x ,/re/+/re2/,/re3/-/re4/

	The default linedef is simply: xo -x /./,/\n/

	Xo reads lines from stdin unless a file list is given. If '-' is 
	present in the file list, xo reads a list of files from stdin 
	instead of treating stdin as a file.

FLAGS
	Linedef:

	-x linedef	Redefine a line based on linedef
	-y linedef	The negation of linedef becomes linedef

	Regexp:

	-v regexp   Reverse. Print the lines not matching regexp
	-f file     Read a list of disjunctive regexps to match from file

	Tagging:

	-o  Mark each match with its file name and rune offset
	-l  Mark each match with its file name and line offset
	-n  Print only the file name containing a match
	-N  Print only the file name not containing a match
	-p  Print extra new line after a match

	-q  Quote (escape) all matched output (may be removed)

EXAMPLE
	Search file.txt for apple, output matching lines
          xo -l apple file.txt
	Search directory for files containing apple, print filenames
          walk -f | xo -N apple -
	Search cpp files for camelcase; tag output with file:line
          walk -f | xo "\.cpp\n" | xo -l [A-Z][a-z]+[A-Z][a-z]+ -
        
	The next examples use this help section, so
          xo -h > help.txt
	Search for words starting with "a", print matching words
          xo -p -x "/[A-Za-z]+/" "^a" help.txt
	Print the description section
          xo -l -x "/[A-Z]+\n/,/\n\n[A-Z]+/" DESC help.txt
	Select fields by whitespace, print each field
          xo -p -l -y "/[ \t]/"  . help.txt

BUGS
	It's difficult to understand -x and -y
`)
}
