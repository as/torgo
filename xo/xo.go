// Copyright 2015 "as". 
// This is the 'new' version of xo, so naturally it doesn't
// even work and has a lot of new bugs.

/*
	for (GOOS in '' windows) go build whatever
	go build xo.go
*/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"regexp"
	"strings"
	"io/ioutil"
	"bytes"
)

import (
	"github.com/as/mute"
	"github.com/as/argfile"
)

const (
	Prefix     = "xo: "
	MaxBuffer  = 1024*1024*512
	Debug      = true               // true false
)

var args struct {
	h, q bool
	r    bool
	o	 bool
	l	 bool
	p	 bool
	v    bool
	f    string
	x    string
	y	 string
	s    string
}

var count struct {
	match, unmatch int64
}
var StartAddr = Addr{}

var f *flag.FlagSet

type Pos [2]int64

type Addr struct {
	Pos
	value interface{}
}

type Reader struct {
	list      []*regexp.Regexp
	match     [][]int
	in         io.Reader
	lines      *bufio.Reader
	sp, ep     Pos
	block      []byte
	buf *bytes.Buffer
	tee *bufio.Reader
	src *bufio.Reader
}
func (a Addr) Begin() int64 {
	return a.Pos[0]
}

func (a Addr) End() int64 {
	return int64(a.Pos[0] + a.Pos[1])
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h,   "h", false,  "")
	f.BoolVar(&args.q,   "?", false,  "")
	f.StringVar(&args.f, "f", "",     "")
	f.BoolVar(&args.l,   "l", false,  "")
	f.BoolVar(&args.o,   "o", false,  "")
	f.BoolVar(&args.p,   "p", false,  "")
	f.StringVar(&args.x, "x", `/\n/`, "")
	f.StringVar(&args.y, "y", "",     "")
	f.BoolVar(&args.r,   "r", false,  "")
	f.BoolVar(&args.v,   "v", false,  "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

func moribound(r *bufio.Reader) (err error) {
	if _, _, err = r.ReadRune(); err != nil {
		return err
	}
	return r.UnreadRune()
}

func (r Reader) Match() [][]int {
	return r.match
}
func NewReader(in io.Reader, c ...cmd) *Reader {
	r := &Reader{
		in:    bufio.NewReader(in),
		buf:   new(bytes.Buffer),
	}
	for _, v := range c {
		r.list = append(r.list, regexp.MustCompile("(?ms)" + v.re))
	}
	r.src = bufio.NewReader(io.TeeReader(bufio.NewReader(r.in), r.buf))
	return r
}

func (r *Reader) Structure() (out []byte, n int, err error) {
	defer func(){printerr("OUT:",out)}()
	var pos []int
	pos = make([]int, 2)
	for i, cmd := range state.prog {
		printerr("ROUND",i,"LB",r.buf.Len())
		pos = r.list[i].FindReaderIndex(r.src)
		printerr("pos",pos)
		if pos == nil {
			return nil, 0, fmt.Errorf("Com no match")
		}
		switch Tok(cmd.op) {
		case Add:
		printerr("Addout",out)
			out = append(out, r.buf.Next((pos[1]+pos[0]))...)
		printerr("Addoutappend",out)
		printerr("Addoutpos",out)
		case Sem:
		case Com:
		printerr("Comout",out)
			need := pos[1]
			out = append(out,r.buf.Next(need)...)
		printerr("Comoutappend",out)
		case Sub:
		case End:
			printerr("End: ")
		}
		printerr("whats left:", string(r.buf.Bytes()))
		r.src = bufio.NewReader(io.MultiReader(bytes.NewBuffer(r.buf.Bytes()),r.src ))
	}
	return out, n, moribound(r.src)
}

func (r Reader) Begin() Pos {
	return r.sp
}

func (r Reader) End() Pos {
	return r.ep
}

func (r Reader) Dot() int64 {
	// After begin, before end
	return r.sp[0]
}

func ipos(i []int) Pos {
	return Pos{int64(i[0]), int64(i[1])}
}

/*
(?P<s>re)(m:re)(?P<e>re)
(start)(?)(end)
(start)
*/
// sregexp returns a structural regular
// expression reader compiled for the
// structure string s. S must be in the
// form of AddrN constants below. The
// syntax is similar to the Sam and Acme
// text editors for Plan 9
//
type Tok string
const (
	Beg Tok = "^"
	End     = ""
	Sla     = "/"
	Com     = ","
	Sem     = ";"
	Add     = "+"
	Sub     =  "-"
)

func parseexp(s string) (ex string, e int, err error) {
	defer func() { debugerr("findexp returns:", ex, err)}()
	var b int
	if len(s) < 2 {
		return "", -1, fmt.Errorf("findexp: short string")
	}
	if b = strings.Index(s, "/"); b < 0 {
		return "", -1, fmt.Errorf("findexp: not found")
	}
	if e = strings.Index(s[1:], "/"); e < 0 {
		return "", -1, fmt.Errorf("findexp: unterminated")
	}
	e++
	return s[b+1:e], e, err
}
func findop(s string) Tok {
	i := strings.IndexAny(s, ";,+-")
	if i < 0 {
		return End
	}
	return Tok(s[:1])
}

func hasslash(s string) bool {
	i := strings.IndexAny(s, "/")
	return i >= 0
}
func inslash(s string) (b bool) {
	defer println("inslash", b)
	if inop(s) {
		return false
	}
	return hasslash(s)
}
func hasop(s string) bool {
	o := findop(s)
	return o == Add || o == Sub || o == Sem || o == Com
}
func inop(s string) bool {
	i := strings.IndexAny(s, "/,+;-")
	return i >= 0 && s[i] != '/'
}
func ineof(s string) bool {
	return s == ""
}

func check(s string, t Tok) bool {
	if len(s) == 0 {
		return t == End
	}
	if len(t) == 0 {
		return len(s) == 0
	}
	return Tok(s[:len(t)]) == t
}

func (c cmd) String() string {
	return fmt.Sprintf("cmd: re: [%s] op [%s]\n", c.re, c.op)
}
type cmd struct{
	re string
	op Tok
}

var state struct {
	lastop    string
	lastcmd   cmd
	prog      []cmd
}

func nuke() {
	state.lastop = ""
	state.lastcmd = cmd{}
	state.prog = nil
}


func begin(s string) {
	func() { debugerr("begin",s)}()
	defer func() { debugerr("begin returns")}()
	state.lastcmd.op = "+"
	parseop(s)
	return 
}
func parseslash(s string) {
	func() { debugerr("SLASH",s)}()
	defer func() { debugerr("		SLASH")}()
	switch{
	case ineof(s):    printerr("eof"); return
	case inop(s):
		printerr("lex: insert $")
		parseop("/$/" + s); return
	case !inslash(s): printerr("parse error"); return
	}
	re, e, err := parseexp(s)
	if err != nil {
		printerr("slash parse error"); return
	}
	state.lastcmd.re = re
	appendcmd()
	parseop(s[e+1:])
}
func parseop(s string) {
	func()         { debugerr("PARSE",s)    }()
	defer func()   { debugerr("		PARSE") }()
	switch{
	case ineof(s):   printerr("eof"); return
	case inslash(s): parseop(string(state.lastcmd.op) + s); return
	case !inop(s):   printerr("parseop: not in op", s); return
	}
	op := findop(s)
	if op == End {
		printerr("eof2"); return
	}
	state.lastcmd.op = op
	parseslash(s[1:])
}
func appendcmd() {
	state.prog = append(state.prog, state.lastcmd)
	debugerr("append call slash", state.prog)
}

func reg(s string) {
	defer func() { debugerr("reg: pre",state.prog)}()
	      func() { debugerr("reg: post",state.prog)}()
	state.lastcmd.re = s
}

func parseaddr(s string) (c []cmd, err error) {
	begin(s)
	return state.prog, err
}

func sregexp(in io.Reader, s string) (*Reader, error) {
	cmd, err := parseaddr(s)
	if err != nil {
		return nil, err
	}
	return NewReader(in, cmd...), err
}

func main() {
	var (
		a = f.Args() // Remaining non-flag args
		re string	 // Regexp to match against
	)
	switch nargs := len(a); {
	case args.h || args.q:
		usage(); os.Exit(0)
	case args.f != "":
		data, err := ioutil.ReadFile(args.f)
		if err != nil {
			printerr(err); os.Exit(1)
		}
		re = string(data)
	case nargs < 1:
		usage(); os.Exit(1)
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
	bin := regexp.MustCompile(re)
	for fd := range argfile.Next(a...) {
		xo(bin, fd)
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

func xo(re *regexp.Regexp, in *argfile.File) {
	var (
		err   error
		n     int
		start int64
		r     *Reader
		buf   []byte
	)
	defer in.Close()
	//
	// What is a line? sregexp will tell us.
	linedef := args.x
	if args.y != "" {
		linedef = args.y
	}
	r, err = sregexp(in, linedef)
	addrs := make([]*Addr, 1, 8192)
	addrs[0] = &StartAddr
	printerr("for")
	for	err == nil {
		printerr("for: xo")
		buf, n, err = r.Structure()
		if buf != nil || err != nil {
			printerr("structure: buf drained /",err)
			break
		}
		a := &Addr{
			Pos{start, int64(n)},
			nil,
		}
		addrs = append(addrs, a)
		fmt.Print(string(buf))
		switch matched := re.Match(buf); {
		case !args.v && !matched:
			continue // bad: no match
		case args.v  &&  matched:
			continue // bad: unwanted match
		default:
			count.match++
		}
		
		if args.l {
			el := 0
			sl := 0
			if el == sl {
				fmt.Printf("%s:%d:	", in.Name, el)
			} else {
				fmt.Printf("%s:%d,%d:	", in.Name, sl, el)
			}
		}
		if args.o {
			fmt.Printf("%s:#%d,#%d:	", in.Name, a.Begin(), a.End())
		}
		fmt.Print(string(buf))
		if args.p {
			fmt.Println()
		}
	}
}

const(
	Before = 0x000001
	Dot    = 0x000010
	After  = 0x000100
	Hole   = 0x001000
	Dot1   = 0x010000
	Over   = 0x100000
)

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
	This default behavior is similar to Plan 9 grep.

	However, the concept of a line is altered using -x by setting
	linedef to a structural regular expression set in the form:

	   -x /start/
	   -x /start/,
	   -x ,/stop/
	   -x /start/,/stop/

	Start, stop, and all the data between these two regular
	expressions, forms linedef, the operational definition of a line.

	The default linedef is simply: /\n/

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

EXAMPLE
	# Examples operate on this help page, so
	xo -h > help.txt

	# Print the DESCRIPTION section from this help
	xo -p -o -x '/^[A-Z]/,/./' . help.txt

	# Print the Tagging sub-section
	xo -h | xo -x '/[A-Z][a-z]+:/,/\n\n/' Tagging

BUGS
	On a multi-line match, xo -l prints the offset
	of the final line in that match.
	
`)
}
