package main

import "bytes"

type Counter struct {
	s []byte
	n int64
}

func NewCounter(s string) *Counter {
	return &Counter{[]byte(s), 0}
}

func (c *Counter) Write(p []byte) (int, error) {
	c.n += int64(bytes.Count(p, c.s))
	return len(p), nil
}

func (c Counter) Seen() int64 {
	return c.n
}
