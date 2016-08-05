package main

import "io"

type Reverser struct {
	b []byte
	i int
}

func NewReverser(b []byte) *Reverser {
	return &Reverser{
		b: b,
		i: len(b),
	}
}

func (r *Reverser) Read(p []byte) (n int, err error) {
	pl := len(p)
	for {
		r.i--
		if r.i < 0 {
			if n == 0 {
				return 0, io.EOF
			}
			return n, nil
		}
		if n >= pl {
			return n, err
		}
		p[n] = r.b[r.i]
		n++
	}
}
