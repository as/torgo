// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

const prefix = "uq: "

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

	dst, err := strconv.Unquote(string(src))
	if err != nil {
		dst, err = strconv.Unquote(`"` + string(src) + `"`)
	}
	if err != nil {
		fmt.Print(string(src))
		os.Exit(1)
	} else {
		fmt.Print(string(dst))
	}
}

func ck(where string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", where, err)
	}
}

func usage() {
	fmt.Print(`
NAME
	uq - unquote input

SYNOPSIS
	uq < quoted.txt

DESCRIPTION
	Uq reads from standard input and unquotes it using the Go rules
	for quotation. Uq is non-destructive. If the operation fails, uq prints
	the original input and exits with a non-zero status.
`)
}
