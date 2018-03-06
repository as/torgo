package main

// Digraph traversal. Adapted from the tool in the Go source tree. Same license as the Go language.
// TODO: add example

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"unicode"
	"unicode/utf8"
)

import (
	"github.com/as/mute"
)

const (
	Prefix    = "digra: "
)

var f *flag.FlagSet
var args struct {
	h, q bool
	f    string
}
func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	err := mute.Parse(f, os.Args[1:])
	fatal(err)
}
func main() {
	var (
		g graph
	)
	go func() {
		fd, err := os.Open(f.Args()[0])
		fatal(err)
		g, err = parse(fd)
		fatal(err)
	}()
	in := bufio.NewScanner(stdin)
	for in.Scan() {
		words, err := split(in.Text())
		fatal(err)
		if len(words) == 0 {
			continue
		}
		err = digraph(g, words[0], words[1:])
	}
}

type nodes []string


func (l nodes) println(sep string) {
	for i, label := range l {
		if i > 0 {
			fmt.Fprint(stdout, sep)
		}
		fmt.Fprint(stdout, label)
	}
	fmt.Fprintln(stdout)
}

type nodeset map[string]bool

func (s nodeset) sort() nodes {
	labels := make(nodes, len(s))
	var i int
	for label := range s {
		labels[i] = label
		i++
	}
	sort.Strings(labels)
	return labels
}

func (s nodeset) addAll(x nodeset) {
	for label := range x {
		s[label] = true
	}
}

// A graph maps nodes to the non-nil set of their immediate successors.
type graph map[string]nodeset

func (g graph) addNode(label string) nodeset {
	edges := g[label]
	if edges == nil {
		edges = make(nodeset)
		g[label] = edges
	}
	return edges
}

func (g graph) addEdges(from string, to ...string) {
	edges := g.addNode(from)
	for _, to := range to {
		g.addNode(to)
		edges[to] = true
	}
}

func (g graph) reachableFrom(roots nodeset) nodeset {
	seen := make(nodeset)
	var visit func(label string)
	visit = func(label string) {
		if !seen[label] {
			seen[label] = true
			for e := range g[label] {
				visit(e)
			}
		}
	}
	for root := range roots {
		visit(root)
	}
	return seen
}

func (g graph) transpose() graph {
	rev := make(graph)
	for label, edges := range g {
		rev.addNode(label)
		for succ := range edges {
			rev.addEdges(succ, label)
		}
	}
	return rev
}

func (g graph) sccs() []nodeset {
	// Kosaraju's algorithm---Tarjan is overkill here.

	// Forward pass.
	S := make(nodes, 0, len(g)) // postorder stack
	seen := make(nodeset)
	var visit func(label string)
	visit = func(label string) {
		if !seen[label] {
			seen[label] = true
			for e := range g[label] {
				visit(e)
			}
			S = append(S, label)
		}
	}
	for label := range g {
		visit(label)
	}

	// Reverse pass.
	rev := g.transpose()
	var scc nodeset
	seen = make(nodeset)
	var rvisit func(label string)
	rvisit = func(label string) {
		if !seen[label] {
			seen[label] = true
			scc[label] = true
			for e := range rev[label] {
				rvisit(e)
			}
		}
	}
	var sccs []nodeset
	for len(S) > 0 {
		top := S[len(S)-1]
		S = S[:len(S)-1] // pop
		if !seen[top] {
			scc = make(nodeset)
			rvisit(top)
			sccs = append(sccs, scc)
		}
	}
	return sccs
}

func parse(rd io.Reader) (graph, error) {
	g := make(graph)

	var linenum int
	in := bufio.NewScanner(rd)
	for in.Scan() {
		linenum++
		// Split into words, honoring double-quotes per Go spec.
		words, err := split(in.Text())
		if err != nil {
			return nil, fmt.Errorf("at line %d: %v", linenum, err)
		}
		if len(words) > 0 {
			g.addEdges(words[0], words[1:]...)
		}
	}
	if err := in.Err(); err != nil {
		return nil, err
	}
	return g, nil
}

var stdin io.Reader = os.Stdin
var stdout io.Writer = os.Stdout

func digraph(g graph, cmd string, args []string) error {
	// Parse the input graph.

	// Parse the command line.
	switch cmd {
	case "help":
		help()
	case "node":
		if len(args) != 0 {
			return fmt.Errorf("usage: digraph nodes")
		}
		nodes := make(nodeset)
		for label := range g {
			nodes[label] = true
		}
		nodes.sort().println("\n")

	case "deg":
		if len(args) != 0 {
			return fmt.Errorf("usage: digraph degree")
		}
		nodes := make(nodeset)
		for label := range g {
			nodes[label] = true
		}
		rev := g.transpose()
		for _, label := range nodes.sort() {
			fmt.Fprintf(stdout, "%d\t%d\t%s\n", len(rev[label]), len(g[label]), label)
		}

	case "next", "prev":
		if len(args) == 0 {
			return fmt.Errorf("usage: digraph %s <label> ...", cmd)
		}
		g := g
		if cmd == "prev" {
			g = g.transpose()
		}
		result := make(nodeset)
		for _, root := range args {
			edges := g[root]
			if edges == nil {
				return fmt.Errorf("no such node %q", root)
			}
			result.addAll(edges)
		}
		result.sort().println("\n")

	case "front", "back":
		if len(args) == 0 {
			return fmt.Errorf("usage: digraph %s <label> ...", cmd)
		}
		roots := make(nodeset)
		for _, root := range args {
			if g[root] == nil {
				return fmt.Errorf("no such node %q", root)
			}
			roots[root] = true
		}
		g := g
		if cmd == "back" {
			g = g.transpose()
		}
		g.reachableFrom(roots).sort().println("\n")

	case "walk":
		if len(args) != 2 {
			return fmt.Errorf("usage: digraph walk <from> <to>")
		}
		from, to := args[0], args[1]
		if g[from] == nil {
			return fmt.Errorf("no such 'from' node %q", from)
		}
		if g[to] == nil {
			return fmt.Errorf("no such 'to' node %q", to)
		}

		seen := make(nodeset)
		var visit func(path nodes, label string) bool
		visit = func(path nodes, label string) bool {
			if !seen[label] {
				seen[label] = true
				if label == to {
					append(path, label).println("\n")
					return true // unwind
				}
				for e := range g[label] {
					if visit(append(path, label), e) {
						return true
					}
				}
			}
			return false
		}
		if !visit(make(nodes, 0, 100), from) {
			return fmt.Errorf("no path from %q to %q", args[0], args[1])
		}

	case "walkall":
		if len(args) != 2 {
			return fmt.Errorf("usage: digraph walkall <from> <to>")
		}
		from, to := args[0], args[1]
		if g[from] == nil {
			return fmt.Errorf("no such 'from' node %q", from)
		}
		if g[to] == nil {
			return fmt.Errorf("no such 'to' node %q", to)
		}

		seen := make(nodeset) // value of seen[x] indicates whether x is on some path to 'to'
		var visit func(label string) bool
		visit = func(label string) bool {
			reachesTo, ok := seen[label]
			if !ok {
				reachesTo = label == to

				seen[label] = reachesTo
				for e := range g[label] {
					if visit(e) {
						reachesTo = true
					}
				}
				seen[label] = reachesTo
			}
			return reachesTo
		}
		if !visit(from) {
			return fmt.Errorf("no path from %q to %q", from, to)
		}
		for label, reachesTo := range seen {
			if !reachesTo {
				delete(seen, label)
			}
		}
		seen.sort().println("\n")

	case "strongs":
		if len(args) != 0 {
			return fmt.Errorf("usage: strongs")
		}
		for _, scc := range g.sccs() {
			scc.sort().println(" ")
		}

	case "strong":
		if len(args) != 1 {
			return fmt.Errorf("usage: strong <label>")
		}
		label := args[0]
		if g[label] == nil {
			return fmt.Errorf("no such node %q", label)
		}
		for _, scc := range g.sccs() {
			if scc[label] {
				scc.sort().println("\n")
				break
			}
		}

	default:
		return fmt.Errorf("no such command %q", cmd)
	}

	return nil
}

// -- Utilities --------------------------------------------------------

// split splits a line into words, which are generally separated by
// spaces, but Go-style double-quoted string literals are also supported.
// (This approximates the behaviour of the Bourne shell.)
//
//   `one "two three"` -> ["one" "two three"]
//   `a"\n"b` -> ["a\nb"]
//
func split(line string) ([]string, error) {
	var (
		words   []string
		inWord  bool
		current bytes.Buffer
	)

	for len(line) > 0 {
		r, size := utf8.DecodeRuneInString(line)
		if unicode.IsSpace(r) {
			if inWord {
				words = append(words, current.String())
				current.Reset()
				inWord = false
			}
		} else if r == '"' {
			var ok bool
			size, ok = quotedLength(line)
			if !ok {
				return nil, errors.New("invalid quotation")
			}
			s, err := strconv.Unquote(line[:size])
			if err != nil {
				return nil, err
			}
			current.WriteString(s)
			inWord = true
		} else {
			current.WriteRune(r)
			inWord = true
		}
		line = line[size:]
	}
	if inWord {
		words = append(words, current.String())
	}
	return words, nil
}

// quotedLength returns the length in bytes of the prefix of input that
// contain a possibly-valid double-quoted Go string literal.
//
// On success, n is at least two (""); input[:n] may be passed to
// strconv.Unquote to interpret its value, and input[n:] contains the
// rest of the input.
//
// On failure, quotedLength returns false, and the entire input can be
// passed to strconv.Unquote if an informative error message is desired.
//
// quotedLength does not and need not detect all errors, such as
// invalid hex or octal escape sequences, since it assumes
// strconv.Unquote will be applied to the prefix.  It guarantees only
// that if there is a prefix of input containing a valid string literal,
// its length is returned.
//
// TODO(adonovan): move this into a strconv-like utility package.
//
func quotedLength(input string) (n int, ok bool) {
	var offset int

	// next returns the rune at offset, or -1 on EOF.
	// offset advances to just after that rune.
	next := func() rune {
		if offset < len(input) {
			r, size := utf8.DecodeRuneInString(input[offset:])
			offset += size
			return r
		}
		return -1
	}

	if next() != '"' {
		return // error: not a quotation
	}

	for {
		r := next()
		if r == '\n' || r < 0 {
			return // error: string literal not terminated
		}
		if r == '"' {
			return offset, true // success
		}
		if r == '\\' {
			var skip int
			switch next() {
			case 'a', 'b', 'f', 'n', 'r', 't', 'v', '\\', '"':
				skip = 0
			case '0', '1', '2', '3', '4', '5', '6', '7':
				skip = 2
			case 'x':
				skip = 2
			case 'u':
				skip = 4
			case 'U':
				skip = 8
			default:
				return // error: invalid escape
			}

			for i := 0; i < skip; i++ {
				next()
			}
		}
	}
}

func help() {
	fmt.Println(`
COMMANDS      ARGS   
	node             print a set of all nodes
	deg              print in/out degree of nodes
	next      n      print n's immediate sucessors
	prev      n      print n's immediate predecessors
	front     n      print nodes transitively reachable from n 
	back      n      print nodes that transitively reach n 
	walk      p   s  walk p to sucessor s via random path 
	walkall   p   s  walk all possible paths leading to s
	beatit           exit
	strong    n      print nodes strongly connected to n
	strongs   n      print all strongly connected nodes
`)
}

func usage() {
	fmt.Println(`
NAME
	digra - Directed graph

SYNOPSIS
	digra 

DESCRIPTION
	node             print a set of all nodes
	deg              print in/out degree of nodes
	next      n      print n's immediate sucessors
	prev      n      print n's immediate predecessors
	front     n      print nodes transitively reachable from n 
	back      n      print nodes that transitively reach n 
	walk      p   s  walk p to sucessor s via random path 
	walkall   p   s  walk all possible paths leading to s
	beatit           exit
	strong    n      print nodes strongly connected to n
	strongs   n      print all strongly connected nodes

FLAGS

EXAMPLE

BUGS
	
`)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func fatal(err error) {
	if err == nil {
		return
	}
	printerr(err)
	os.Exit(1)
}
