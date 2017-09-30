package main

/*
	go build ffs.go && cp ffs /bin/ &&
		cat /home/Manifest.xml | ffs /tmp/d1 | rc
	GOOS=windows go build ffs.go
*/

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
)

import (
	"github.com/as/mute"
)

const (
	Prefix     = "ffs: "
	BufferSize = 65535
	Debug      = false
)

type action func(*decoder) action

type decoder struct {
	err error

	x     *xml.Decoder
	token xml.Token
	selem xml.StartElement
	eelem xml.EndElement
	attr  xml.Attr

	pc    int
	depth int
	spool []*token
	stack []string
}

type token struct {
	name string
	action
	trigger xml.Token
}

var auxfn map[string]action

func walk(t *Node, fn func(n *Node)) {
	if t == nil {
		return
	}
	fn(t)
	for _, v := range t.kids {
		walk(v, fn)
	}
}

func main() {
	a := f.Args()
	if len(a) == 0 || args.h || args.q {
		usage()
		os.Exit(1)
	}

	d := &decoder{
		x: xml.NewDecoder(os.Stdin),
	}

	Tree = &Node{name: a[0]}
	Tree.dad = Tree
	tp = Tree

	auxinit()
	for state := begin; state != nil; d.pc++ {
		state = state(d)
	}
	dumpcmd(d)

	os.Exit(0)
}

//
// Second-order auxillary actions.
// Called by the decoder
//

func auxinit() {

	auxfn = map[string]action{
		"Begin":        noop,
		"Err":          noop,
		"EOF":          noop,
		"StartElement": remkdir,
		"EndElement":   repopdir,
		"EmptyElement": emptyelement,
		"CharData":     noop,
		"Comment":      noop,
		"Directive":    noop,
		"ProcInst":     noop,
		"Attribute":    rattr,
		"NilAttrName":  rbsattr,
		"NilAttrValue": rbadattr,
	}
}

func remkdir(d *decoder) action {
	node := mknode(d.Top())
	tp.AddKid(node)
	tp = node
	return nil
}

func repopdir(d *decoder) action {
	tp = tp.dad
	return noop
}

func emptyelement(d *decoder) action {
	debugerr("element has no attributes")
	return classify
}

func rattr(d *decoder) action {
	tp.attrs = append(tp.attrs, d.attr)
	return nil
}

func rbadattr(d *decoder) action {
	debugerr("attr: no value: ", d.attr.Name.Local)
	return nil
}

func rbsattr(d *decoder) action {
	debugerr("WOW!: ", d.attr)
	return nil
}

var Tree, tp *Node

type Node struct {
	name  string
	attrs []xml.Attr
	id    int
	kids  []*Node
	dad   *Node
}

func (n *Node) String() string {
	return n.name
}

func (n *Node) AddKid(k *Node) {
	n.kids = append(n.kids, k)
	k.dad = n
}

func (n *Node) NKids() int {
	return len(n.kids)
}

func (n *Node) Runt() *Node {
	i := n.NKids() - 1
	if i < 0 {
		return nil
	}
	return n.kids[i]
}

func mknode(s string) *Node {
	return &Node{name: s}
}

//
// dumpcmd prints commands to create a file
// system representation of the XML.
// The tree is walked thrice.
//

func dumpcmd(d *decoder) {
	m := make(map[*Node]map[string]int)

	// Allocate
	walk(Tree, func(n *Node) {
		m[n] = make(map[string]int)
	})

	// Preprocess
	walk(Tree, func(n *Node) {
		nsame := m[n.dad][n.name]
		m[n.dad][n.name]++
		n.id = nsame
	})

	// Create functions for the final walk:
	// downfn: creates dirs and files
	// upfn: ascends the file system

	downfn := func(n *Node) {
		nsame := m[n.dad][n.name]
		pad := fmt.Sprint(nsame)
		dir := n.name
		if nsame > 1 {
			dir += fmt.Sprintf(".%0*d", len(pad), n.id)
		}
		d.Println("mkdir", dir)
		d.Println("cd", dir)
		d.depth++
		for _, v := range n.attrs {
			d.Println(attrecho(v))
			v = v
		}
	}

	upfn := func(n *Node) {
		d.depth--
		d.Println("cd ..")
	}

	// Print
	walk(Tree, func(n *Node) {
		downfn(n)
		if n == n.dad.Runt() && n.dad != Tree {
			upfn(n)
		}
		if n.kids == nil {
			upfn(n)
		}
	})
}

func attrecho(a xml.Attr) string {
	qn, qv := q(a.Name.Local), q(a.Value)
	if a.Value == "" {
		qn += c("Empty")
	}
	return fmt.Sprintf("echo %s > %s", qv, qn)
}

func (d *decoder) tokenize(x interface{}) (t token) {
	switch z := x.(type) {
	case xml.StartElement:
		t = token{"StartElement", startelement, x}
	case xml.EndElement:
		t = token{"EndElement", endelement, x}
	case xml.CharData:
		t = token{"CharData", chardata, x}
	case xml.Comment:
		t = token{"Comment", comment, x}
	case xml.Directive:
		t = token{"Directive", directive, x}
	case xml.ProcInst:
		t = token{"ProcInst", procinst, x}
	case string:
		s := string(z)
		fn, ok := auxfn[s]
		if !ok {
			printerr("bad aux fn:", s)
			os.Exit(1)
		}
		t = token{s, fn, x}
	default:
		debugerr("eof", t)
		t = token{"EOF", eof, x}
	}
	d.spool = append(d.spool, &t)
	return t
}

func (t token) Action() action {
	return func(d *decoder) action {
		t.start(d)
		defer t.stop(d)
		return t.action(d)
	}
}

func (t token) String() string {
	return t.name
}

func (t token) start(d *decoder) action {
	return noop
}

func (t token) stop(d *decoder) action {
	return noop
}

func (d *decoder) Aux() action {
	tok := d.LastTokenString()
	return d.Call(tok)
}

func (d *decoder) Call(fn string) action {
	a := d.tokenize(fn).Action()
	return a(d)
}

func (d *decoder) LastToken() *token {
	l := len(d.spool)
	if l == 0 {
		return nil
	}
	return d.spool[l-1]
}

func (d *decoder) LastTokenString() string {
	t := d.LastToken()
	if t == nil {
		return ""
	}
	return t.String()
}

func (d *decoder) Push(name string) string {
	d.stack = append(d.stack, name)
	return name
}

func (d *decoder) Pop() string {
	l := len(d.stack)
	if l <= 0 {
		return ""
	}
	d.stack = d.stack[:l-1]
	return d.Top()
}

func (d *decoder) Top() string {
	l := len(d.stack)
	if l == 0 {
		return ""
	}
	return d.stack[l-1]
}

func (d *decoder) Token() (xml.Token, error) {
	tmp, err := d.x.Token()
	if err != nil {
		d.token = nil
		return nil, err
	}
	d.token = xml.CopyToken(tmp)
	return d.token, nil
}

func classify(d *decoder) action {
	t, err := d.Token()
	if err != nil {
		return errfn(d)
	}
	switch v := t.(type) {
	default:
		tok := d.tokenize(v)
		return tok.Action()
	}
}

func startelement(d *decoder) action {
	d.selem = d.token.(xml.StartElement)
	name := d.selem.Name.Local
	d.Push(name)
	d.depth++
	d.Aux()
	if len(d.selem.Attr) == 0 {
		return d.Call("EmptyElement")
	}
	return attr
}

func attr(d *decoder) action {
	for _, v := range d.selem.Attr {
		d.attr = v
		if v.Name.Local == "" {
			d.Call("NilAttrName")
		}
		if v.Value == "" {
			d.Call("NilAttrValue")
		}
		d.Call("Attribute")
	}
	return classify
}

func endelement(d *decoder) action {
	d.Call("EndElement")
	d.depth--
	d.Pop()
	return classify
}

func chardata(d *decoder) action {
	e := d.token.(xml.CharData)
	e = e
	return classify
}

func comment(d *decoder) action {
	e := d.token.(xml.Comment)
	e = e
	return classify
}

func procinst(d *decoder) action {
	return classify
}

func directive(d *decoder) action {
	return classify
}

func eof(d *decoder) action {
	return nil
}

func begin(d *decoder) action {
	return classify
}

func noop(d *decoder) action {
	return nil
}

func reflect(d *decoder) action {
	debugerr(d.LastToken())
	return nil
}

func errfn(d *decoder) action {
	printerr(d.err)
	return eof
}

// q returns a quoted variant of string s
// suitable for the rc shell
func q(s string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`"%s"`, s)
	}
	return fmt.Sprintf("'%s'", s)
}

func c(s string) string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("::%s", s)
	}
	return fmt.Sprintf("#%s", s)
}

var args struct {
	h, q bool
	r    bool
	k    string
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)

	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.StringVar(&args.k, "k", "", "")

	f.BoolVar(&args.r, "r", false, "")

	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}

}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func debugerr(v ...interface{}) {
	if Debug {
		printerr(v)
	}
}
func (d *decoder) Println(i ...interface{}) {
	prefix := strings.Repeat("	", d.depth)
	fmt.Print(prefix)
	fmt.Println(i...)
}

func (d *decoder) Printf(f string, i ...interface{}) {
	prefix := strings.Repeat("	", d.depth)
	fmt.Print(prefix)
	fmt.Printf(f, i...)
}

func usage() {
	fmt.Println(`
NAME
	ffs - Convert an atrocious document into a filesystem

SYNOPSIS
	ffs dir

DESCRIPTION
	ffs reads in the devil's format froms stdin and converts it to a
	filesystem rooted at dir.

OPTIONS
	-g	Generate rc commands that perform the conversion
	(NO) -r	Reverse: Convert a filesystem into XML

EXAMPLE
	cat Nonsense.xml | ffs nonsense

BUGS
	XML is a bug

SEE ALSO

`)
}
