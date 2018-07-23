// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/as/xo"
)

const prefix = "dx: "

var (
	h1, h2 = flag.Bool("h", false, "help"), flag.Bool("?", false, "help")

	nocase = flag.Bool("i", false, "Use case-insensitive matching")
	f      = flag.Bool("f", false, "Apply regexp only to the diff path")
	chunk      = flag.Bool("c", false, "Apply regexp to each chunk in the diff")
	v      = flag.Bool("v", false, "Reverse. Print items not matching regexp")
	u      = flag.Bool("u", false, "List untracked files under the working directory")
	uu     = flag.Bool("uu", false, "List untracked files in the repository")
	a      = flag.Bool("a", false, "Print absolute paths")
	r      = flag.Bool("r", false, "Raw list output")
)

var wd, wderr = os.Getwd()

func grubber() (gitdir, attach string) {
	file := ".git"
	ceil := 256
	up := "."
	if wd != "" {
		ceil = strings.Count(wd, string(filepath.Separator)) + 1
	}

	for ; ceil != 0; ceil-- {
		file = filepath.Join(up, file)
		up = ".."
		info, err := os.Stat(file)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Printf("grubber: %s: %v\n", file, err)
			}
			continue
		}
		if !info.IsDir() {
			log.Printf("grubber: not dir: %q", filepath.Join(wd, file))
			continue
		}
		attach, err := filepath.Abs(filepath.Join(file, ".."))
		ck("grubber: attach", err)
		return file, attach
	}

	log.Fatalf("grubber: dir tree missing git repo: leaf: %s", wd)
	return "", ""
}

func init() {
	log.SetPrefix(prefix)
	log.SetFlags(0)
	flag.Parse()
	if *h1 || *h2 {
		usage()
		os.Exit(0)
	}
}

func mkprinter() func(string) {
	f := func(s string) {
		fmt.Println(s)
	}
	if *a {
		g := f
		f = func(s string) {
			g(filepath.Join(wd, s))
		}
	}
	if !*r {
		g := f
		f = func(s string) {
			g("git add " + s)
		}
	}
	return f
}

func main() {
	if *u || *uu {
		untracked(mkprinter(), *uu)
		os.Exit(0)
	}
	if *chunk{
		chunk1()
	} else {
		xoxo()
	}
}

func untracked(println func(string), all bool) {
	git, attach := grubber()

	gitroot := "."
	prefix, err := filepath.Rel(attach, wd)
	if prefix == "." {
		prefix = ""
	}
	ck("rel", err)

	fd, err := os.Open(filepath.Join(git, "index"))
	ck("index", err)

	defer fd.Close()

	dir := &Dir{}
	ck("read index", dir.ReadBinary(fd))

	known := make(map[string]bool, len(dir.Ent))
	if all {
		gitroot, err = filepath.Rel(wd, attach)
		ck("rel", err)
		for _, v := range dir.Ent {
			known[string(v.Path)] = true
		}
	} else {
		for _, v := range dir.Ent {
			p := string(v.Path)
			known[p] = strings.HasPrefix(p, prefix)
		}
	}

	filepath.Walk(gitroot, filepath.WalkFunc(func(path string, info os.FileInfo, err error) error {
		if filepath.Base(path) == ".git" {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if !known[filepath.ToSlash(filepath.Join(prefix, path))] {
			println(path)
		}
		return nil
	}))
	//fmt.Println(dir)
}

func xoxo() {
	resrc := strings.Join(flag.Args(), " ")
	if *f {
		resrc = fmt.Sprintf(`diff..+git.+%s`)
	}
	flags := "s"
	if *nocase {
		flags += "i"
	}
	re := regexp.MustCompile(fmt.Sprintf("(?%s)%s", flags, resrc))
	linedef := `/diff..-git/,/(diff..-git|$)/-/(..........|\n?)/`
	r, err := xo.NewReaderString(os.Stdin, "", linedef)
	matchfn := r.X

	n, d := 0, 0
	for {
		if _, _, err = r.Structure(); err != nil && err != io.EOF {
			break
		}
		b := matchfn()
		found := re.Match(b)
		d++
		if found {
			n++
		}
		if !(*v && found || !*v && !found) {
			// Augustus De Morgan, forgive me from beyond the grave
			fmt.Print(string(b))
		}

		if r.Err() != nil {
			break
		}
	}
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	if n == 0 && d != 0 {
		os.Exit(1)
	}
}

// TODO(as): this is mostly a copy of xoxo()
func chunk1() {
	resrc := strings.Join(flag.Args(), " ")
	if *f {
		resrc = fmt.Sprintf(`diff..+git.+%s`)
	}
	flags := "s"
	if *nocase {
		flags += "i"
	}
	re := regexp.MustCompile(fmt.Sprintf("(?%s)%s", flags, resrc))
	linedef := `/diff..-git/,/(diff..-git|$)/-/(..........|\n?)/`
	r, err := xo.NewReaderString(os.Stdin, "", linedef)
	matchfn := r.X

	n, d := 0, 0
	for {
		if _, _, err = r.Structure(); err != nil && err != io.EOF {
			break
		}
		b := matchfn()
		found := re.Match(b)
		d++
		if found {
			n++
		}
		if !(*v && found || !*v && !found) {
			// Augustus De Morgan, forgive me from beyond the grave
			l1 := bytes.Index(b, []byte("\n@@"))
			nn := 0
			if l1 != -1 {
				label := string(b[:l1])
				b=b[l1:]
				for{
				c0 := bytes.Index(b, []byte("\n@@"))
				if c0 == -1 {
					break
				}
				h1 := bytes.Index(b[c0:], []byte("@@ "))
				if h1 == -1 {
					break
				}
				h1 += c0
				c1 := bytes.Index(b[h1:], []byte("\n@@"))
				if c1 == -1 {
					c1 = len(b)
				} else{
					c1+=h1
				}

				if chunk := b[:c1]; re.Match(chunk) {
					if nn == 0{
						fmt.Print(label)
					}
					nn++
					fmt.Print(string(chunk))
				}
				
				b = b[c1:]
				}
			}
		}

		if r.Err() != nil {
			break
		}
	}
	if err != nil && err != io.EOF {
		log.Fatal(err)
	}
	if n == 0 && d != 0 {
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
	git diff | dx [-f] [-v] [regexp]
	dx [-u | -uu]

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
	
	-i,	Use case-insensitive matching
	-v,	Reverse. Match items only if they dont match regexp
	
	-f, Apply regexp only to git diff path (for filtering/including files)
	-c, Apply regexp to each chunk in the diff (does not recompute sha1s)
	
	-u, List untracked files under the working directory
	-uu, List untracked files in the repository
	-r,	Raw list output (for -u)
	
EXAMPLE
	
	See what changed in your repository without the million changelog files
        git diff | dx -f -v changelog

	List untracked files in the repository (pipeable output)
		dx -uu
	
	As above, but only under the current working directory
		dx -u
		
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
