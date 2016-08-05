package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
)

type Xo struct {
	list         []*regexp.Regexp
	in           io.Reader
	buf          *bytes.Buffer
	tee          *bufio.Reader
	src          *bufio.Reader
	loc          []int
	last         [2]int
	dot          [2]int
	adv          int
	cnt          *Counter
	lastfn       func(re *regexp.Regexp) (err error)
	line0, line1 int64
	err          error // next read returns EOF

	cmd []cmd
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

// Add looks for the matching regexp, advancing dot0 to
// where the match ends. It does not modify dot1....
//
// Well, it shouldn't anyway. The current implementation
// thinks its a good idea, but HOW ABOUT YOU REMOVE IT
// INSTEAD OF JUST WRITING A COMMENT ABOUT IT HERE!
//
// TODO: See above
//

func NewReader(in io.Reader, c ...cmd) *Xo {
	r := &Xo{
		in:  bufio.NewReader(in),
		buf: new(bytes.Buffer),
		cnt: NewCounter(NL),
	}
	flags := ""
	if args.i {
		flags += "i"
	}

	if len(c) == 0{
		return nil
	}
	r.cmd = c
	verb.Println("c:", r.cmd)

	for _, v := range r.cmd {
		text := fmt.Sprintf("(?%s)%s", flags, v.re)
		r.list = append(r.list, regexp.MustCompile(text))
	}
	r.tee = bufio.NewReader(io.TeeReader(r.in, r.buf))
	r.src = r.tee
	r.cnt.n++
	return r
}

func (r *Xo) Add(re *regexp.Regexp) (err error) {
	r.status("Add")
	r.loc = re.FindReaderIndex(r.src)
	if r.loc == nil {
		return io.EOF
	}
	r.adv = len(r.buf.Bytes())
	r.dot[0] = r.last[1] + r.loc[0]
	r.dot[1] = r.last[1] + r.loc[1]
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

	r.status("Add")
	r.last[0] = r.dot[0]
	r.last[1] = r.dot[1]
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
	r.dot[1] = r.last[1] + r.loc[1]
	b := r.buf.Bytes()
	if extra < 0 {
		extra = 0
	}
	r.Rewind(bytes.NewReader(b[len(b)-extra:]))
	r.last[0] = r.dot[0]
	r.last[1] = r.dot[1]
	return
}

// Sub looks backwards for the matching regexp. It moves
// dot1 to where the match begins in reverse. It does not
// modify dot0.
//
// The final effect shinks the selection approaching from
// the right.
//
func (r *Xo) Sub(re *regexp.Regexp) (err error) {
	b := r.buf.Bytes()
	n := len(b)
	if n < 1 {
		return fmt.Errorf("no buffer to reverse seek")
	}
	verb.Printf("will invert: %q\n", string(b[:r.last[1]]))
	rev := NewReverser(b[:r.last[1]])
	r.loc = re.FindReaderIndex(bufio.NewReader(rev))

	// Convert loc into non-projected coordinates
	r.status("Sub: 0/3")
	//
	r.dot[1] = r.last[1] - r.loc[1]

	r.status("Sub: 1/3")
	if r.dot[1] < 0 {
		r.dot[1] = -r.dot[1]
	}
	r.status("Sub: 2/3")
	if r.dot[0] > r.dot[1] {
		verb.Printf("REV will swap dot[0]=%v dot[1]=%v\n", r.dot[1], r.dot[0])
		r.dot[0], r.dot[1] = r.dot[1], r.dot[0]
	}
	r.status("Sub: 3/3")
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
	verb.Printf("will invert: %q\n", string(b[:r.last[1]]))
	rev := NewReverser(b[:r.last[1]])
	r.loc = re.FindReaderIndex(bufio.NewReader(rev))

	// Convert loc into non-projected coordinates
	r.status("Sem: 0/3")
	r.dot[0] = r.dot[1] - r.loc[1]

	r.status("Sem: 1/3")
	if r.dot[1] < 0 {
		r.dot[1] = n + r.dot[1]
	}
	r.status("Sem: 2/3")
	if r.dot[0] > r.dot[1] {
		verb.Printf("REV will swap dot[0]=%v dot[1]=%v\n", r.dot[1], r.dot[0])
		r.dot[0], r.dot[1] = r.dot[1], r.dot[0]
	}
	r.status("Sem: 3/3")
	return
}

func (r *Xo) Last(re *regexp.Regexp) (err error) {
	if r.lastfn == nil {
		return r.Add(re)
	}
	return r.lastfn(re)
}

func (r *Xo) Rewind(b *bytes.Reader) {
	r.src = bufio.NewReader(io.MultiReader(b, r.tee))
}

func (r *Xo) Structure() (out []byte, n int, err error) {
	if r.err != nil {
		return nil, 0, err
	}
	r.last[0], r.last[1] = 0, 0
	verb.Printf("\nstate.prog len=%d %v\n", len(r.cmd), r.cmd)
	if len(r.cmd) == 0{
		return nil, 0, fmt.Errorf("r.state.prog is nil")
	}
	for i, cmd := range r.cmd {
		verb.Println("Structure: Loop", i, cmd)
		switch Tok(cmd.op) {
		case TokBeg:
			err = r.Last(r.list[i])
		case TokAdd:
			err = r.Add(r.list[i])
		case TokCom:
			err = r.Com(r.list[i])
		case TokSub:
			err = r.Sub(r.list[i])
		case TokSem:
			err = r.Sem(r.list[i])
		case TokEnd:
			printerr("End: ")
		}
		verb.Printf("\nStructure r.dot=%v b=%q\n", r.dot, r.buf.Bytes())
		moribound(r.src)
		if err != nil {
			r.err = err
			break
		}
	}
	verb.Printf("\nStructure r.dot=%v b=%q\n", r.dot, r.buf.Bytes())
	b := r.buf.Bytes()
	if len(b) == 0 {
		// printerr(fmt.Sprint("b == nil || len(b) == 0 "))
	}

	p := r.buf.Next(r.dot[0])
	r.cnt.Write(p)
	r.line0 = r.cnt.Seen()

	s := r.buf.Next(r.dot[1] - r.dot[0])
	r.cnt.Write(s)
	r.line1 = r.cnt.Seen()

	if len(b) != 0 {
		r.Rewind(bytes.NewReader(r.buf.Bytes()))
	}
	if err != nil && err != io.EOF {
		return s, len(s), err
	}
	return s, len(s), moribound(r.src)
}

func (r Xo) status(label string) {
	verb.Println()
	verb.Println()
	verb.Println("Reader Status:", label)
	verb.Println("r.loc:", r.loc)
	verb.Println("r.dot:", r.dot)
	verb.Println("r.last:", r.last)
	verb.Printf("r.buf (len): %d\n", len(r.buf.Bytes()))
	verb.Printf("r.buf: %q ...\n", r.buf.Bytes())
	verb.Println()
	verb.Println()
}
