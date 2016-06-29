// Copyright 2015 "as". All rights reserved. Same license as Go.
//
// comm reads in two sorted files and outputs three columns: lines
// exclusively in file1, lines exclusively in file2, and lines present in
// both files. The columns are tab-seperated. A column can be hidden (not
// printed) by running comm with the flag [-123]. For example, to print
// lines exclusively present in both files:
//
// comm -12 file1 file2

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	a := os.Args
	if len(a) < 3 {
		fmt.Println("comm: missing args")
		usage()
		os.Exit(1)
	}

	// true is suppressed, false is not
	hidden := make([]bool, 4)
	maxtabs := 2

	if len(a[1]) > 1 && a[1][0] == '-' {
		hide := a[1][1:]
		a = a[1:]

		for _, v := range hide {
			col, err := strconv.Atoi(string(v))
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if col < 1 || col > 3 {
				fmt.Println("comm: bad column", col)
				usage()
				os.Exit(1)
			}

			hidden[col] = true
			maxtabs--
		}
	}

	filenm := a[1:3]
	file, err := mkScanners(filenm)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	// print fn prints 's'  w/ right no. tabs
	//  if the column 'n' isn't hidden
	print := func(s string, n int) {
		if hidden[n] {
			return
		}
		ntabs := n - (3 - maxtabs)
		fmt.Printf("%s%s\n", tabs(ntabs), s)
	}

	// flush fn prints the longer file with the
	// right no. tabs after the shorter file is eof
	flush := func(b *bufio.Scanner, n int) {
		for {
			if ok := file[n-1].Scan(); !ok {
				return
			}
			fmt.Printf("%s%s", tabs(maxtabs-n), b.Text())
		}
	}

	ln1, ok1 := nextln(file[0])
	ln2, ok2 := nextln(file[1])
	for {
		switch {
		case !ok1:
			// file 1 eof
			print(ln2, 2)
			flush(file[1], 1)
			os.Exit(0)
		case !ok2:
			// file 2 eof
			print(ln1, 1)
			flush(file[0], 2)
			os.Exit(0)
		case ln1 == ln2:
			// in both files
			print(ln1, 3)
			ln1, ok1 = nextln(file[0])
			ln2, ok2 = nextln(file[1])
		case ln1 < ln2:
			// in file one only
			print(ln1, 1)
			ln1, ok1 = nextln(file[0])
		default:
			// in file two only
			print(ln2, 2)
			ln2, ok2 = nextln(file[1])
		}
	}
}

func mkScanners(fname []string) ([]*bufio.Scanner, error) {
	file := make([]*bufio.Scanner, 2)

	for i, v := range fname {
		if v == "-" {
			file[i] = bufio.NewScanner(os.Stdin)
		} else {
			fp, err := os.Open(v)
			if err != nil {
				return nil, err

			}
			file[i] = bufio.NewScanner(fp)
		}
	}

	return file, nil
}

func nextln(b *bufio.Scanner) (string, bool) {
	ok := b.Scan()
	if !ok {
		return "", false
	}
	return b.Text(), true
}

func tabs(n int) string {
	if n < 0 {
		return ""
	}
	return strings.Repeat("\t", n)
}

func usage() {
	fmt.Println(`
NAME
	comm - compare two sorted files

SYNOPSIS
	comm [-123] file1 file2

DESCRIPTION
	Comm reads two sorted files and outputs three columns: lines
	exclusively in file1, lines exclusively in file2, and lines present in both files. 
	
	The columns are tab-seperated. A column can be suppressed by
	running comm with the flag [-123]. For example, to print lines
	only present in both files:

	# Suppress column 1 and 2, printing only column 3
	comm -12 file1 file2

WARNINGS
	Files must be in sorted order
`)
}
