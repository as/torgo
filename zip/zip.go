// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
//
// Walk traverses directory trees, printing visited files

package main

import (
	"bytes"
	"archive/zip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/as/argfile"
	"github.com/as/mute"
)

func init() { log.SetPrefix("zip: ") }

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.c,  "c",  true,  "")
	f.BoolVar(&args.x,  "x",  false, "")
	f.BoolVar(&args.v,  "v",  false, "")
	f.BoolVar(&args.t,  "t",  false, "")
	f.StringVar(&args.f, "f", "",  "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

var args struct {
	h, q       bool
	v, x, c, t bool
	f          string
}

func info(file string) (os.FileInfo, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return fd.Stat()
}

func writezip(w io.Writer) {
	tw := zip.NewWriter(w)
	defer tw.Close()
	for file := range argfile.Next(f.Args()...) {
		// Close this meaningless abstraction and open
		// the real file for reading, also obtain its size
		// for the header.
		err := func(file *argfile.File) error {
			name := file.Name
			defer file.Close()
			fi, err := info(name)
			if err != nil {
				return err
			}
			if args.t {
				fmt.Println(name)
				return nil
			}
			if args.v {
				fmt.Fprintln(os.Stderr, name)
			}
			// Write the header and file into the archive
			hdr := &zip.FileHeader{
				Name: name,
				UncompressedSize64: uint64(fi.Size()),
			}
			out, err := tw.CreateHeader(hdr)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, file.ReadCloser)
			if err != nil {
				return err
			}
			return tw.Flush()
		}(file)
		if err != nil {
			log.Println(err)
			break
		}
	}
}

func readzip(in io.Reader, size int64) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(in)
	err := func() error {
		tr, err  := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		if err != nil{
			return err
		}
		for _, f := range tr.File {
			hdr := f.FileHeader
			if err != nil{
				log.Printf("error:", err)
				continue
			}
			if args.t {
				fmt.Println(hdr.Name)
				continue
			}
			if args.v {
				fmt.Fprintln(os.Stderr, hdr.Name)
			}
			os.MkdirAll(filepath.Dir(hdr.Name), 0700)
	
			fd, err := os.Create(hdr.Name)
			if err != nil {
				// memory leak
				log.Println(err)
				continue
			}
			ifd, err := f.Open()
			if err != nil {
				// memory leak
				log.Println(err)
				continue
			}
			if _, err = io.Copy(fd, ifd); err != nil {
				ifd.Close()
				fd.Close()
				return err
			}
			ifd.Close()
			fd.Close()
		}
		return nil
	}()
	if err != nil{
		log.Printf("error: %s", err)
	}
}

func main() {
	var err error
	infd, outfd := io.Reader(bufio.NewReader(os.Stdin)), os.Stdout
	switch {
	case args.x || args.t:
		var size int64
		if args.f != "" {
			infd, err = os.Open(args.f)
			no(err)
			fi, err := os.Stat(args.f)
			no(err)
			size = fi.Size()
		}
		readzip(infd, size)
	case args.c:
		if args.f != "" {
			outfd, err = os.Create(args.f)
			no(err)
		}
		buf := bufio.NewWriter(outfd)
		writezip(buf)
		err = buf.Flush()
	default:
		log.Fatalln("bad option: need -c or -x")
	}
	if err != nil {
		log.Fatalln(err)
	}
}

func no(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func usage() {
	fmt.Println(`
NAME

SYNOPSIS

DESCRIPTION

EXAMPLES

`)
}
