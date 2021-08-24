package pkcs7
// Package pkcs7 implements padding and unpadding
// as per PKCS#7, a.k.a, rfc2315.

/*
	go test -bench .
*/
import (
	"errors"
	"bytes"
	"io"
)

const (
	Max int = 256
)

var (
	ErrShort      = errors.New("short")
	ErrLong       = errors.New("long")
	ErrCRC        = errors.New("pad integrity")
	ErrPadByte    = errors.New("bad pad byte")
	ErrNone error = nil
)

// Pad pads message m with respect to block size bs and
// returns a padded message.
func Pad(m []byte, bs int) ([]byte, error) {
	err := validblocksize(bs)
	if err != nil {
		return nil, err
	}
	ps := bs - len(m) % bs
	b := byte(ps % bs)
	return append(m, bytes.Repeat([]byte{b}, ps)...), nil
}

// Unpad removes padding from m and returns a slice pointing to
// the unpadded region of the message. An error occurs if m is 
// not a multiple of bs.
func Unpad(m []byte, bs int) ([]byte, error) {
	err := validblocksize(bs)
	if err != nil {
		return nil,err
	}

	ms := len(m)
	if ms < bs {
		return nil, ErrShort
	}
	pb := m[ms-1]
	pl := int(pb)

	if pl > bs {
		return nil, ErrPadByte
	}
	if pl == 0 {
		pl = bs
	}

	pi := ms - pl  // pad start index
	for _, ac := range m[pi:] {
		if ac != pb  {
			return nil, ErrCRC
		}	
	}
	return m[:pi], nil
}

func validblocksize(bs int) error {
	if bs < 1 || bs > Max {
		return ErrLong
	}
	return nil
}

// TODO: Not tested

type writer struct {
	u io.Writer
	bs int
}

type reader struct {
	u io.Reader
	bs int
}

func NewWriter(w io.Writer, bs int) *writer {
	return &writer{
		u: w,
		bs: bs,
	}
}

func NewReader(r io.Reader, bs int) *reader {
	return &reader{
		u: r,
		bs: bs,
	}
}

func (r reader) Read(p []byte) (n int, err error){
	n, err = r.u.Read(p)
	if err != nil && n < r.bs {
		return 0, err
	}
	tmp, err := Unpad(p, r.bs)
	if err != nil {
		return
	}
	n = copy(p, tmp)
	return
}

func (w writer) Write(p []byte) (n int, err error){
	tmp, err := Pad(p, w.bs)
	if err != nil {
		return
	}
	n, err = w.u.Write(tmp)
	return n-len(tmp), err
}
