// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
//

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"
)

const Prefix = "crawl: "

var absurl = regexp.MustCompile(`http[s]://[^"]+`)

type ec chan error

var (
	get0, get1, get2 = make(chan string), make(chan []byte), make(ec)
	link0, link1     = make(chan []byte), make(chan [][]byte)
)

func (e ec) ck(ctx string, err error) (ok bool) {
	if err == nil {
		return true
	}
	select {
	case e <- err:
	default:
		println("error handler overloaded", err)
	}
	return false
}

func (e ec) readclose(dst *[]byte, rc io.ReadCloser) (ok bool) {
	defer rc.Close()
	b, err := ioutil.ReadAll(rc)
	*dst = b
	return get2.ck("readclose", err)
}
func link(done chan bool) {
	for {
		select {
		case <-done:
			return
		case body := <-link0:
			link1 <- absurl.FindAll([]byte(body), -1)
		}
	}
}
func get(done chan bool) {
	c := http.Client{
		Timeout: time.Second * 5,
	}
	for {
		select {
		case <-done:
			return
		case url := <-get0:
			b := []byte{}
			resp, err := c.Get(url)
			if !get2.ck("get", err) || !get2.readclose(&b, resp.Body) {
				continue
			}
			get1 <- b
		}
	}
}

var n = 1

func main() {
	done := make(chan bool)
	go get(done)
	go link(done)
	go func() {
		for body := range get1 {
			link0 <- body
		}
	}()
	link0 <- []byte(os.Args[1])

Loop:
	for {
		select {
		case url, more := <-link1:
			if !more {
				break Loop
			}
			n = len(url) - 1
			if n == 0 {
				println("done?")
				close(done)

			}
			for _, url := range url {
				fmt.Println(url)
				get0 <- string(url)
			}
		}
	}
}
