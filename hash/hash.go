package main

import (
	"bufio"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"flag"
	"fmt"
	"hash"
	"io"
	"os"
	"sync"
)

import (
	"github.com/as/mute"
)

import (
	"golang.org/x/crypto/md4"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

const (
	Prefix     = "hash: "
	BufferSize = 2e16
	Unset      = "‡"
)

var args struct {
	h, q bool
	b    bool
	stfu bool
	csp  int
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.b, "b", false, "")
	f.IntVar(&args.csp, "csp", 0, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

var std = map[string]func() hash.Hash{
	"md4":       md4.New,
	"md5":       md5.New,
	"sha1":      sha1.New,
	"sha256":    sha256.New,
	"sha512":    sha512.New,
	"ripemd160": ripemd160.New,
	"sha3":      sha3.New224,
	"sha3/224":  sha3.New224,
	"sha3/256":  sha3.New256,
	"sha3/384":  sha3.New384,
	"sha3/512":  sha3.New512,
}

type Reader io.ReadCloser
type File struct {
	Reader
	name *string
	hash []byte
}

func main() {
	a := f.Args()
	if len(a) == 0 {
		usage()
		os.Exit(1)
	}
	initfn, ok := std[a[0]]
	if !ok {
		printerr("no hash alg:", a[0])
		os.Exit(1)
	}
	if args.b {
		h := initfn()
		fmt.Println(h.BlockSize())
		os.Exit(0)
	}
	files := a[1:]
	printfn := print2
	if len(files) == 0 {
		printfn = print1
	}

	in := make(chan File, args.csp)
	out := make(chan File, args.csp)
	go walker(in, files...)
	go hasher(initfn, in, out)
	printer(printfn, out)
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

func hasher(init func() hash.Hash, in, out chan File) {
	var wg sync.WaitGroup
	for f := range in {
		wg.Add(1)
		go func(f File) {
			h := init()
			io.Copy(h, f)
			f.Close()
			f.hash = h.Sum(nil)
			out <- f
			wg.Done()
		}(f)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
}

func printer(fn func(f File), ch <-chan File) {
	for h := range ch {
		fn(h)
	}
}

func print1(f File) {
	fmt.Printf("%x\n", f.hash)
}

func print2(f File) {
	fmt.Printf("%x	%s\n", f.hash, *f.name)
}

// Mux combines a slice of channels into one channel
func mux(c ...chan File) chan File {
	var wg sync.WaitGroup
	out := make(chan File, len(c))
	output := func(c <-chan File) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(c))
	for _, v := range c {
		go output(v)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func usage() {
	fmt.Println(`
NAME
	hash - Compute crytographic hashes

SYNOPSIS
	hash alg [file ...]

DESCRIPTION
	Hash uses algorithm alg to compute hashes of each file named
	in the file list. An empty list names stdin as a file.  A '-'
	in the initial file list names stdin as a line-seperated list of 
	named files.

	Each computed hash is printed to stdout. If the file list is not
	empty, the named file is printed alongside each digest.

ALGORITHMS
	sha1
	sha256
	sha512
	sha3       sha3 is sha3/224
	sha3/224
	sha3/256
	sha3/384
	sha3/512
	md4
	md5
	ripemd160

EXAMPLE
	Compute a sha1 of similarly-named image files.
		hash sha1 pic.jpg pic.jpeg

	Compute a sha3 of every file walk visits
		walk /lib/ndb | hash sha3 -
		walk C:\Windows\System32 | hash sha3 -

	Compute one md5 for the entire stdin
		cat dvd.iso | hash md5

SEE ALSO
	Enc
	Gen
	Hmac
`)
}
