package main

import (
	"bufio"
	"os"
)

var (
	no  = []byte("no")
	err error
)

func main() {
	if len(os.Args) > 1 {
		no = []byte(os.Args[1])
	}
	for br := bufio.NewWriter(os.Stdout); err == nil; _, err = br.Write(no) {
	}
}
