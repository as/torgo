package main

type Win struct {
	Q0, Q1 int64
	R      []byte
}

func (w *Win) Select(q0, q1 int64) {
	w.Q0, w.Q1 = q0, q1
}

func (w *Win) Insert(s []byte, q0 int64) int64 {
	n := int64(len(s))
	if n == 0 {
		return 0
	}
	nr := int64(len(w.R))
	if q0 > nr {
		q0 = nr
	}
	back := []byte{}
	if q0 < nr-1 {
		back = w.R[q0:]
	}
	if w.R == nil {
		w.R = []byte{}
	}
	w.R = append(w.R[:q0], append(s, back...)...)
	return int64(len(s))
}

func (w *Win) Delete(q0, q1 int64) int64 {
	n := q1 - q0
	if n == 0 {
		return 0
	}

	Nr := int64(len(w.R))
	copy(w.R[q0:], w.R[q1:][:Nr-q1])
	w.R = w.R[:Nr-n]
	return n
}

func (w *Win) Dot() (q0, q1 int64) {
	nr := int64(len(w.R))
	q0 = clamp(w.Q0, 0, nr)
	q1 = clamp(w.Q1, 0, nr)
	return
}

func (w *Win) Dirty() bool {
	return false
}

func (w *Win) Bytes() []byte {
	return w.R
}

func clamp(v, l, h int64) int64 {
	if v < l {
		return l
	}
	if v > h {
		return h
	}
	return v
}
