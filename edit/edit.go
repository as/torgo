package main

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/as/edit"
	"github.com/as/text"
)

func main() {
	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	cmd := edit.MustCompile(strings.Join(os.Args[1:], " "))
	data, err := ioutil.ReadAll(in)
	if err != nil {
		log.Fatalf("edit: %s", err)
	}

	buf, _ := text.Open(text.BufferFrom(data))
	if err = cmd.Run(buf); err != nil {
		log.Fatalln("edit: %s", err)
	}

	io.Copy(out, bytes.NewReader(buf.Bytes()))
}
