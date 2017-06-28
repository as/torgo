package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

type parser struct {
	in        chan item
	out       chan func()
	last, tok item
	err       error
	stop      chan error
	cmd       []*Command
	addr      Address
}

// Put

func (p *parser) mustatoi(s string) int64 {
	i, err := strconv.Atoi(s)
	if err != nil {
		p.fatal(err)
	}
	return int64(i)
}
func (p *parser) fatal(why interface{}) {
	switch why := why.(type) {
	default:
		//fmt.Println(why)
		why = why
	}
}

func parseAddr(p *parser) (a Address) {
	a0 := parseSimpleAddr(p)
	//fmt.Printf("parseAddr:1 %s\n", p.tok)
	p.Next()
	//fmt.Printf("parseAddr:2 %s\n", p.tok)
	op, a1 := parseOp(p)
	if op == '\x00' {
		return a0
	}
	p.Next()
	return &Compound{a0: a0, a1: a1, op: op}
}

func parseOp(p *parser) (op byte, a Address) {
	//fmt.Printf("parseOp:1 %s\n", p.tok)
	if p.tok.kind != kindOp {
		return
	}
	v := p.tok.value
	if v == "" {
		eprint("no value" + v)
		return
	}
	if strings.IndexAny(v, "+-;,") == -1 {
		//		eprint(fmt.Sprintf("bad op: %q", v))
	}
	p.Next()
	return v[0], parseSimpleAddr(p)
}

func tryRelative(p *parser) int {
	v := p.tok.value
	k := p.tok
	if k.kind == kindRel {
		defer p.Next()
		if v == "+" {
			return 1
		}
		return -1
	}
	return 0
}

// Put
func parseSimpleAddr(p *parser) (a Address) {
	//fmt.Printf("parseSimpleAddr:1 %s\n", p.tok)
	back := false
	rel := tryRelative(p)
	v := p.tok.value
	k := p.tok
	//fmt.Printf("%s\n", k)
	switch k.kind {
	case kindRegexpBack:
		back = true
		fallthrough
	case kindRegexp:
		re, err := regexp.Compile(v)
		if err != nil {
			p.fatal(err)
			return
		}
		return &Regexp{re, back, rel}
	case kindLineOffset, kindByteOffset:
		i := p.mustatoi(v)
		if rel < 0 {
			i = -i
		}
		if k.kind == kindLineOffset {
			return &Line{i, rel}
		}
		return &Byte{i, rel}
	case kindDot:
		return &Dot{}
	}
	p.err = fmt.Errorf("bad address")
	return
}

type Command struct {
	fn   func(File)
	s    string
	args string
	next *Command
}

func (c *Command) Next() func(File) {
	return c.next.fn
}
func (c *Command) n() *Command {
	return c.next
}

func parseArg(p *parser) (arg string) {
	//fmt.Printf("parseArg: %s\n", p.tok.value)
	p.Next()
	//fmt.Printf("parseArg: %s\n", p.tok.value)
	if p.tok.kind != kindArg {
		//		p.fatal(fmt.Errorf("not arg"))
	}
	return p.tok.value
}

func n(i ...interface{}) (n int, err error) {
	return
}

var eprint = n

// Put
func parseCmd(p *parser) (c *Command) {
	c = new(Command)
	v := p.tok.value
	//fmt.Printf("parseCmd: %s\n", v)
	c.s = v
	switch v {
	case "a", "i":
		argv := parseArg(p)
		c.args = argv
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			b := []byte(argv)
			if v == "i" {
				f.Insert(b, q0)
			} else {
				f.Insert(b, q1)
			}
		}
		return
	case "c":
		argv := parseArg(p)
		c.args = argv
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			f.Delete(q0, q1)
			f.Insert([]byte(argv), q0)
		}
		return
	case "d":
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			f.Delete(q0, q1)
		}
		return
	case "e":
	case "k":
	case "s":
	case "r":
		argv := parseArg(p)
		c.args = argv
		c.fn = func(f File) {
			data, err := ioutil.ReadFile(c.args)
			if err != nil {
				eprint(err)
				return
			}
			q0, q1 := f.Dot()
			if q0 != q1 {
				f.Delete(q0, q1)
			}
			f.Insert(data, q0)
		}
		return
	case "w":
		argv := parseArg(p)
		c.args = argv
		c.fn = func(f File) {
			fd, err := os.Create(argv)
			if err != nil {
				eprint(err)
				return
			}
			defer fd.Close()
			q0, q1 := f.Dot()
			_, err = io.Copy(fd, bytes.NewReader(f.Bytes()[q0:q1]))
			if err != nil {
				eprint(err)
			}
		}
		return
	case "m":
		a1 := parseSimpleAddr(p)
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			p := append([]byte{}, f.Bytes()[q0:q1]...)
			a1.Set(f)
			_, a1 := f.Dot()
			f.Delete(q0, q1)
			f.Insert(p, a1)
		}
		return
	case "t":
		a1 := parseSimpleAddr(p)
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			p := f.Bytes()[q0:q1]
			a1.Set(f)
			_, a1 := f.Dot()
			f.Insert(p, a1)
		}
		return
	case "g":
		argv := parseArg(p)
		c.args = argv
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			ok, err := regexp.Match(argv, f.Bytes()[q0:q1])
			if err != nil {
				panic(err)
			}
			if ok {
				if nextfn := c.Next(); nextfn != nil {
					nextfn(f)
				}
			}
		}
		return
	case "v":
		argv := parseArg(p)
		c.args = argv
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			ok, err := regexp.Match(argv, f.Bytes()[q0:q1])
			if err != nil {
				panic(err)
			}
			if !ok {
				if nextfn := c.Next(); nextfn != nil {
					nextfn(f)
				}
			}
		}
		return
	case "|":
		argv := parseArg(p)
		c.args = argv
		c.fn = func(f File) {
			x := strings.Fields(argv)
			if len(x) == 0 {
				eprint("|: nothing on rhs")
			}
			n := x[0]
			var a []string
			if len(x) > 1 {
				a = x[1:]
			}

			cmd := exec.Command(n, a...)
			q0, q1 := f.Dot()
			f.Delete(q0, q1)
			q1 = q0
			var fd0 io.WriteCloser
			fd1, err := cmd.StdoutPipe()
			if err != nil {
				panic(err)
			}
			fd2, err := cmd.StderrPipe()
			if err != nil {
				panic(err)
			}
			fd0, err = cmd.StdinPipe()
			if err != nil {
				panic(err)
			}
			_, err = io.Copy(fd0, bytes.NewReader(append([]byte{}, f.Bytes()[q0:q1]...)))
			if err != nil {
				eprint(err)
				return
			}
			fd0.Close()
			var wg sync.WaitGroup
			donec := make(chan bool)
			outc := make(chan []byte)
			errc := make(chan []byte)
			wg.Add(2)
			go func() {
				defer wg.Done()
				b := make([]byte, 65536)
				for {
					select {
					case <-donec:
						return
					default:
						n, err := fd1.Read(b)
						if err != nil {
							if err == io.EOF {
								break
							}
							eprint(err)
						}
						outc <- append([]byte{}, b[:n]...)
					}
				}
			}()

			go func() {
				defer wg.Done()
				b := make([]byte, 65536)
				for {
					select {
					case <-donec:
						return
					default:
						n, err := fd2.Read(b)
						if err != nil {
							if err == io.EOF {
								break
							}
						}
						errc <- append([]byte{}, b[:n]...)
					}
				}
			}()
			go func() {
				cmd.Start()
				cmd.Wait()
				close(donec)
			}()
		Loop:
			for {
				select {
				case p := <-outc:
					f.Insert(p, q1)
					q1 += int64(len(p))
				case p := <-errc:
					f.Insert(p, q1)
					q1 += int64(len(p))
				case <-donec:
					break Loop
				}
			}
		}
		return
	case ">":
		argv := parseArg(p)
		c.args = argv
		c.fn = func(f File) {
			fd, err := os.Create(argv)
			if err != nil {
				eprint(err)
				return
			}
			defer fd.Close()
			q0, q1 := f.Dot()
			_, err = io.Copy(fd, bytes.NewReader(f.Bytes()[q0:q1]))
			if err != nil {
				eprint(err)
			}
		}
		return
	case "x":
		argv := parseArg(p)
		c.args = argv
		re, err := regexp.Compile(argv)
		if err != nil {
			p.fatal(err)
			return
		}
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			x0, x1 := q0, q0
			for {
				if x1 > q1 {
					break
				}
				buf := bytes.NewReader(f.Bytes()[x1:q1])
				loc := re.FindReaderIndex(buf)
				if loc == nil {
					eprint("not found")
					break
				}
				x0, x1 = int64(loc[0])+x1, int64(loc[1])+x1
				f.Select(x0, x1)
				a := len(f.Bytes())
				if nextfn := c.Next(); nextfn != nil {
					nextfn(f)
				}
				//d0, d1 := f.Dot()
				b := len(f.Bytes()) - a
				x1 += int64(b) //+ (d1-d0)
				q1 += int64(b)
				x0 = x1
			}
		}
		return
	case "y":
		argv := parseArg(p)
		c.args = argv
		re, err := regexp.Compile(argv)
		if err != nil {
			p.fatal(err)
			return
		}
		c.fn = func(f File) {
			q0, q1 := f.Dot()
			x0, x1 := q0, q0
			y0, y1 := q0, q0
			for {
				if x1 > q1 {
					break
				}
				buf := bytes.NewReader(f.Bytes()[x1:q1])
				loc := re.FindReaderIndex(buf)
				if loc == nil {
					if x1 < q1 {
						f.Select(x1, q1)
						if nextfn := c.Next(); nextfn != nil {
							nextfn(f)
						}
					}
					break
				}
				y0 = x0
				x0, x1 = int64(loc[0])+x1, int64(loc[1])+x1
				y1 = x0

				f.Select(y0, y1)
				a := len(f.Bytes())
				if nextfn := c.Next(); nextfn != nil {
					nextfn(f)
				}
				//d0, d1 := f.Dot()
				b := len(f.Bytes()) - a
				x1 += int64(b) //+ (d1-d0)
				q1 += int64(b)
				x0 = x1
			}
		}
		return
	}
	return nil
}

func (p *parser) Next() *item {
	p.last = p.tok
	p.tok = <-p.in
	return &p.tok
}

func parse(i chan item) *parser {
	p := &parser{
		in:   i,
		stop: make(chan error),
	}
	go p.run()
	return p
}

func (p *parser) run() {
	tok := p.Next()
	if tok.kind == kindEof || p.err != nil {
		if tok.kind == kindEof {
			//			p.fatal(fmt.Errorf("run: unexpected eof"))
			return
		}
		//		p.fatal(fmt.Errorf("run: %s", p.err))
		return
	}
	p.addr = parseAddr(p)
	for {
		c := parseCmd(p)
		if c == nil {
			break
		}
		p.cmd = append(p.cmd, c)
		//		eprint(fmt.Sprintf("(%s) %#v and cmd is %#v\n", tok, p.addr, c))
		p.Next()
	}
	p.stop <- p.err
	close(p.stop)
}

func compileAddr(a Address) func(f File) {
	return a.Set
}

func compile(p *parser) (cmd *Command) {
	for i := range p.cmd {
		if i+1 == len(p.cmd) {
			break
		}
		p.cmd[i].next = p.cmd[i+1]
	}
	fn := func(f File) {
		addr := compileAddr(p.addr)
		if addr != nil {
			addr(f)
		}
		if p.cmd != nil && p.cmd[0] != nil && p.cmd[0].fn != nil {
			p.cmd[0].fn(f)
		}
	}
	return &Command{fn: fn}
}

func Cmdparse(s string) (cmd *Command) {
	return cmdparse(s)
}
func cmdparse(s string) (cmd *Command) {
	_, itemc := lex("cmd", s)
	p := parse(itemc)
	err := <-p.stop
	if err != nil {
		eprint(err)
		return nil
	}
	return compile(p)
}
