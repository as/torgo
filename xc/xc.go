package main

import (
//	"encoding/binary"
//	"encoding/base64"
//	"encoding/base32"
	"encoding/hex"
//	"encoding/pem"
//	"encoding/csv"
	"bytes"
	"unicode"
	"io"
	"bufio"
	"os"
	"fmt"
	"flag"
)

import (
	"github.com/as/mute"
)

const (
	Prefix     = "xc: "
	NBuffer = 1024*1024
)
var f *flag.FlagSet
var args struct {
	h, q  bool
	r     bool
	k string
}
var (
	in io.Reader = os.Stdin
	out io.Writer = os.Stdout
)

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.StringVar(&args.k, "k", "", "")
	f.BoolVar(&args.r, "r", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
	if args.h || args.q {
		usage()
		os.Exit(1)
	}
}

func delete(src []byte) []byte {
	m := func(r rune) rune {
		if unicode.IsSpace(r) {
			return(rune(-1))
		}
		return r
	}
	return bytes.Map(m, src)
}

type Filter func([]byte) []byte

type FilterReader struct {
	r   io.Reader
	fn  Filter
}

func NewFilterReader(r io.Reader, fn Filter) *FilterReader {
	return &FilterReader{r, fn}
}

func (f *FilterReader) Read(p []byte) (n int, err error) {
	n, err = f.r.Read(p)
	if n > 0{
		n = copy(p, f.fn(p[:n]))
	}
	return n, err
}



// Modreader wraps an io.Reader to guarantee aligned reads
// per each read call.
type ModReader struct {
	mod int
	r   io.Reader
	buf *bytes.Buffer
	mux io.Reader
}

// NewModReader wraps an io.Reader, returning a new
// reader that aligns reads to the modulus m. Residues
// are rounded down and remainders are stored in a ring
// buffer.
func NewModReader(r io.Reader, m int) *ModReader {
	b := new(bytes.Buffer)
	mux := io.MultiReader(b, r)
	return &ModReader{
		m,
		r,
		b,
		mux,
	}
}

// Read reads a number of bytes congruent to the
// underlying modulus. An EOF is returned after
// the ring buffer is drained.
//
// Todo: Behavior of what happens to the residue
// after EOF
func (m ModReader) Read(p []byte) (n int, err error) {
	n, err = m.mux.Read(p)
	if err != nil {
		return 
	}
	d := n % m.mod
	if d != 0{
		// Copy the remainder to buf for the next Read
		n -= d
		m.buf.Write(p[n:])
	}
	return
}

func main() {
	if args.r {
		hexdecode()
		os.Exit(0)
	}
	hexencode()
}
var(
	n  int
	err   error
	nospace = bytes.TrimSpace
	B  = make([]byte, 1024*1024)
)


func hexencode() {
	B2 := make([]byte, len(B)*2)
	in := bufio.NewReader(in)
	for err == nil {
		n, err = in.Read(B);
		if  err != nil && n < 1 {
			break
		}
		n = hex.Encode(B2, B[:n])
		n, err = out.Write(B2[:n])
	}
	if err != nil && err != io.EOF {
		printerr(err)
	}
}

func hexdecode() {
	in = NewModReader(NewFilterReader(in, delete), 2)
	for err == nil {
		n, err = in.Read(B)
		if err != nil {break}
		n, err = hex.Decode(B[:n], B[:n])
		if err != nil {break}
		n, err = out.Write(B[:n])
	}
	if err != nil && err != io.EOF {
		printerr(err)
	}
}


func usage() {
	fmt.Println(`
NAME
	xc - Transcode the input stream

SYNOPSIS
	xc [-r] [fmt]

DESCRIPTION
	Xc transcodes a byte stream read from standard
	input to fmt, writing the result to standard
	output. 

	The -r flag reverses the encoding.

OPTIONS
	-r   	Reverse: Decode from ASCII to fmt

FORMATS
	16	Base16		Hexidecimal
	64	Base64		Base 64
	32	Base32		RFC 4648 Standard encoding
	32x Base62 Hex	RFC 4648 Extended Hex Alphabet

EXAMPLE
	echo down over | xc 64 | xc -r 64

BUGS
	

SEE ALSO

`)
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}
