// Copyright 2015 "as". All rights reserved. Same license as Go.
//
// Listens announces on a network port

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
)
import (
	"github.com/as/mute"
)

const (
	Prefix = "listen: "
	Debug  = false
)

var args struct {
	h, q, v bool
	k       bool
	m       bool
	a       int
	n       string
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.k, "k", false, "")
	f.BoolVar(&args.m, "m", false, "")
	f.IntVar(&args.a, "a", 4096, "")
	f.StringVar(&args.n, "n", "tcp4", "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

func main() {
	nargs := len(f.Args())
	if args.h || args.q || nargs == 0 {
		usage()
		os.Exit(1)
	}

	srv := f.Args()[0]
	cmd := f.Args()[1:]

	ln, err := listener(args.n, srv)
	sysfatal(err)

	verb("announce:", ln.Addr())
	if args.m {
		streammux(ln, cmd...)
	} else {
		stream(ln, cmd...)
	}
}

func stream(ln net.Listener, cmd ...string) {
	sem := make(chan bool, args.a)
	for {
		sem <- true
		go func(cfd net.Conn, err error) {
			defer func() { <-sem }()
			defer cfd.Close()
			if err != nil {
				printerr(err)
				return
			}
			verb("accept:", cfd.RemoteAddr().String())
			if len(cmd) == 0 {
				err := term3(cfd)
				printerr(err)
			} else {
				err := run3(cfd, cmd[0], cmd[1:]...)
				printerr(err)
			}
		}(ln.Accept())
	}
}
func streammux(ln net.Listener, cmd ...string) {
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
			pimped[p] = i
			pimper <- mrw
		}
	}()
	for {
		sem <- true
		go func(cfd net.Conn, err error) {
			defer func() { <-sem }()
			defer cfd.Close()
			pimper <- cfd
			lol := <-pimper
			if err != nil {
				printerr(err)
				return
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
		}(ln.Accept())
	}
}

var (
	socket string
	proto  string
	cmd    []string
)

func sysfatal(err error) {
	if err == nil {
		return
	}
	printerr(err)
	os.Exit(1)
}

func listener(network string, srv string) (net.Listener, error) {
	if srv != "file" {
		return net.Listen(network, srv)
	}
	return net.FileListener(os.Stdin)
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

	func() {
		verb("open: stdin|net")
		defer verb("close: stdin|net")
		if _, err := io.Copy(rw, os.Stdin); err != nil {
			printerr("stdin|net", err)
		}
	}()
	return <-fin
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

	func() {
		verb("open: cmd|net")
		defer verb("close: cmd|net")
		if _, err := io.Copy(rw, pr); err != nil {
			printerr("cmd|net", err)
		}
	}()

	if err := <-fin; err != nil {
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
	listen - listen on a network interface

SYNOPSIS
	listen [options] port [cmd ...]
	listen [options] [net!host!]port [cmd ...]

DESCRIPTION
	Listen listens for incoming calls and optionally runs
	cmd for each new caller.

	Cmd names an executable and arguments. Callers are wired
	to unique processes created by running cmd for each call; set -s to
	instead attach all callers to a single shared process. An empty cmd
	defaults to -s and all callers interact directly with listen.

	Listen shares file descriptors and environment variables with
	each cmd and injects auxillary variables identifying the local line
	and remote caller: lnet, lhost, lport, rnet, rhost, rport.

NETWORKS
	The net, host, and port are specified with plan9
	dial strings, but ":" may be used instead of "!"

	net!host!port

	Net: Network protocol (tcp)
	Host: The system name or address (*)
	Port: Network port, required for tcp

LIMITERS
	Set -a lim to limit active calls. If lim is reached no more
	calls are answered until a call ends. Set -e n to end calls
	forcefully once lim is reached: n > 0, ends oldest n calls
	while n < 0, ends the |n| most-recent calls. Set -k n to kill
	listen after n callers hang up.

BROADCASTS
	Mux (-m) and dmux (-d) provide a broadcasting for callers and
	processes. Both can utilize an optional ring buffer of n bytes.
	The default is no buffer.

OPTIONS
	-t      Maintain the current user's privledges
	-v      Verbose output

	-a lim  Limit number of active calls to lim
	-e n    After lim, n > 0 ends last n calls, else oldest |n| calls
	-k n    Kill listen after the n'th call ends

	-s      Connect all callers to one shared cmd process
	-m buf  Mux: cmds write to all callers
	-d buf  Demux: cmds read from all callers
	-r      Record traffic to stdout

EXAMPLE
	Listen on tcp port 80 and serve index.html
		listen :80 cat index.html

	Forward connections on port 80 to google.com:80
		listen :80 dial google.com:80

	
BUGS
	Redundant on Plan 9.

	Stdout and stderr are fused together into a miserable gulash.
`)
}
