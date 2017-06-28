package main

type File interface {
	Insert(p []byte, at int64) (wrote int64)
	Delete(q0, q1 int64)(del int64)
	Select(q0, q1 int64)
	Dot() (q0, q1 int64)
	Dirty() bool
	Bytes() []byte
}
