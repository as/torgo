package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	leftMeta  = "{"
	rightMeta = "}"
	backTick  = "`"
	eof       = '‡'
)

const (
	itemError itemType = iota
	itemDot
	itemEOF
	itemLHS
	itemRHS
	itemEquals // 5
	itemLeftMeta
	itemRightMeta
	itemNumber
	itemPipe
	itemText // 10
	itemBackTick
	itemHereString
	itemFnStart
	itemFnInside
	itemEnv // 15
)

type itemType int

type statefn func(*lexer) statefn

type item struct {
	typ itemType
	val string
}

type lexer struct {
	name  string
	input string
	start int
	pos   int
	width int
	items chan item
}

func (i item) String() string {
	return fmt.Sprintf("%d %s", i.typ, i.val)
}

func lex(name, input string) (*lexer, chan item) {
	l := &lexer{
		name:  name,
		input: input,
		items: make(chan item),
	}
	go l.run() // run state machine
	return l, l.items
}

func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) errorf(format string, args ...interface{}) statefn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) run() {
	for state := lexText; state != nil; {
		state = state(l)
	}
	close(l.items)
}

func lexIdentifier(l *lexer) statefn {
	l.acceptRun("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	if isAlphaNumeric(l.peek()) {
		l.next()
		return l.errorf("bad identifier syntax: %q",
			l.input[l.start:l.pos])
	}
	l.emit(itemText)
	return lexInsideAction
}

func lexNumber(l *lexer) statefn {
	l.accept("+-")
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") {
		digits += "abcdefABCDEF"
	}
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}
	// imaginary
	if isAlphaNumeric(l.peek()) {
		l.next()
		return l.errorf("bad number syntax: %q",
			l.input[l.start:l.pos])
	}
	l.emit(itemNumber)
	return lexInsideAction
}

func space(r rune) bool {
	return unicode.IsSpace(r)
}

func ignoreSpaces(l *lexer) {
	if l.accept(" 	") {
		l.acceptRun(" 	")
		l.ignore()
	}
}

func lexText(l *lexer) statefn {
	ignoreSpaces(l)
	l.acceptRun("/abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if l.pos == l.start && l.next() == eof {
		l.emit(itemEOF)
		return nil
	}
	ignoreSpaces(l)
	l.emit(itemText)

	fmt.Printf("%#v", l.input[l.pos:])
	switch r := l.peek(); {
	case r == '=':
		return lexEquals
	case r == '$':
		return lexEnv
	case r == eof:
		if l.pos == l.start {
			println("itemEOF")
			l.emit(itemEOF)
			return nil
		}
	}
	return nil
}

func lexEquals(l *lexer) statefn {
	l.acceptRun("/abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	l.emit(itemText)
	return lexText
}

func lexEnv(l *lexer) statefn {
	if !l.accept("$") {
		return l.errorf("Invalid variable", l.input[:])
	}
	l.ignore()
	l.acceptRun("/abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	l.emit(itemEnv)
	return lexText
}

func lexLeftMeta(l *lexer) statefn {
	l.pos += len(leftMeta)
	l.emit(itemLeftMeta)
	return lexInsideAction
}

func lexRightMeta(l *lexer) statefn {
	l.pos += len(rightMeta)
	l.emit(itemRightMeta)
	return lexText
}

func lexInsideAction(l *lexer) statefn {
	// Either num, string, or id
	for {
		if strings.HasPrefix(l.input[l.pos:], rightMeta) {
			return lexRightMeta
		}
		switch r := l.next(); {
		case r == eof || r == '\n':
			return l.errorf("unclosed action")
		case unicode.IsSpace(r):
			l.ignore()
		case r == '|':
			l.emit(itemPipe)
		case r == '+' || r == '-' || '0' <= r && r <= '9':
			l.backup()
			return lexNumber
		case isAlphaNumeric(r):
			l.backup()
			return lexIdentifier
		}
	}
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func prompt() {
	fmt.Print("; ")
}

func (cl cmdline) push(i item, fn func()) {
	cl = append(cl, fn)
}

type cmdline []cmd

type cmd func()

type Op int

func (p *parser) Push(fn func()) {
	p.cl = append(p.cl, fn)
}

func (p *parser) Next() *item {
	p.last = p.tok
	p.tok = <-p.in
	return &p.tok
}

func (p *parser) String() string {
	return fmt.Sprintf("last: %v	tok: %v\n", p.last, p.tok)
}

type parser struct {
	in        chan item
	cl        cmdline
	out       chan func()
	last, tok item
}

func parse(i chan item) *parser {
	p := &parser{
		in: i,
	}
	go p.run()
	return p
}

var parseActs = map[itemType]func(p *parser) func(){
	itemEnv:    parseEnv,
	itemText:   parseCmd,
	itemEquals: parseEquals,
	itemEOF:    nil,
}

func pl(i ...interface{}) {
	fmt.Println(i...)
}

func (p *parser) run() {
	for {
		tok := p.Next()
		pl("parser: next tok: ", tok)
		fn, ok := parseActs[tok.typ]
		if !ok {
			fmt.Println("no action for", tok)
			return
		}
		if fn == nil {
			fmt.Println("eof")
			return
		}
		fmt.Println(p)
		fn(p)()
		//		p.Push(fn(p))
	}
}

func parseEnv(p *parser) func() {
	return func() {
		extract(p.tok)
	}
}

func parseCmd(p *parser) func() {
	return func() {
		cmdexec(p.tok)
	}
}

func parseEquals(p *parser) func() {
	lhs := p.tok
	if lhs.typ != itemText {
		fmt.Println("assign: crap LHS:", lhs.typ)
		return nil
	}
	eq := p.Next()
	pl("equals token:", eq)
	rhs := p.Next()
	if rhs.typ != itemText {
		fmt.Println("assign: crap RHS")
		return nil
	}
	return func() {
		assign(&lhs, rhs)
	}
}

func assign(r, l *item) {
	fmt.Printf("assign %v to %v\n", r, l)
	err := os.Setenv(r.val, l.val)
	if err != nil {
		fmt.Println(err)
	}
}

func extract(i item) item {
	return item{itemText, os.Getenv(i.val)}
}

//
// Execution

func cmdexec(i item) {
	switch i.val {
	case "cd":
		bcd(i)
	case "echo":
		becho(i)
	case "exit":
		bexit()
	default:
		run(i)
	}
}

func bcd(i item) {
	/*
		if len(args) > 1 {
			fmt.Println("usage: cd [dir]")
			return
		}
	*/
	err := os.Chdir(i.val)
	if err != nil {
		fmt.Println(err)
	}
}

func becho(i item) {
	fmt.Println(i.val)
}

func bexit() {
	os.Exit(0)
}

func run(i item) {
	c := exec.Command(i.val)
	if c == nil {
		fmt.Println("no:", i.val)
	}
	out, err := c.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Print(string(out))
}

func main() {
	s := bufio.NewScanner(os.Stdin)
	var p *parser
	prompt()
	for s.Scan() {
		l, item := lex("test", s.Text())
		p = parse(item)
		l = l
	}
	p = p
	os.Exit(0)
}
