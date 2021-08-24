package xo

// Xo is an attempt to continue research done by Rob Pike on
// structural regular expressions.

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/as/io/count"
	"github.com/as/io/rev"
)

/*
	TODO:
    xo  -l TODO *.go
*/

type Xo struct {
	list         []*regexp.Regexp
	in           io.Reader
	tee          io.Reader
	buf          *bytes.Buffer
	src          *bufio.Reader
	loc          []int
	Last         [2]int
	dot          [2]int
	adv          int
	cnt          *count.Writer
	lastfn       func(re *regexp.Regexp) (err error)
	Line0, Line1 int64
	err          error // next read returns EOF`
	y            []byte
	x            []byte

	cmd []cmd
}

func (r Xo) Err() error { return r.err }

// NewReaderString returns an Xo initialized to execute
// commands from the contents of string s.
func NewReaderString(in io.Reader, flags string, s string) (*Xo, error) {
	c, err := parseaddr(s)
	if err != nil {
		return nil, err
	}
	return NewReader(in, flags, c...), nil
}

//
// Dot is a coordinate pair that controls the selected bytes in
// the buffer.
//
// There are four operations that operate on dot to advance
// through the input:
//
// Add: +/regexp/ Advance dot[0]
// Com: ,/regexp/ Advance dot[1]
// Sub: -/regexp/ Reverse dot[1]
// Sem: ;/regexp/ Reverse dot[0]
//
// TODO: Substitute loose slices for dot in the package
//
type Dot struct {
	P0, P1 int
	val    []byte
}

// NewReader returns an Xo initialized to execute
// command list c.
func NewReader(in io.Reader, flags string, c ...cmd) *Xo {
	r := &Xo{
		in:  in,
		buf: new(bytes.Buffer),
		cnt: count.NewWriter("\n"),
	}
	if len(c) == 0 {
		return nil
	}
	r.cmd = c
	//verb.Println("c:", r.cmd)

	for _, v := range r.cmd {
		text := fmt.Sprintf("(?%s)%s", flags, v.re)
		r.list = append(r.list, regexp.MustCompile(text))
	}
	r.tee = io.TeeReader(r.in, r.buf)
	r.src = bufio.NewReader(r.tee)
	r.cnt.N++
	return r
}

// Add looks for the matching regexp, advancing dot0 to
// where the match ends. It does not modify dot1....
//
// Bug: The current implementation modifies dot1
//
// TODO: See above
//
func (r *Xo) Add(re *regexp.Regexp) (err error) {
	//r.status("Add")
	r.loc = re.FindReaderIndex(r.src)
	if r.loc == nil {
		return io.EOF
	}
	r.adv = len(r.buf.Bytes())
	r.dot[0] = r.Last[1] + r.loc[0]
	r.dot[1] = r.Last[1] + r.loc[1]
	extra := r.adv - r.dot[1]

	b := r.buf.Bytes()
	n := len(b)
	if n < 1 {
		return nil
	}

	n -= extra
	if n < 1 {
		return nil
	}
	if n > len(b) {
		return io.EOF
	}

	r.Rewind(bytes.NewReader(b[n:]))

	//r.status("Add")
	r.Last[0] = r.dot[0]
	r.Last[1] = r.dot[1]
	return
}

// Com looks for the matching regexp, advancing dot1 to
// where the match ends. It does not modify dot0.
func (r *Xo) Com(re *regexp.Regexp) (err error) {
	r.loc = re.FindReaderIndex(r.src)
	if r.loc == nil {
		return io.EOF
	}
	r.adv = len(r.buf.Bytes())
	if r.adv < 0 {
		r.adv = 0
	}
	extra := r.adv - r.dot[1]
	r.dot[1] = r.Last[1] + r.loc[1]
	b := r.buf.Bytes()
	if extra < 0 {
		extra = 0
	}
	r.Rewind(bytes.NewReader(b[len(b)-extra:]))
	r.Last[0] = r.dot[0]
	r.Last[1] = r.dot[1]
	return
}

// Sub looks backwards for the matching regexp. It moves
// dot1 to where the match begins in reverse. It does not
// modify dot0.
//
// The final effect shinks the selection approaching from
// the right.
func (r *Xo) Sub(re *regexp.Regexp) (err error) {
	b := r.buf.Bytes()
	n := len(b)
	if n < 1 {
		return fmt.Errorf("no buffer to reverse seek")
	}
	if len(b) <= r.Last[1] {
		return io.EOF
	}
	//verb.Printf("will invert: %q\n", string(b[:r.Last[1]]))
	rv := rev.NewReader(b[:r.Last[1]])
	r.loc = re.FindReaderIndex(bufio.NewReader(rv))

	// Convert loc into non-projected coordinates
	//r.status("Sub: 0/3")
	//
	r.dot[1] = r.Last[1] - r.loc[1]

	//r.status("Sub: 1/3")
	if r.dot[1] < 0 {
		r.dot[1] = -r.dot[1]
	}
	//r.status("Sub: 2/3")
	if r.dot[0] > r.dot[1] {
		//verb.Printf("REV will swap dot[0]=%v dot[1]=%v\n", r.dot[1], r.dot[0])
		r.dot[0], r.dot[1] = r.dot[1], r.dot[0]
	}
	//r.status("Sub: 3/3")
	return
}

// Sem looks backwards for the matching regexp, advancing
// dot0 to where the match ends in reverse. It does not
// modify dot1.
func (r *Xo) Sem(re *regexp.Regexp) (err error) {
	b := r.buf.Bytes()
	n := len(b)
	if n < 1 {
		return fmt.Errorf("no buffer to reverse seek")
	}
	//verb.Printf("will invert: %q\n", string(b[:r.Last[1]]))
	rv := rev.NewReader(b[:r.Last[1]])
	r.loc = re.FindReaderIndex(bufio.NewReader(rv))

	// Convert loc into non-projected coordinates
	//r.status("Sem: 0/3")
	r.dot[0] = r.dot[1] - r.loc[1]

	//r.status("Sem: 1/3")
	if r.dot[1] < 0 {
		r.dot[1] = n + r.dot[1]
	}
	//r.status("Sem: 2/3")
	if r.dot[0] > r.dot[1] {
		//verb.Printf("REV will swap dot[0]=%v dot[1]=%v\n", r.dot[1], r.dot[0])
		r.dot[0], r.dot[1] = r.dot[1], r.dot[0]
	}
	//r.status("Sem: 3/3")
	return
}

// LastOp executes the last operation on the input regexp
func (r *Xo) LastOp(re *regexp.Regexp) (err error) {
	if r.lastfn == nil {
		return r.Add(re)
	}
	return r.lastfn(re)
}

// Rewind prepends the input reader to the front of the
// read buffer. The next read returns the contents of b.Bytes()
// concatenated with the existing buffer.
func (r *Xo) Rewind(b *bytes.Reader) {
	r.src = bufio.NewReader(io.MultiReader(b, r.tee))
}

// Structure executes the next operation on the read buffer
// and returns the selection bound by dot.
func (r *Xo) Structure() (out []byte, n int, err error) {
	if r.err != nil {
		return nil, 0, err
	}
	r.Last[0], r.Last[1] = 0, 0
	//verb.Printf("\nstate.prog len=%d %v\n", len(r.cmd), r.cmd)
	if len(r.cmd) == 0 {
		return nil, 0, fmt.Errorf("r.state.prog is nil")
	}
	for i, cmd := range r.cmd {
		//verb.Println("Structure: Loop", i, cmd)
		switch Tok(cmd.op) {
		case TokBeg:
			err = r.LastOp(r.list[i])
		case TokAdd:
			err = r.Add(r.list[i])
		case TokCom:
			err = r.Com(r.list[i])
		case TokSub:
			err = r.Sub(r.list[i])
		case TokSem:
			err = r.Sem(r.list[i])
		case TokEnd:
			debugerr("End: ")
		}
		//verb.Printf("\nStructure r.dot=%v b=%q\n", r.dot, r.buf.Bytes())
		moribound(r.src)
		if err != nil {
			r.err = err
			break
		}
	}
	//verb.Printf("\nStructure r.dot=%v b=%q\n", r.dot, r.buf.Bytes())
	b := r.buf.Bytes()
	if len(b) == 0 {
		// printerr(fmt.Sprint("b == nil || len(b) == 0 "))
	}

	y := r.buf.Next(r.dot[0])
	r.y = append([]byte{}, y...)
	// fmt.Printf("y is %q\n" , string(r.y))
	r.cnt.Write(r.y)

	r.Line0 = r.cnt.Seen()

	r.x = r.buf.Next(r.dot[1] - r.dot[0])
	r.cnt.Write(r.x)
	r.Line1 = r.cnt.Seen()

	if len(b) != 0 {
		r.Rewind(bytes.NewReader(r.buf.Bytes()))
	}
	if err != nil && err != io.EOF {
		return r.x, len(r.x), err
	}
	return r.x, len(r.x), moribound(r.src)
}

// X returns the Dot selected by the last
// call to Structure().
func (r *Xo) X() []byte {
	return r.x
}

// Y returns the negation of X(). Everything
// not matched by X() is returned by Y().
//
// Implementation bug: The current Y() returns
// every byte leading up to X(), but not after.
//
// TODO: See above
func (r *Xo) Y() []byte {
	return r.y
}

func (r Xo) status(label string) {
	//verb.Println()
	//verb.Println()
	//verb.Println("Reader Status:", label)
	//verb.Println("r.loc:", r.loc)
	//verb.Println("r.dot:", r.dot)
	//verb.Println("r.Last:", r.Last)
	//verb.Printf("r.buf (len): %d\n", len(r.buf.Bytes()))
	//verb.Printf("r.buf: %q ...\n", r.buf.Bytes())
	//verb.Println()
	//verb.Println()
}

func moribound(r *bufio.Reader) (err error) {
	defer r.UnreadRune()
	if _, _, err = r.ReadRune(); err != nil {
		//verb.Println("moribound", err)
		return err
	}
	return nil
}
