// Copyright 2016 "as".

package main

/*
	for (GOOS in '' windows) go build whatever
	go build xo.go
*/

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/as/argfile"
	"github.com/as/mute"
	x2j "github.com/basgys/goxml2json"
)

const (
	Prefix    = "xml2json: "
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

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.verb, "verb", false, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.H, "?", false, "")
	f.BoolVar(&args.q, "q", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		//printerr(err)//todo
		os.Exit(1)
	}
}

func main() {
	a := f.Args()
	for fd := range argfile.Next(a...) {
		xml2json(fd)
		fd.Close()
	}
}

func xml2json(fd io.Reader) {
	d, err := x2j.Convert(fd)
	d2 := new(bytes.Buffer)
	json.Indent(d2, d.Bytes(), " ", "   ")
	if err != nil{
		printerr(err)
		os.Exit(1)
	}
	fmt.Println(string(d2.Bytes()))
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
