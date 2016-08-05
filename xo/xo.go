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
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"
)

import(
	"github.com/as/argfile"
	"github.com/as/mute"
)

const (
	Prefix    = "xo: "
	MaxBuffer = 1024 * 1024 * 512
	Debug     = false // true false
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
	f       string
	x       string
	y       string
	s       string
}

var count struct {
	match, unmatch int64
}

var f *flag.FlagSet

func init() {
	verb = 0
	if args.verb {
		verb = 5
	}
}

func init() {
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
	f.StringVar(&args.x, "x", `/.*\n/`, "")
	f.StringVar(&args.y, "y", "", "")
	f.BoolVar(&args.r, "r", false, "")
	f.BoolVar(&args.v, "v", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		//printerr(err)//todo
		os.Exit(1)
	}
}

func xo(re *regexp.Regexp, in *argfile.File)  {
	var buf []byte
	//
	// What is a line? sregexp will tell us.
	linedef := args.x
	if args.y != "" {
		linedef = args.y
	}

	r, err := sregexp(in, linedef)
	var el, sl int
	for err == nil && r.err == nil {
		buf, _, err = r.Structure()
		if err != nil && err != io.EOF {
			fmt.Println("asdf",err)
			break
		}

		switch matched := re.Match(buf); {
		case !args.v && !matched:
			continue // bad: no match
		case args.v && matched:
			continue // bad: unwanted match
		default:
			count.match++
		}

		if args.l {
			if r.line1 == r.line0 {
				fmt.Printf("%s:%d:	", in.Name, r.line0)
			} else {
				fmt.Printf("%s:%d,%d:	", in.Name, r.line0, r.line1)
			}
		}
		if args.o {
			el += r.last[1]
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
	if err == io.EOF {
		err = nil
	}
	if err != nil {
		printerr(err)
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
		data, err := ioutil.ReadFile(args.f)
		if err != nil {
			//printerr(err)
			os.Exit(1)
		}
		re = string(data)
	case nargs < 1:
		usage()
		os.Exit(1)
	default:
		re = a[0]
		a = a[1:]
	}

	oneline := strings.Index(re, NL) == -1
	if !oneline {
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
		xo(prog, fd)
		fd.Close()
	}
	if count.match == 0 {
		os.Exit(1)
	}
}

// sregexp returns a structural regular
// expression reader compiled for the
// structure string s. S must be in the
// form of AddrN constants below. The
// syntax is similar to the Sam and Acme
// text editors for Plan 9
//
func sregexp(in io.Reader, s string) (*Xo, error) {
	cmd, err := parseaddr(s)
	if err != nil {
		return nil, err
	}
	if len(cmd) == 0{
		return nil, fmt.Errorf("sregexp: empty cmd")
	}
	return NewReader(in, cmd...), err
}

var NL = func() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}()

func moribound(r *bufio.Reader) (err error) {
	defer r.UnreadRune()
	if _, _, err = r.ReadRune(); err != nil {
		verb.Println("moribound", err)
		return err
	}
	return nil
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func debugerr(v ...interface{}) {
	if Debug {
		printerr(v)
	}
}

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
	delimited by lines.

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
	present in the file list, xo reads a list of files from
	stdin instead of treating stdin as a file.

FLAGS
	Linedef:

	-x linedef	Redefine a line based on linedef
	-y linedef	The negation of linedef becomes linedef

	Regexp:

	-v regexp	Reverse. Print the lines not matching regexp
	-f file     File contains a list of regexps, one per line
				the newline is treated as an OR

	Tagging:

	-o  Preprend file:rune,rune offsets
	-l	Preprend file:line,line offsets
	-L  Print file names containing no matches
	-p  Print new line after every match

	-q  Quote (escape) all output (may be removed)


EXAMPLE
	# Examples operate on this help page, so
	xo -h > help.txt

	# Print the DESCRIPTION section from this help
	xo -l -x '/[A-Z]+\n/,/\n\n[A-Z]+/' DESC help.txt


BUGS
	On a multi-line match, xo -l prints the offset
	of the final line in that match.

	It's difficult to understand -x from this manual.

	As of Jul 31 2016, the following do not work:

	1) xo -y
	2) Line and byte offsets

	
`)
}
