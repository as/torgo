// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
package main

import (
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const prefix = "dx: "

var (
	h1, h2 = flag.Bool("h", false, "help"), flag.Bool("?", false, "help")

	f = flag.Bool("f", false, "Apply regexp only to the diff path")
	v = flag.Bool("v", false, "Reverse. Print items not matching regexp")
	u = flag.Bool("u", false, "Compute untracked files in git, then print the git command that would start tracking them to stdout")
)

func init() {
	log.SetPrefix(prefix)
	flag.Parse()
	if *h1 || *h2 {
		usage()
		os.Exit(0)
	}
}

func untracked() {
	fd, err := os.Open(".git/index")
	ck("open git index", err)
	defer fd.Close()

	dir := &Dir{}
	ck("read index", dir.ReadBinary(fd))

	seen := make(map[string]bool, len(dir.Ent))
	for _, v := range dir.Ent {
		seen[string(v.Path)] = true
	}

	filepath.Walk(".", filepath.WalkFunc(func(path string, info os.FileInfo, err error) error {
		if path == ".git" {
			return filepath.SkipDir
		}
		if !seen[filepath.ToSlash(path)] {
			fmt.Println(path)
		}
		return nil
	}))
	//fmt.Println(dir)
}

/*
func dirty(){
		fd, err := os.Open(".git/index")
		ck("open git index", err)
		defer fd.Close()

		dir := &Dir{}
		ck("read index", dir.ReadBinary(fd))

		type pathhash struct{
			path string
			hash [sha1.Size]byte
		}
		seenhash := make(map[hashpath]bool)
		seen := make(map[string]bool, len(dir.Ent))
		for _, v := range dir.Ent {
			p := string(v.Path)
			seen[p] = true
			seen[pathhash{p, v.Hash}]=true
		}

		filepath.Walk(".", filepath.WalkFunc(func(path string, info os.FileInfo, err error) error {
			if path == ".git"{
				return filepath.SkipDir
			}
			if !seen[filepath.ToSlash(path)] {
				fmt.Println(path)
			}
			return nil
		}))
		//fmt.Println(dir)
}
*/

func main() {
	if *u {
		untracked()
		os.Exit(0)
	}
	_, err := exec.LookPath("xo")
	if err != nil {
		log.Fatal(`xo binary missing; install with "go get github.com/as/torgo/xo"`)
	}

	xoflags := []string{"-x", `/^diff..-git/,/\ndiff..-git/-/........../`}
	if *v {
		xoflags = append(xoflags, "-v")
	}
	regexp := strings.Join(flag.Args(), " ")
	if *f {
		regexp = fmt.Sprintf(`diff..+git.+%s`)
	}
	xoflags = append(xoflags, regexp)

	cmd := exec.Command("xo", xoflags...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	ck("start", cmd.Start())

	err = cmd.Wait()
	if err != nil {
		os.Exit(1)
	}
}

func ck(where string, err error) {
	if err != nil {
		log.Fatalf("%s: %v", where, err)
	}
}

func usage() {
	fmt.Print(`
NAME
	dx - git tool

SYNOPSIS
	git diff | dx [-f] [-v] regexp
	dx -u

DESCRIPTION
	Dx processes git metadata without the git program. The default
	behavior processes a git diff from standard input and matches
	each file in that diff for a regular expression. The -f flag modifies
	the regular expression to the file's name, allowing garbage files
	such as changelogs to be filtered out of the output of a git diff
	command. The -v flag negates the matching logic of regexp
	in a manner similar to grep.
	
	Alternatively, -u lists files git doesn't currently track, and prints
	a set of commands to could be piped into a command processor
	to have git start tracking those files.
	
	Dx does not invoke the git executable or rely on its presense on
	the system to operate.
	
	Options:
	
	-f, Apply regexp only to git diff path (for filtering/including files)
	-v, Reverse. Match items only if they dont match regexp
	-u, List untracked files by git, output command to start tracking them to stdout
	
EXAMPLE
	
	See what changed in your repository without the million changelog files
        git diff | dx -f -v changelog 
`)
}

func (e Ent) String() string {
	return fmt.Sprintf(
		"%s	%d	%d %d	%s	%s	%x	%s\n",
		e.Mode, e.Size, e.Uid, e.Gid, e.MTime, e.Flag, e.Hash, e.Path,
	)
}

type BinaryReader interface {
	ReadBinary(r io.Reader) error
}

func NewDecoder(r io.Reader) *decoder {
	d, ok := r.(*decoder)
	if ok {
		return d
	}
	return &decoder{r, nil, 0}
}

type decoder struct {
	io.Reader
	err error
	n   int
}

func (r *decoder) Decode(v interface{}) bool {
	if r.err != nil {
		return false
	}
	if br, ok := v.(BinaryReader); ok {
		r.err = br.ReadBinary(r)
	} else {
		r.err = binary.Read(r, binary.BigEndian, v)
	}
	return r.err == nil
}
func (r *decoder) Read(p []byte) (n int, err error) {
	if r.err == nil {
		n, err = r.Reader.Read(p)
		r.err = err
		r.n += n
	}
	return n, err
}
func (r *decoder) Err() error {
	return r.err
}

type Dir struct {
	DirHdr
	Ent []Ent
}
type DirHdr struct {
	Sig   [4]byte
	Ver   uint32
	Count uint32
}
type Ent struct {
	EntHdr
	FlagV3 FlagV3
	Chop   Varint
	Path   Cstr
}
type EntHdr struct {
	CTime, MTime Tm
	Dev          uint32
	Inode        uint32
	Mode         os.FileMode
	Uid, Gid     uint32
	Size         uint32
	Hash         [sha1.Size]byte
	Flag         Flag
}

type Tm struct {
	Sec, Nano uint32
}

func (t Tm) String() string {
	return t.Time().Format(time.Stamp)
}
func (t Tm) Time() time.Time {
	return time.Unix(int64(t.Sec), int64(t.Nano))
}

type FlagV3 uint16
type Flag uint16

func (f Flag) Valid() bool { return f>>15&1 != 0 }
func (f Flag) Ext() bool   { return f>>14&1 != 0 }
func (f Flag) Stage() int  { return int(f >> 12 & 3) }
func (f Flag) Size() int   { return int(f & ((1 << 12) - 1)) }
func (f Flag) String() (s string) {
	x := []byte("--")
	if f.Valid() {
		x[0] = 'v'
	}
	if f.Ext() {
		x[1] = 'x'
	}
	return fmt.Sprintf("%s%02x", x, f.Stage())
}

func (d *Dir) ReadBinary(r io.Reader) error {
	dec := NewDecoder(r)

	if !dec.Decode(&d.DirHdr) {
		return fmt.Errorf("dir: %v", dec.Err())
	}
	switch v := d.Ver; v {
	case 2:
		return d.v2(dec)
	case 3:
		return d.v3(dec)
	case 4:
		return d.v4(dec)
	default:
		return fmt.Errorf("dir: bad version: have %d, support [2, 4]", v)
	}
}

func (d *Dir) v2(r *decoder) error {
	d.Ent = make([]Ent, d.Count)
	var tmp [1]byte
	for n := 0; n < int(d.Count); n++ {
		e := &d.Ent[n]
		n := r.n
		r.Decode(&e.EntHdr)
		if !r.Decode(&e.Path) {
			return r.Err()
		}

		for (r.n-n)%8 != 0 {
			r.Read(tmp[:])
		}
		if r.Err() != nil {
			return r.Err()
		}
	}
	return nil
}
func (d *Dir) v3(r *decoder) error {
	d.Ent = make([]Ent, d.Count)
	for n := 0; n < int(d.Count); n++ {
		e := &d.Ent[n]
		r.Decode(&e.EntHdr)
		r.Decode(&e.FlagV3)
		if !r.Decode(&e.Path) {
			return r.Err()
		}
	}
	return nil
}
func (d *Dir) v4(r *decoder) error {
	d.Ent = make([]Ent, d.Count)
	pre := ""
	for n := 0; n < int(d.Count); n++ {
		e := &d.Ent[n]
		r.Decode(&e.EntHdr)
		r.Decode(&e.FlagV3)
		r.Decode(&e.Chop)
		if !r.Decode(&e.Path) {
			return r.Err()
		}

		x := len(pre)
		len := x - int(e.Chop)
		if len < 0 {
			log.Printf("dirV4: entry %d: bad prefix size: len=%d - chop=%d: %d", n, x, e.Chop, len)
			continue
		}
		e.Path = Cstr(pre[:len]) + e.Path
	}
	return nil
}

var (
	MaxVLen64 = 10
)

type Cstr string

func (c *Cstr) WriteBinary(w io.Writer) error {
	_, err := io.WriteString(w, string(*c)+"\x00")
	return err
}

func (c *Cstr) ReadBinary(r io.Reader) (err error) {
	var tmp [1]byte
	v := make([]byte, 0, 64)
	n := 0
	for {
		n, err = r.Read(tmp[:])
		if n > 0 {
			if tmp[0] == 0 {
				break
			}
			v = append(v, tmp[0])
		}
		if err != nil {
			return err
		}
	}
	*c = Cstr(v)
	return err
}

type Varint = V
type V uint64

var ErrOverflow = errors.New("varint: varint overflows a 64-bit integer")

// WriteBinary writes the varint to the underlying writer.
func (v V) WriteBinary(w io.Writer) (err error) {
	for err == nil {
		u := byte(v % 128)
		v /= 128
		if v > 0 {
			u |= 128
		}
		_, err = w.Write([]byte{u})
		if v <= 0 {
			break
		}
	}
	if err != nil || err != io.EOF {
		return err
	}
	return nil
}

// ReadBinary read a varint from the underlying reader. It does not
// read beyond the varint.
func (v *V) ReadBinary(r io.Reader) error {
	var b [1]byte
	m := int64(1)
	for n := 0; n < MaxVLen64; n++ {
		_, err := r.Read(b[:])
		if err != nil && err != io.EOF {
			return err
		}
		*v += V((int64(b[0]&127) * m))
		m *= 128
		if b[0]&128 == 0 {
			return nil
		}
		if err == io.EOF {
			return io.ErrUnexpectedEOF
		}
	}
	return ErrOverflow
}
