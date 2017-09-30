// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.

package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/as/mute"
)

const Prefix = "gzip: "
var (
	args struct {
		h, q                   bool
		d, l, v, t, fast, best bool
	}
	f *flag.FlagSet
)
func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.d, "d", false, "")
	f.BoolVar(&args.l, "l", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.t, "t", false, "")
	f.BoolVar(&args.fast, "fast", false, "")
	f.BoolVar(&args.best, "best", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
}

type Request struct {
	fd      *os.File
	mode    int
	dst     string
	replyto chan error
}

var workin = make(chan *Request)

func worker() {
	for req := range workin {
		func(req *Request) {
			ok := func(err error) bool {
				if err == nil {
					return true
				}
				req.replyto <- err
				close(req.replyto)
				return false
			}

			if args.d {
				zr, err := gzip.NewReader(req.fd)
				if !ok(err) {
					return
				}
				defer zr.Close()

				ofd, err := os.Create(req.dst)
				if !ok(err) {
					return
				}

				_, err = io.Copy(ofd, req.fd)
				if !ok(err) {
					return
				}
			}
		}(req)
	}
}

func toobig(file string) bool{ return false}
func clash(file string)bool{ return false}

func decompress(file string)  {
	fd, err := os.Open(file)
	ck(err)
	defer fd.Close()

	if toobig(file) {
		log.Printf("decompress: too big: %q\n", file)
		return
	}
	if clash(file) {
		log.Printf("decompress: name conflict: %q\n", file)
		return
			}
	errc := make(chan error)
	workin <- &Request{fd: fd, replyto: errc}
	err = <-errc
	if err != nil{
		log.Printf("decompress: %s\n", err)
	}
}

func compress(file string){
}

func main() {
	go worker()
	if len(f.Args()) == 0{
		if args.d{
			zr, err := gzip.NewReader(os.Stdin)
			ck(err)
			defer zr.Close()
			io.Copy(os.Stdout, zr)
		} else {
			zw:= gzip.NewWriter(os.Stdout)
			defer zw.Close()
			io.Copy(zw, os.Stdin)
		}
		os.Exit(0)
	}
	
	log.Fatalln("unfinished program: only supports stdin | stdout right now")

	alg := compress
	if args.d {
		alg = decompress
	}
	
	var wg sync.WaitGroup
	paths := f.Args()
	for _, v := range paths {
		wg.Add(1)
		go func(v string) {
			if v != "-" {
				decompress(v)
			} else {
				in := bufio.NewScanner(os.Stdin)
				for in.Scan() {
					alg(in.Text())
				}
			}
			wg.Done()
		}(v)
	}
	wg.Wait()
}

func ck(err error){
	if err != nil{
		log.Fatalln(err)
	}
}

func usage() {
	fmt.Println(`
NAME
	gzip - compress or decompress stdin

SYNOPSIS
	gzip [-d | -l | -t] [-n | -fast | -best] [file ...]

DESCRIPTION
	gzip compresses or decompresses a file read from standard input
	using LZ77.

	There are a number of options:

    -d     Decompress.

	None of the options below work yet, with the exception of compression
	which is used in the abscense of -d.
	
    -l     List compression ratio
    -v     verbose
    -t     check compression integrity
    -n     n represents compression level with values 1-9 inclusive
    -fast  set n to 1
    -best  set n to 9

EXAMPLES
    cat some.tar.gz | gzip -d | tar -x 
`)
}
