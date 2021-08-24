package count

import "bytes"

type Writer struct {
	s []byte
	N int64
}

func NewWriter(s string) *Writer {
	return &Writer{[]byte(s), 0}
}

func (c *Writer) Write(p []byte) (int, error) {
c.N += int64(bytes.Count(p, c.s))
	return len(p), nil
}

func (c Writer) Seen() int64 {
	return c.N
}
