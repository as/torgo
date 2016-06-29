// Copyright 2015 "as". All rights reserved. The program and its corresponding
// gotools package is governed by an MIT license.
//
// Dial dials a network endpoint

package main

import (
	"fmt"
	"flag"
	"io"
	"net"
	"os"
	"os/exec"
	"bufio"
)
import (
	"github.com/as/mute"
)

const (
	Prefix     = "dial: "
	Debug      = false
)

var args struct {
	h, q, v bool
	k       bool
	m       bool
	a       int
	n	string
}


func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.k, "k", false, "")
	f.BoolVar(&args.m, "m", false, "")
	f.IntVar(&args.a,  "a", 4096, "")
	f.StringVar(&args.n,  "n", "tcp4", "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

var (
	socket string
	proto  string
	cmd    []string
	done chan error
)


func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.k, "k", false, "")
	f.BoolVar(&args.m, "m", false, "")
	f.IntVar(&args.a,  "a", 4096, "")
	f.StringVar(&args.n,  "n", "tcp4", "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

var f *flag.FlagSet

func main() {
	nargs := len(f.Args())
	if args.h || args.q || nargs == 0 {
		usage()
		os.Exit(1)
	}

	srv := f.Args()[0]
	cmd := f.Args()[1:]
	
	fd, err := net.Dial(args.n, srv)
	sysfatal(err)

	if args.m {
		streammux(fd, cmd...)
	} else {
		stream(fd, cmd...)
	}
}

func stream(fd net.Conn, cmd ...string) {
	var err error
	func(cfd net.Conn) {
		defer cfd.Close()
		if err != nil {
			printerr(err); return
		}
		verb("dial:", cfd.RemoteAddr().String())
		if len(cmd) == 0 {
			err := term3(cfd)
			printerr(err)
		} else {
			err := run3(cfd, cmd[0], cmd[1:]...)
			printerr(err)
		}
	}(fd)
}
func streammux(fd net.Conn, cmd ...string) {
	sem := make(chan bool, args.a)
	pimper := make(chan io.ReadWriter)
	pimpedwr := make([]io.Writer, 0)
	pimpedrr := make([]io.Reader, 0)
	pimped := make(map[io.Reader]int)
	go func() {
		i := 0
		for p := range pimper {
			pimpedwr = append(pimpedwr, p)
			pimpedrr = append(pimpedrr, p)
			mr := io.MultiReader(pimpedrr...)
			mw := io.MultiWriter(pimpedwr...)
			mrw := bufio.NewReadWriter(bufio.NewReader(mr), bufio.NewWriter(mw))
			i = len(pimpedwr) - 1
			pimped[p]=i
			pimper <- mrw
		}
	}()
	for {
		sem <- true 
		go func(cfd net.Conn) {
			var err error
			defer func() { <- sem }()
			defer cfd.Close()
			pimper <- cfd
			lol := <- pimper
			if err != nil {
				printerr(err); return
			}
			verb("accept:", cfd.RemoteAddr().String())
			if len(cmd) == 0 {
			verb("accept:", cfd.RemoteAddr().String())
				err := term3(cfd)
				printerr(err)
			} else {
				err := run3(lol, cmd[0], cmd[1:]...)
				printerr(err)
			}
		}(fd)
	}
}

func sysfatal(err error) {
	if err == nil {
		return
	}
	printerr(err)
	os.Exit(1)
}

func verb(i ...interface{}) {
	if args.v {
		printerr(i...)
	}
}
func term3(rw net.Conn) (err error) {
	fin := make(chan error)
	defer close(fin)
	defer verb("stdin: released")
	go func() {
		verb("open: net|stdout")
		defer verb("close: net|stdout")
		//defer pw.Close()
		_, err := io.Copy(os.Stdout, rw)
		fin <- err
	}()

	func () {
		verb("open: stdin|net")
		defer verb("close: stdin|net")
		if _, err := io.Copy(rw, os.Stdin); err != nil {
			printerr("stdin|net", err)
		}
	}()
	return <- fin
}

func run3(rw io.ReadWriter, cmd string, args ...string) (err error) {
	fin := make(chan error)
	defer close(fin)
	defer verb("cmd: released")

	c := exec.Command(cmd, args...)
	in, err := c.StdinPipe()
	if err != nil {
		return err
	}
	pr, pw := io.Pipe()
	defer pw.Close()
	c.Stdout, c.Stderr = pw, pw

	verb("cmd: born")
	if err = c.Start(); err != nil {
		return err
	}
	
	go func() {
		verb("open: net|cmd")
		defer verb("close: net|cmd")
		if _, err := io.Copy(in, rw); err != nil {
			printerr(err)
		}
		pw.Close()
		fin <- err
	}()

	func () {
		verb("open: cmd|net")
		defer verb("close: cmd|net")
		if _, err := io.Copy(rw, pr); err != nil {
			printerr("cmd|net", err)
		}
	}()

	if err := <- fin; err != nil {
		printerr("net|cmd", err)
	}
	in.Close()
	verb("cmd: moribound")
	return c.Wait()
}

/*
 *	Below is UDP stuff. Experimental and not tested
 */

func newudp(p *pkt) *udp {
	rx := make(chan *pkt)
	return &udp{
		p.src,
		nil,
		p,
		rx,
		nil,
	}
}

func (u udp) Read(b []byte) (n int, err error) {
	printerr("udp.read")
	if u.rx == nil {
		return 0, nil
	}

	b2, ok := <-u.rx
	if !ok {
		return 0, fmt.Errorf("rx closed")
	}
	if int(b2.size) > len(b) {
		err = fmt.Errorf("short read")
	}
	copy(b, b2.data)
	printerr("e.read")
	return len(b), err
}

func (u udp) Write(b []byte) (n int, err error) {
	printerr("udp.write")
	if u.tx == nil {
		return 0, nil
	}
	u.tx <- &pkt{len(b), u.raddr, b}
	printerr("e.write")
	return len(b), nil
}

func (u udp) Close() error {
	printerr("udp.close")
	if u.tx != nil {
		close(u.tx)
	}
	if u.rx != nil {
		close(u.rx)
	}
	printerr("e.close")
	return nil
}

type udp struct {
	raddr, laddr *net.UDPAddr
	first        *pkt
	rx           chan *pkt
	tx           chan *pkt
}

type pkt struct {
	size int
	src  *net.UDPAddr
	data []byte
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func printdebug(v ...interface{}) {
	if Debug {
		printerr(v...)
	}
}

func usage() {
	fmt.Println(`
NAME
	dial - connect to a listener

SYNOPSIS
	dial [-v -k -r rflags] [-p proto] [host]:port [cmd args ...]
	dial example.com:80 

DESCRIPTION
	Dial establishes a connection with the listener on the
	remote host and runs cmd. Cmd's three standard file
	descriptors (stdin, stdout+stderr) are connected to the
	listener via proto (default tcp).

	If cmd is not given, the standard file descriptors are
	instead connected to dial's standard input, output, and
	error.

EXAMPLE
	Speak HTTP
		echo GET / HTTP/1.1 | ./dial example.com:80

	RDP tunnel through port 80
		listen :80 dial 10.2.64.20:3389
		# All calls to localhost:80 now go to 10.2.64.20:3389
		rd localhost:80 

BUGS
	Redundant on Plan 9.

	Stdout and stderr are fused together into a miserable gulash.
`)
}
