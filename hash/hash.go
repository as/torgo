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
	"runtime"

	"golang.org/x/crypto/md4"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

var (
	help  = flag.Bool("h", false, "")
	quest = flag.Bool("?", false, "")
	quiet = flag.Bool("q", false, "")
	bs    = flag.Bool("b", false, "")
	csp   = flag.Int("csp", runtime.NumCPU(), "")
)

func init() {
	flag.Usage = usage
	flag.Parse()
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

func main() {
	a := flag.Args()
	if *help || *quest {
		usage()
		os.Exit(0)
	}
	if len(a) == 0 {
		usage()
		os.Exit(1)
	}
	initfn, ok := std[a[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "no hash alg: %q", a[0])
		os.Exit(1)
	}
	if *bs {
		h := initfn()
		fmt.Println(h.BlockSize())
		os.Exit(0)
	}
	file := a[1:]

	n := *csp
	in := make(chan work, n)
	out := make(chan work, n)
	done := make(chan int)

	for i := 0; i < n; i++ {
		go func() {
			ha{Hash: initfn(), in: in, out: out, done: done}.run()
		}()
	}
	go func() {
		defer close(in)
		if len(file) == 0 {
			in <- work{file: ""}
			return
		}
		for _, f := range file {
			if f != "-" {
				in <- work{file: f}
			} else {
				list := bufio.NewScanner(os.Stdin)
				for list.Scan() {
					in <- work{file: list.Text()}
				}
			}
		}
	}()
	for {
		select {
		case <-done:
			n--
		case w := <-out:
			if *quiet || len(file) == 0 {
				fmt.Printf("%x\n", w.hash)
			} else {
				fmt.Printf("%x	%s\n", w.hash, w.file)
			}
		}
		if n == 0 && len(out) == 0 {
			break
		}
	}
}

type work struct {
	file string
	hash string
}
type ha struct {
	hash.Hash
	in   <-chan work
	out  chan<- work
	done chan<- int
}

func (h ha) run() {
	defer func() {
		h.done <- 1
	}()
	for w := range h.in {
		h.Reset()
		fd := os.Stdin
		var err error
		if w.file != "" {
			if fd, err = os.Open(w.file); err != nil {
				continue
			}
		}
		io.Copy(h, fd)
		fd.Close()
		w.hash = string(h.Sum(nil))
		h.out <- w
	}
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
