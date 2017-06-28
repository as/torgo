package main

import (
	"bufio"
	"bytes"
	"io"

	"github.com/as/io/rev"
)

func findlinerev(p []byte, org, N int64) (q0, q1 int64) {
	N = -N + 1
	p0 := p
	p = p[:org]
	q0, q1 = findline2(N, rev.NewReader(p)) // 0 = len(p)-1
	l := q1 - q0
	q0 = org - q1
	q1 = q0 + l
	q0 = q1 - l
	if q0 >= 0 && q0 < int64(len(p0)) && p0[q0] == '\n' {
		q0++
	}
	return
}
func findline3(p []byte, org, N int64) (q0, q1 int64) {
	p = p[org:]
	q0, q1 = findline2(N, bytes.NewReader(p))
	return q0 + org, q1 + org
}

// Put	Edit 354
func findline2(N int64, r io.Reader) (q0, q1 int64) {
	br := bufio.NewReader(r)
	nl := int64(0)
	for nl != N {
		b, err := br.ReadByte()
		if err != nil {
			break
		}
		q1++
		if b == '\n' {
			nl++
			if nl == N {
				break
			}
			q0 = q1
		}
	}
	return
}

func findline(N int64, p []byte) (q0, q1 int64) {
	nl := int64(0)
	l := int64(len(p))
	for ; q1 < l; q1++ {
		if p[q1] != '\n' {
			continue
		}
		nl++
		if nl == N {
			if q0 != 0 {
				q0++
			}
			q1++
			break
		}
		q0 = q1
	}
	if q1 >= l {
		q0++
	}
	return q0, q1
}
