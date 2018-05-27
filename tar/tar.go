// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the Go programming language.

package main

import (
	//"bufio"
	"archive/tar"
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

func init() {
	log.SetPrefix("tar: ")
	log.SetFlags(0)
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.c, "c", true, "")
	f.BoolVar(&args.x, "x", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.t, "t", false, "")
	f.StringVar(&args.f, "f", "", "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if args.h || args.q {
		usage()
		os.Exit(0)
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

func writetar(w io.Writer) {
	tw := tar.NewWriter(w)
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
			hdr := &tar.Header{
				Name: filepath.ToSlash(name),
				Mode: int64(fi.Mode()),
				Size: int64(fi.Size()),
			}
			if err = tw.WriteHeader(hdr); err != nil {
				return err
			}
			_, err = io.Copy(tw, file.ReadCloser)
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
func readtar(in io.Reader) {
	err := func() error {
		tr := tar.NewReader(in)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("next:", err)
				continue
			}
			if args.t {
				fmt.Println(filepath.FromSlash(hdr.Name))
				continue
			}
			if args.v {
				fmt.Fprintln(os.Stderr, hdr.Name)
			}
			if fi := hdr.FileInfo(); fi.IsDir() {
				// Does this ever get triggered?
				os.MkdirAll(hdr.Name, fi.Mode())
			} else {
				if dir := filepath.Dir(hdr.Name); !exists(dir) {
					if dir == hdr.Name {
						log.Printf("bug: mkdir creating file in tar as the directory instead")
					}
					os.MkdirAll(dir, fi.Mode())
				}
				fd, err := os.Create(hdr.Name)
				if err != nil {
					return err
				}
				if _, err = io.Copy(fd, tr); err != nil {
					fd.Close()
					return err
				}
				fd.Close()
			}
		}
		return nil
	}()
	if err != nil {
		log.Printf("error: %s", err)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func main() {
	var err error
	infd, outfd := io.Reader(bufio.NewReader(os.Stdin)), os.Stdout
	switch {
	case args.x || args.t:
		if args.f != "" {
			infd, err = os.Open(args.f)
			no(err)
		}
		readtar(infd)
	case args.c:
		if args.f != "" {
			outfd, err = os.Create(args.f)
			no(err)
		}
		buf := bufio.NewWriter(outfd)
		writetar(buf)
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
