package main
// This package operates on Microsoft Cab files. It's
// unfinished. The structure is represented with a
// pair of dynamic and fixed size structs.
//
// Note: Work-in progress

import (
	"encoding/binary"
	"compress/flate"
	"fmt"
	"io"
	"os"
	"reflect"
	"time"
	"bufio"
	"strings"
	"bytes"
	"log"
	"io/ioutil"
)

const (
	HasPrev Bitflag = 1 << iota
	HasNext
	HasRes
)

const (
	MaxBloatCab	  = 60000
	MaxBloatDir	  = 255
	MaxBloatBlock = 255
)

const (
	ReadOnly Bitflag = 0x01
	Hidden   = 0x02
	System   = 0x04
	Modified = 0x20
	AutoRun  = 0x40
	Unicode  = 0x80 // Ignored. UTF8 only.
)

type Bitflag uint16

func (bf Bitflag) In(b uint16) bool {
	fmt.Printf("%x\n", b)
	return b & uint16(bf) != 0
}


type Cab struct {
	Head
	Bloat
	Res          []byte //  
	LCab, LDisk  []byte //  Optional
	RCab, RDisk  []byte	//  

	Dirs     []Dir
	Fids     []Fid

	Mz io.ReadCloser
}

type Head struct {
	Sig        [4]byte
	R1          uint32
	Size,   R2  uint32
	Fidpos, R3  uint32
	Ver
	NDirs, NFids uint16
	Flags        uint16
	ID, GID      uint16
}

type Ver struct {
	Min, Maj byte
}

type Bloat struct {
	Cab   uint16  // Cab
	Dir    uint8  // Dirs
	Block  uint8  // and Blocks
}

type Dir struct {
	HDir
	Hole []byte
}

type HDir struct {
	Pos     int32   // distance to block⁰
	NBlocks int16   // no. blocks in Dir
	AlgID   int16   // compression type
}

// Fid identifies a file in a Cab. Fid⁰ starts at Head.Fidpos
// Fid¹-Fidⁿ follow Fid⁰ contiguously; n is the number 
// of expected files in the Cab.
type Fid struct {
	HFid
	Name   []byte
	Blocks []*Block
	Mz MsReader
}
type HFid struct {
	Size, Pos uint32 // File's size and position
	DirPos    uint16 // Dir index (or overloaded bit rot crap)
	Stamp	         // 4 bytes
	Flags uint16
}

type Block struct {
	HBlock
	Hole         []byte // dubious bit goulash, size Blockholes
	ZData        []byte // compressed data
}
type HBlock struct {
	CRC          int32
	Size, GzSize uint16
}

type Stamp struct {
	Date, Time uint16
}

type Reader struct {
	*bufio.Reader
	*Cab
}

type Zipper interface  {
	io.ReadCloser
	flate.Resetter
}
type MsReader struct {
	sig [2]byte // omg so cool!
	br  io.Reader
	zr Zipper
}

var (
	goodzip = [2]byte{0x43, 0x4b}
)

func (m *MsReader) Read(p []byte) (n int, err error) {
	fmt.Println("MsReader.Read: Enter\n")
	defer func(){ fmt.Printf("MsReader.Read: %q\n", p[:n]) }()
	defer m.zr.Reset(m.br, nil)
	if n, err = m.br.Read(m.sig[:]); err != nil {
		return n, err
	}
	if m.sig != goodzip {
		fmt.Printf("not good %x", m.sig)
		fmt.Printf("read %d bytes", n)
	}
	
	// TODO: Bug: wrong n bytes read returned
	return m.zr.Read(p)
}

func NewMsReader(i io.Reader) *MsReader {
	b := bufio.NewReader(i)
	return &MsReader{
		br: b,
		zr: flate.NewReader(b).(Zipper),
	}
}

func main() {
	r := NewReader(os.Stdin)
	chatty := func(do func() error) {
		if e := do(); e != nil {
			fmt.Println("err",e)
		}
	}
	chatty(r.ReadHead)
	Dump(r.Head)
	chatty(r.ReadRes)
		fmt.Println("e chatty(r.ReadRes)")
	Dump(r.Bloat)
		fmt.Println("e chatty(Dump(r.Bloat))")
	fmt.Printf("s %s\n", r.Res)
	
	fmt.Println("Reading Dirs")
	for i := uint16(0); i < r.NDirs; i++ {
		chatty(r.ReadDir)
		fmt.Printf("s %#v\nh %#v", r.Dirs, r.Bloat.Dir)
	}
	
	fmt.Println("Reading Fids")
	for i := uint16(0); i < r.NFids; i++ {
		fmt.Println("Reading Fids: ", i, "/", r.NFids)
		chatty(r.ReadFid)
		Dump(r.Fids[i].HFid)
		fmt.Printf("s %#s %s\n", r.Fids[i].Name, r.Fids[i].Stamp)
	}
	
	fmt.Printf("Cab header: %#v\n", r.Cab)

	fmt.Println("Reading Fid Blocks")
	plaintext := new(bytes.Buffer)
	
	for i, v := range r.Fids {
		name := strings.TrimSpace(strings.Trim(string(v.Name), "\x00"))
		fmt.Printf("fid %d (%s)\n", i, name)
		fmt.Println("fid is in dir #", v.DirPos)
		fmt.Println("fid is at offset #", v.Pos)
		fmt.Println("fid size ", v.Size)
		fmt.Printf("dirinfo: %#v\n", r.Dirs[v.DirPos])
		if v.DirPos >= r.NDirs || v.DirPos < 0{
			log.Fatalln("bad fid dir: cab total/fid: %d / %d\n", r.NDirs, v.DirPos)
		}
		if v.Size < 0{
			log.Println("fid: %s: bad size: %d\n", name, v.Size)
		}		
		block, n, err := decodeblock(plaintext, r)
		fmt.Printf("block is %#v\n", block)
		fmt.Printf("size is %d\n", block.size)
		if n > int64(v.Size){
			// part of the next file is already in the plaintext buffer
			log.Println("block contains multiple files")
		}
		log.Printf("fid %d %q\n", i, name)
		log.Printf("file (%d bytes) content %x \n", plaintext.Len(), plaintext.Bytes() )
		err = ioutil.WriteFile(name, plaintext.Next(int(v.Size)), 0777)
		if err != nil{
			log.Fatalln("writefile:", err)
		}
		
		//if err := r.ReadBlock(&r.Fids[i]); err != nil {
		//	fmt.Println("err", err)
		//}
	}
}

//wire9 block crc[4] size[2] zsize[2] mshdr[2] zdata[size-2]

func decodeblock(dst *bytes.Buffer, r io.Reader) (b *block, n int64, err error){
	b = &block{}
	if err = b.ReadBinary(r); err != nil{
		return b, n, fmt.Errorf("decodeblock: %s", err)
	}
	zr := flate.NewReader(bytes.NewReader(b.Bytes()))
	if n, err = dst.ReadFrom(zr); err != nil{
		log.Fatalln(err)
	}
	return b, n, err
}

func (b *block) Bytes() []byte{
	return b.zdata
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		bufio.NewReader(r),
		&Cab{},
	}
}

func (r *Reader)ReadTo(i interface{}) (err error) {
	return binary.Read(r, binary.LittleEndian, i)
}

func (r *Reader)ReadHead() (err error) {
	err = r.ReadTo(&r.Cab.Head)
	if err != nil {
		return err
	}
	return checkHead(&r.Cab.Head)
}

func (r *Reader)ReadDir() (err error) {
	d := Dir{}
	if err = r.ReadTo(&d.HDir); err != nil {
		return err
	}
	if r.Bloat.Dir != 0 {
		d.Hole = make([]byte, r.Bloat.Dir)
		if err := r.ReadTo(d.Hole); err != nil {
			return err
		}
	}
	r.Dirs = append(r.Dirs, d)
	return err
}

func (r *Reader) ReadFid() (err error) {
	f := Fid{}
	if err = r.ReadTo(&f.HFid); err != nil {
		return err
	}
	if f.Name, err = r.ReadBytes(0); err != nil {
		return err
	}
	r.Fids = append(r.Fids, f)
	return err
}

func (r *Reader) ReadBlock(f *Fid) (err error) {
	b := &Block{}
	if err = r.ReadTo(&b.HBlock); err != nil {
		return err
	}
	if r.Bloat.Block != 0 {
		b.Hole = make([]byte, r.Bloat.Block)
		if err := r.ReadTo(b.Hole); err != nil {
			return err
		}
	}

	//
	//

	f.Mz = *NewMsReader(r)
	buf2 := make([]byte, f.Size)
	Dump(b.HBlock)
	
	for i := 0; i < int(f.Size); i++ {
		_, err := f.Mz.Read(buf2)
		if err != nil {
			return err
		}
	}
	return err
}

func (r *Reader) ReadRes() (err error) {
	var (
		c   = r.Cab
		fl  = c.Flags
	)
	if HasRes.In(fl) {
		if err = r.ReadTo(&c.Bloat); err != nil {
			return err
		}
		c.Res = make([]byte, int(c.Bloat.Cab))
		if _, err = io.ReadFull(r, c.Res); err!=nil {
			return err
		}
	}
	if HasPrev.In(fl) {
		c.LCab,  err = r.ReadBytes(0)
		c.LDisk, err = r.ReadBytes(0)
	}
	if HasNext.In(fl) {
		c.RCab,  err = r.ReadBytes(0)
		c.RDisk, err = r.ReadBytes(0)
	}
	return err
}

func checkBloat(c *Cab) error {
	switch nb := c.Bloat; {
	case uint32(nb.Cab)   > c.Size: fallthrough
	case uint32(nb.Dir)   > c.Size: fallthrough
	case uint32(nb.Block) > c.Size:
		return fmt.Errorf("restab: buffer points beyond cab")
	case nb.Cab   > MaxBloatCab:   fallthrough
	case nb.Dir   > MaxBloatDir:   fallthrough
	case nb.Block > MaxBloatBlock:
		return fmt.Errorf("restab: buffer points beyond brain")
	case HasRes.In(c.Flags) && nb.Cab != 0:
		return fmt.Errorf("incoherent reserved bit and size: %d", nb.Cab)
	}
	return nil
}

func checkHead(h *Head) error {
	s := string(h.Sig[:])
	switch {
	case s != "MSCF":
		return fmt.Errorf("hdr: bad magic: %s", s)
	case h.Fidpos > h.Size:
		return fmt.Errorf("hdr: bad offset: %d ep=%d", h.Fidpos, h.Size)
	}
	return nil
}

func (d Dir) Check() error {
	return nil
}

//
// TODO: Move this
//
func UnDerp(s Stamp) time.Time {
	sd := s.Date
	y := sd>>9 + 1980
	mo := (sd >> 5) & 0x1f
	d := sd & 0x0f

	st := s.Time
	h := (st >> 11) & 0x1f
	mi := (st >> 5) & 0x1f
	se := st * 2
	return time.Date(int(y),
		time.Month(mo),
		int(d),
		int(h),
		int(mi),
		int(se),
		0, time.Local)
}
func Derp(T time.Time) (s Stamp) {
	var d, t int
	d = (T.Year() - 1980) << 9
	d |= int(T.Month()) << 5
	d |= T.Day()
	t = T.Hour() << 11
	t |= T.Minute() << 5
	t |= T.Second() / 2
	return Stamp{uint16(d), uint16(t)}
}

func (s Stamp) String() string {
	return fmt.Sprint(UnDerp(s))
}

type Tab string

var tabs Tab

func (t Tab) Printf(f string, i ...interface{}) {
	fmt.Print(t)
	fmt.Printf(f, i...)
}
func (t Tab) Print(i ...interface{}) {
	t.Printf("%v", i...)
}
func (t Tab) Println(i ...interface{}) {
	t.Print(i)
	fmt.Println()
}
func Dump(s interface{}) {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			x := tabs
			if f.Anonymous {
				tabs = Tab(f.Name) + "." + tabs
			} else {
				tabs = "	" + tabs
				tabs.Printf("%s=", f.Name)
			}
			Dump(v.Field(i).Interface())
			tabs = x
		}
	case reflect.Slice:
		tabs.Printf("slice %V\n", s)
		for i := 0; i < v.Len(); i++ {
			fmt.Printf("elem %V\n", v.Index(i))
		}
		tabs.Println()
	case reflect.Ptr:
		tabs.Println("pointer")
	default:
		fmt.Printf("%v\n", s)
	}
}
