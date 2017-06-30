package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
)

// Op maps a file descriptor to a command
type Op struct {
	fd   uintptr
	mode string
	cmd  string
}

// parser massages the command line arguments
// into redirections and shell commands, the
// final state is a list of redirs to perform
// before running finalcmd
type parser struct {
	fd   uintptr
	mode string
	cmd  string
	tok  string

	redirs   []Op
	source   []string
	finalcmd []string
}

// Parse parses the input list for redirections
func Parse(s []string) *parser {
	if s == nil {
		no(fmt.Errorf("empty input to parser"))
	}
	p := &parser{source: s}
	err := p.parseFd()
	no(err)

	return p
}

// parseFd parses a file descriptor, it calls
// parseCmd to consume a shell command or parseFinal
// to consume the final command.
func (p *parser) parseFd() (err error) {
	defer un(trace("parseFd"))
	const valid = "0123456789"
	token := p.next()
	if token == "" {
		return nil // EOF
	}
	if !strings.HasPrefix(token, "-") {
		return p.parseFinal()
	}

	i := 0
	for _, v := range token[1:] {
		if strings.IndexRune(valid, rune(v)) == -1 {
			break
		}
		i++
	}
	if i == 0 {
		no(fmt.Errorf("its just a dash"))
	}
	i++
	fd, err := strconv.Atoi(token[1:i])
	if err != nil {
		return err
	}
	p.fd, p.mode = uintptr(fd), token[i:]
	return p.parseCmd()
}

// parseCmd parses and consumes a command. It calls parseFd
// to consume the next file descriptor or final command.
func (p *parser) parseCmd() error {
	defer un(trace("parseCmd"))
	p.cmd = p.next()
	p.redirs = append(p.redirs, Op{fd: p.fd, mode: p.mode, cmd: p.cmd})
	return p.parseFd()
}

// parseFinal parses the final command. Any tokens proceeding it are
// invalid and trigger an error.
func (p *parser) parseFinal() error {
	defer un(trace("parseFinal"))

	// Let exec handle the last argument
	for p.tok != ""{
		p.finalcmd = append(p.finalcmd, p.tok)
		p.tok = p.next()
	}
	return nil
}

// next advances the parser by one field.
func (p *parser) next() (tok string) {
	defer un(trace("next"))
	if p.source == nil || len(p.source) == 0 {
		return ""
	}
	p.tok = p.source[0]
	p.source = p.source[1:]
	return p.tok
}

func un(s string) {
	debug("leave: %s", s)
}
func trace(s string) string {
	debug("enter: %s", s)
	return s
}

func no(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
