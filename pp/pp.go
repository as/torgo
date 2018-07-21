// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
)

const prefix = "pp: "

var (
	h1, h2 = flag.Bool("h", false, "help"), flag.Bool("?", false, "help")
)

func init() {
	log.SetPrefix(prefix)
	flag.Parse()
	if *h1 || *h2 {
		usage()
		os.Exit(0)
	}
}

func main() {
	src, err := ioutil.ReadAll(os.Stdin)
	ck("read", err)

	dst := new(bytes.Buffer)

	if err = json.Indent(dst, src, "", "\t"); err != nil {
		fmt.Print(string(src))
		os.Exit(1)
	}
	_, err = io.Copy(os.Stdout, dst)
	ck("write", err)
}

func ck(where string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", where, err)
	}
}

func usage() {
	fmt.Print(`
NAME
	pp - pretty print

SYNOPSIS
	pp

DESCRIPTION
	Pp is a pretty printer. If it can't make it pretty,
	it outputs the current ugly and returns a non-zero exit code

BUGS
	Only works on JSON for now
`)
}
