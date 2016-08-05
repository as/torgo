package main

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func driver(rie []string) (ac string, err error) {
	re, in, ex := rie[0], rie[1], rie[2]
	r, err := sregexp(
		bytes.NewReader([]byte(in)),
		re,
	)
	if err != nil && err != io.EOF {
		return
	}
	buf, _, err := r.Structure()
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("fail: unexpected err: %s", err)
	}
	ac = string(buf)
	if ac != ex {
		return ac, fmt.Errorf("fail: %q != %q\n", ac, ex)
	}
	return ac, nil
}

func multi(rie []string) (ac string, err error) {
	re, in := rie[0], rie[1]
	r, err := sregexp(
		bytes.NewReader([]byte(in)),
		re,
	)
	if err != nil && err != io.EOF {
		return
	}
	for i, ex := range rie[2:] {
		verb.Printf("\tmulti #%d: ex=%q\n", i, ex)
		buf, _, err := r.Structure()
		if buf == nil {
			return ac, fmt.Errorf("buf is nil")
		}
		if err != nil {
			return ac, fmt.Errorf("fail: %q != %q\n", ac, ex)
		}
		ac = string(buf)
		if ac != ex {
			return ac, fmt.Errorf("fail: %q != %q\n", ac, ex)
		}
	}
	return ac, nil
}
var ls = `./go/src/9fans.net/go/.git/config
./go/src/9fans.net/go/.hgignore
./go/src/9fans.net/go/LICENSE
./go/src/9fans.net/go/README
./go/src/9fans.net/go/acme/Dict/Dict.go
./go/src/9fans.net/go/acme/Makefile
./go/src/9fans.net/go/acme/Watch/main.go
./go/src/9fans.net/go/acme/acme.go`

func TestMultiLine(t *testing.T) {
	var err error
	ndb := "sys=xray ip=1.1.1.1\nsys=present ip=1.1.1.2\nsys=web ip=1.1.1.1.3\n"

	table := [][]string{
		{`/\n/;/[^.]+/`, ls, "git/config\n", "hgignore\n"},
		{`/\n/;/(\x2f|\.)/`, ls, "/config\n", ".hgignore\n"},
		{`/\n/`, "this\nis\na\ntest", "\n", "\n"},
		{`,/\n/`, "this\nis\na\ntest", "this\n", "is\n"},
		{`,/\n/`, ndb, "sys=xray ip=1.1.1.1\n", "sys=present ip=1.1.1.2\n"},
		{`,/ /`, ndb, "sys=xray ", "ip=1.1.1.1\nsys=present ", "ip=1.1.1.2\nsys=web "},
		{`,/[\n ]/`, ndb, "sys=xray ", "ip=1.1.1.1\n", "sys=present ", "ip=1.1.1.2\n", "sys=web "},
		{`,/(\n| )/`, ndb, "sys=xray ", "ip=1.1.1.1\n", "sys=present ", "ip=1.1.1.2\n", "sys=web "},
		{`,/(\n| |=)/`, ndb, "sys=", "xray ", "ip=", "1.1.1.1\n", "sys=", "present ", "ip=", "1.1.1.2\n", "sys=", "web "},
		{`/[A-Z]/,/\n[A-Z]/-/./                 `, man, "NAME\n	xo - Search for patterns in arbitrary structures\n\n"},
		{`/[A-Z]+/,/\n\n/`, man,
			"NAME\n	xo - Search for patterns in arbitrary structures\n\n",
			"SYNOPSIS\n\txo [flags] [-x linedef] regexp [file ...]\n\n",
		},
		//{`,/$/-/^/+/./`, "this\nis\na\test", "\n"},
		//{`,/./`, "dot test", "d", "o", "t", " ", "t", "e", "s", "t"},
		// {`/\n/;/(\/|\.)/`, ls, ".git/config", ".hgignore"}, // crash: fix the parser
		// {`,/(\n )/`,  ls, "sys=xray ", "ip=1.1.1.1\n", "sys=present ", "ip=1.1.1.2\n", "sys=web "},  // crash
		//{`/abc/-/../`, "abcdefg", "a"},
	}
	for i, v := range table {
		_, err = multi(v)
		if err != nil {
			t.Logf("test: %d: %s", i, err)
			t.Fail()
			return
		}
		t.Log("test", i, "pass")
	}
}
func TestBasic(t *testing.T) {
	var err error
	table := [][]string{
		{`/abcd/-/../`, "abcdefg", "ab"},
		{`/abc/-/.../`, "abcdefg", ""},
		{`/abc/-/./`, "abcdefg", "ab"},
		{`,/\n/`, "sys=xray ip=1.1.1.1\nsys=present ip=1.1.1.2\nsys=web ip=1.1.1.1.3\n", "sys=xray ip=1.1.1.1\n"},
		{`/aa/-/./`, "aaa", "a"},
		{`/abc/-/./`, "zabcdefg", "ab"},
		{`/abc/-/./`, "abcdefg", "ab"},
		{`/[^.]+/`, "/dir/f0.jpg.bat.png\n", "/dir/f0"},
		{`/\n/-/..../`, "/dir/f1.png.bat.jpg\n", ".jpg"},
		{`/\n/-/[^.]+\./`, "/dir/f2.jpg.bat.png\n", ".png"},
		{`/ /-/..../`, "the quick brown fox", "the"},
		{"/a//b/", "aaabb", "b"},
		{"/gab//we/", "gabenewell", "we"},
		{"/dddddddddddddd/", "dddddddddddddd", "dddddddddddddd"},
		{"/a/", "a", "a"},
		{"/a/", "aa", "a"},
		{"/aa/", "aa", "aa"},
		{"/ab/", "ab", "ab"},
		{"/a./", "ab", "ab"},
		{"/(one|two)/", "zeroonetwo", "one"},
		{"/[abc123]+//./", "abc123z", "z"},
		{"/⁵/", "⁵", "⁵"},
		{"/\x00/", "\x00", "\x00"},
		{"/../", "ab", "ab"},
		{"/./", "a\n", "a"},
		{"/c//c/", "cc", "c"},
		{"/zx/", "zzzzzzzzzzzzzzzzzzzzzzzzx", "zx"},
		{"/zx/", "xzzzzzzzzzzzzzzzzzzzzzzzx", "zx"},
		{"/(xz)+../", "xzxzzzzzzxzzzzzx", "xzxzzz"},
		{"/./", "xzxzzzzzzxzzzzzx", "x"},
		{"/z/,/./", "idontzzebraz", "zz"},
		{`,/ /`, "the quick brown fox", "the "}, // crash
		{`/[abc]+/`, "abc", "abc"},
		{"/[a-z]+/,/sucks/", "xml sucks", "xml sucks"},
		{"/\x00\x00/,/\x01/", "\x00\x00\x11\x01", "\x00\x00\x11\x01"},
		{"/a/,/b/", "ab", "ab"},
		{"/a/,/a/", "aa", "aa"},
		{"/d/,/d/", "dd", "dd"},
		{"/a//b/", "aaab", "b"},
		{"/a/,/b/", "abcd", "ab"},
		{"/a/,/b/", "abab", "ab"},
		{"/a/,/b/", "aaabaab", "aaab"},
		{"/a/,/b/", "aabaab", "aab"},
		{"/aa/,/b/", "aaabaab", "aaab"},
		{`/ab/,/./`, "abcdefg", "abc"},
		{`/abc/,/./`, "abcdefg", "abcd"},
		{"/a/,/b/", "abc", "ab"},
		{"/a/,/c/", "abc", "abc"},
		{"/aaa+/", "aaaaaaaaaaaaah", "aaaaaaaaaaaaa"},
		{"/a/,/b/", "abab", "ab"},
		{"/a/,/b/", "aaabaab", "aaab"},
		{"/a/,/b/", "aabaab", "aab"},
		{"/gab/+/we/", "gabenewell", "we"},
		{`/abc/-/./`, "abcdefg", "ab"},
		{"/aa/-/a/", "aaaa", "a"},
		{"/aaaa/-/a/", "aaaaa", "aaa"},
		{"/@//./,/[a-z]+./", "@bullshit@", "bullshit@"},
		{"/aaa/-/a/", "aaaa", "aa"},
		{"/baaaa/-/aa/", "baaaaaa", "baa"},
		{"/z/,/./", "idontzzebraz", "zz"},
		//{"/..../,/z+/,/b/", "idontzebra", "idontzeb"},
		// {`/abc/-/.../`, "abcdefg", ""}, // drained
	}
	for i, v := range table {
		_, err = driver(v)
		if err != nil && err != io.EOF {
			t.Logf("test: %d: %s", i, err)
			t.Fail()
			return
		}
		t.Log("test", i, "pass")
	}
}

var man = `
NAME
	xo - Search for patterns in arbitrary structures

SYNOPSIS
	xo [flags] [-x linedef] regexp [file ...]

DESCRIPTION
	Xo scans files for pattern using regexp. By default xo
	applies regexp to each line and prints matching lines found.
	This default behavior is similar to Plan 9 grep.

	However, the concept of a line is altered using -x by setting
	linedef to a structural regular expression set in the form:

	   -x /start/
	   -x /start/,
	   -x ,/stop/
	   -x /start/,/stop/

	Start, stop, and all the data between these two regular
	expressions, forms linedef, the operational definition of a line.

	The default linedef is simply: /\n/

	Xo reads lines from stdin unless a file list is given. If '-' is 
	present in the file list, xo reads a list of files from
	stdin instead of treating stdin as a file.

FLAGS
	Linedef:

	-x linedef	Redefine a line based on linedef
	-y linedef	The negation of linedef becomes linedef

	Regexp:

	-v regexp	Reverse. Print the lines not matching regexp
	-f file     File contains a list of regexps, one per line
				the newline is treated as an OR

	Tagging:

	-o  Preprend file:rune,rune offsets
	-l	Preprend file:line,line offsets
	-L  Print file names containing no matches
	-p  Print new line after every match

EXAMPLE
	# Examples operate on this help page, so
	xo -h > help.txt

	# Print the DESCRIPTION section from this help
	xo -p -o -x '/^[A-Z]/,/./' . help.txt

	# Print the Tagging sub-section
	xo -h | xo -x '/[A-Z][a-z]+:/,/\n\n/' Tagging

BUGS
	On a multi-line match, xo -l prints the offset
	of the final line in that match.
	
`
