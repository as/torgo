package xo

import (
	"fmt"
	"log"
	"strings"
)

type Tok string

type cmd struct {
	re string
	op Tok
}

type state struct {
	lastop  string
	lastcmd cmd
	prog    []cmd
}

const (
	TokNone Tok = ""
	TokBeg      = "^"
	TokEnd      = ""
	TokSla      = "/"
	TokCom      = ","
	TokSem      = ";"
	TokAdd      = "+"
	TokSub      = "-"
)

func (s *state) parseexp(in string) (ex string, e int, err error) {
	defer func() { debugerr("findexp returns:", ex, err) }()
	var b int
	if len(in) < 2 {
		return "", -1, fmt.Errorf("findexp: short string")
	}
	if b = strings.Index(in, "/"); b < 0 {
		return "", -1, fmt.Errorf("findexp: not found")
	}
	if e = strings.Index(in[1:], "/"); e < 0 {
		return "", -1, fmt.Errorf("findexp: unterminated")
	}
	e++
	return in[b+1 : e], e, err
}
func (s *state) findop(in string) Tok {
	i := strings.IndexAny(in, ";,+-")
	if i < 0 {
		return TokEnd
	}
	return Tok(in[:1])
}

func (s *state) hasslash(in string) bool {
	i := strings.IndexAny(in, "/")
	return i >= 0
}
func (s *state) inslash(in string) (b bool) {
	if s.inop(in) {
		return false
	}
	return s.hasslash(in)
}
func (s *state) hasop(in string) bool {
	o := s.findop(in)
	return o == TokAdd || o == TokSub || o == TokSem || o == TokCom
}
func (s *state) inop(in string) bool {
	i := strings.IndexAny(in, "/,+;-")
	return i >= 0 && in[i] != '/'
}
func (s *state) ineof(in string) bool {
	return in == ""
}

func (s *state) check(in string, t Tok) bool {
	if len(in) == 0 {
		return t == TokEnd
	}
	if len(t) == 0 {
		return len(in) == 0
	}
	return Tok(in[:len(t)]) == t
}

func (c cmd) String() string {
	return fmt.Sprintf("cmd: re: [%s] op [%s]\n", c.re, c.op)
}

func (s *state) nuke() {
	s.lastop = ""
	s.lastcmd = cmd{}
	s.prog = nil
}

func (s *state) begin(in string) {
	func() { debugerr("begin", in) }()
	defer func() { debugerr("begin returns") }()
	s.lastcmd.op = "+"
	s.parseop(in)
	return
}
func (s *state) parseslash(in string) {
	func() { debugerr("SLASH", s) }()
	defer func() {
		debugerr("		SLASH")
	}()
	switch {
	case s.ineof(in):
		debugerr("evof")
		return
	case s.inop(in):
		debugerr("lex: insert $")
		s.parseop("/$/" + in)
		return
	case !s.inslash(in):
		debugerr("parse error")
		return
	}
	re, e, err := s.parseexp(in)
	if err != nil {
		debugerr("slash parse error")
		return
	}
	s.lastcmd.re = re
	s.appendcmd()
	s.parseop(in[e+1:])
}
func (s *state) parseop(in string) {
	func() { debugerr("PARSE", s) }()
	defer func() {
		debugerr("		PARSE")
	}()
	switch {
	case s.ineof(in):
		debugerr("eof")
		return
	case s.inslash(in):
		s.parseop(string(s.lastcmd.op) + in)
		return
	case !s.inop(in):
		debugerr("parseop: not in op", s)
		return
	}
	op := s.findop(in)
	if op == TokEnd {
		debugerr("eof2")
		return
	}
	s.lastcmd.op = op
	s.parseslash(in[1:])
}
func (s *state) appendcmd() {
	s.prog = append(s.prog, s.lastcmd)
	debugerr("append call slash", s.prog)
}

func (s *state) reg(in string) {
	defer func() { debugerr("reg: pre", s.prog) }()
	func() { debugerr("reg: post", s.prog) }()
	s.lastcmd.re = in
}

func parseaddr(in string) (c []cmd, err error) {
	s := &state{}
	s.begin(in)
	debugerr(fmt.Sprintf("program: %#v\n", s.prog))
	return s.prog, err
}

func debugerr(v ...interface{}) {
	if debug {
		log.Println(v)
	}
}

var debug = false // true false
