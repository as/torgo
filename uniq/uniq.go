// Copyright 2015 "as". Go license.
// TODO: This is the old version, so naturally it has
// a slice bounds violation in -x and -s.
/*
	for (GOOS in '' windows) go build whatever
	go build uniq.go
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

import (
	"github.com/as/mute"
)

const (
	Prefix     = "uniq: "
	BufferSize = 1024 * 1024
	Debug      = false // true false
)

type Reader io.ReadCloser

type File struct {
	Reader
	name *string
}

var nmatched int

var args struct {
	h, q bool
	r    bool
	d    bool
	c    bool
	l    bool
	x    string
	y    string
	s    string
}

var f *flag.FlagSet
var selfn Selector

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)

	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.d, "d", false, "")
	f.BoolVar(&args.l, "l", false, "")
	f.BoolVar(&args.c, "c", false, "")
	f.StringVar(&args.x, "x", ".+", "")
	f.StringVar(&args.y, "y", "", "")
	f.StringVar(&args.s, "s", "0:1", "")

	err := mute.Parse(f, os.Args[1:])
	if args.x != "" && args.y != "" {
		usage()
		os.Exit(1)
	}

	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

type Selector func(string, int) []string

type field [2]int

func (r field) Extract(s ...string) []string {
	l := len(s)
	if l == 0 {
		return nil
	}
	n := max(r[0], 0)
	m := min(r[1], l)
	if n == m {
		m = n + 1
	}
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("s %T %v\n", s, s)
			fmt.Printf("l %T %v\n", l, l)
			fmt.Printf("n %T %v\n", n, n)
			fmt.Printf("m %T %v\n", m, m)
			fmt.Printf("r %T %v\n", r, r)
			os.Exit(1)
		}
	}()
	if n > m && n < l {
		return []string{s[n]}
	}
	if n > m {
		return nil
	}
	debugerr(s, "slice", r)
	return s[n:m]
}

func mustAtoi(s string) (i int) {
	i, err := strconv.Atoi(s)
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
	return i
}
func mustSplit(s, sep string, no int) (a []string) {
	a = strings.Split(s, sep)
	if len(a) < no {
		printerr("bad range:", s)
		os.Exit(1)
	}
	return a
}

func mustRange(s string) field {
	if s == ":" {
		return field{0, ^-1}
	}
	n := strings.Index(s, ":")
	switch n {
	case -1:
		return field{mustAtoi(s), 0}
	case 0:
		return field{0, mustAtoi(s)}
	default:
		mn := mustSplit(s, ":", 2)
		return field{mustAtoi(mn[0]), mustAtoi(mn[1])}
	}
}

func max(n, m int) int {
	if n > m {
		return n
	}
	return m
}

func min(n, m int) int {
	return -max(-n, -m)
}

func main() {
	var (
		re    *regexp.Regexp
		selfn Selector
		fs    field
	)
	if args.h || args.q {
		usage()
		os.Exit(1)
	}
	fs = mustRange(args.s)

	if args.y != "" {
		re = regexp.MustCompile(args.y)
		selfn = re.Split
	} else {
		re = regexp.MustCompile(args.x)
		selfn = re.FindAllString
	}
	if args.l {
		args.d = true
	}

	printfn := func(i int, s string) {
		fmt.Println(s)
	}

	in := make(chan File)
	go walker(in, f.Args()...)

	buf := make([]string, 0, 1024*1024)
	full := make(map[string]int)
	sel := make(map[int]string)
	seen := make(map[string]int)
	printed := make(map[string]int)
	i := 0
	if args.c {
		printfn = func(i int, s string) {
			fmt.Print(i, "	")
			fmt.Println(s)
		}
	}
	isdup := func(s string) int {
		match := selfn(s, -1)
		subf := strings.Join(fs.Extract(match...), "")
		sel[i] = subf
		if args.d || seen[subf] == 0 {
			// memorize the complete string represented by the selection
			// -d is lifo
			full[subf] = i
		}
		seen[subf]++
		buf = append(buf, s)
		i++
		return seen[subf]
	}

	for file := range in {
		for sc := bufio.NewScanner(file); sc.Scan(); {
			isdup(sc.Text())
		}
		file.Close()
	}

	if !args.d {
		for i, s := range buf {
			ct := seen[sel[i]]
			if ct == 1 {
				printfn(ct, s)
			} else {
				if printed[sel[i]] == 0 {
					printfn(ct, s)
					printed[sel[i]]++
				}
			}
		}
	} else {
		for i, s := range buf {
			ct := seen[sel[i]]
			if ct == 1 {
				continue
			}
			if printed[sel[i]] < ct-1 && args.l {
				printed[sel[i]]++
			} else {
				printfn(ct, s)
			}
		}
	}

}

func walker(to chan File, args ...string) {
	if len(args) == 0 {
		to <- File{Reader: os.Stdin}
		close(to)
		return
	}

	emitfd := func(n string) {
		fd, err := os.Open(n)
		if err != nil {
			printerr(err)
			fd.Close()
		} else {
			to <- File{name: &n, Reader: fd}
		}
	}

	go func() {
		for _, v := range args {
			if v != "-" {
				emitfd(v)
			} else {
				in := bufio.NewScanner(os.Stdin)
				for in.Scan() {
					emitfd(in.Text())
				}
			}
		}
		close(to)
	}()
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
	uniq - emit or filter unique lines

SYNOPSIS
	uniq [-u -d -c] [file]
	uniq [-x regexp | -y regexp] [-u -d -c] [file]
	uniq [-x regexp | -y regexp] [-s n¹:nⁿ] [-u -d -c] [file]

DESCRIPTION
	Uniq reads lines from file or standard input and prints out
	unique lines.

	The -x and -y flags use regexp to partition each input line
	into substrings.  The -s flag uses an index expression f¹:fⁿ
	to select a range of fields to test for uniqueness.

	The default operates on the entire line, same as:
		uniq -y ''

FLAGS
	-c	Prefix repetition count to each duplicate.
	-d	Print only duplicates.
	-l	Print only the last duplicate.
	-u	Print only lines without duplicates

	-x regexp   Extract substrings matching regexp and 
				limit uniqueness tests to those strings.

	-y regexp   Inverse of -x: excludes matches to regexp.

	-s n:m     Select numbered fields n through m to
			   participate in the substring comparison.

	-s n:      Field n and every field after
	-s  :m     Every field up to m

EXAMPLE
	Let books be a list of book prices and titles:
	   echo 90.00  The Go Programming Language > books
	   echo 80.00  The Elements of Style      >> books
	   echo 85.00  The C Programming Language >> books
	   echo  3.59  Snookie: A Shore Thing     >> books
	   echo  0.19  Hardcore Java              >> books
	   echo  0.03  The C++ Programming Language >> books
	   echo  0.03  The C++ Programming Language >> books
	   echo  0.01  XML: Its Not So bad        >> books

	   uniq books	                # The second C++ line
	   uniq -d books	            # Only the C++ line
	   uniq -x '[0-9]+'             # Lines with unique dollar value.
	   uniq -x '[0-9]+' -s 2 books	# Lines with unique cent value.

	Flag -y computes regexp⁻¹ from regexp and executes -x
	regexp⁻¹.

BUGS
	

`)
}
