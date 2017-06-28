package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	in := os.Stdin
	out := os.Stdout
	var p [65536]byte
	w := &Win{R: nil, Q0: 0, Q1: 0}
	cmd := Cmdparse(strings.Join(os.Args[1:], " "))
	for {
		n, err := in.Read(p[:])
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			return
		}
		w.R = p[:n]
		cmd.fn(w)
		io.Copy(out, bytes.NewReader(w.Bytes()))
	}
}
