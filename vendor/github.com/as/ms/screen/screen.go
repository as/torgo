package screen

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"

	"runtime"
	"unsafe"
//	"io/ioutil"
	"golang.org/x/image/bmp"
	"golang.org/x/sys/windows"
)

type Handle windows.Handle

type Bitmap struct {
	Type       int32
	Width      int32
	Height     int32
	WidthBytes int32
	Planes     int16
	BPP        int16
	Bits       uintptr
}
type BitmapInfo struct {
	Size      int32
	Width     int32
	Height    int32
	Planes    int16
	BPP       int16
	Compress  int32
	ImageSize int32
	DxMeter   int32
	DyMeter   int32
	Used      int32
	Important int32
	//RGBA [1]int32
}
type BitmapFileHeader struct {
	Type   int16
	Size   uint32
	_      [4]byte
	DataAt uint32
}

const RGBColors = 0


func Capture(scr int, win int, r image.Rectangle) (img image.Image, err error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	screen := Handle(scr)
	var (
		mem, screenbmp Handle
	)
	x0, y0, x1, y1 := r.Min.X,r.Min.Y,r.Max.X,r.Max.Y
	if screen, err = GetDC(screen); err != nil {
		return nil, fmt.Errorf("GetDC: %s", err)
	}
	if mem, err = CreateCompatibleDC(screen); err != nil || mem == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC: %s", err)
	}
	if screenbmp, err = CreateCompatibleBitmap(screen, x1-x0, y1-y0); err != nil || screenbmp == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap: %s", err)
	}
	if err = SelectObject(mem, screenbmp); err != nil {
		return nil, fmt.Errorf("SelectObject: %s", err)
	}
	if !BitBlt(mem, 0, 0, x1-x0, y1-y0, screen, x0, y0, 0xCC0020) {
		return nil, fmt.Errorf("BitBlt: %s", err)
	}
	bm := Bitmap{}
	if err = GetObject(screenbmp, int(unsafe.Sizeof(bm)), Handle(uintptr(unsafe.Pointer(&bm)))); err != nil {
		return nil, fmt.Errorf("GetObject: %s", err)
	}
	bi := BitmapInfo{
		Width: bm.Width, Height: bm.Height, Planes: 1,
		BPP:      24,
		Compress: RGBColors,
	}
	bi.Size = 40
	bmpsize := int(((bi.Width*int32(bi.BPP) + 31) / 32) * 4 * bi.Height)
	gh, err := GlobalAlloc(0x0042, bmpsize)
	if err != nil{
		return nil, fmt.Errorf("GlobalAlloc: %s", err)
	}
	defer GlobalFree(gh)
	hand, err := GlobalLock(gh)
	if err != nil{
		return nil, fmt.Errorf("GlobalLock: %s", err)
	}
	if err = GetDIBits(screen, screenbmp, uint32(0), uint32(r.Dy()), uintptr(hand), uintptr(unsafe.Pointer(&bi)), RGBColors); err != nil {
		return nil, fmt.Errorf("GetDIBits: %s", err)
	}
	x := (*(*[1<<31 - 1]byte)(unsafe.Pointer(hand)))[:bmpsize]
	GlobalUnlock(gh)
	hdr := BitmapFileHeader{Type: 0x4D42}
	hdr.DataAt = uint32(unsafe.Sizeof(bi)) + uint32(unsafe.Sizeof(hdr)) - 2
	hdr.Size = uint32(uint32(bmpsize) + hdr.DataAt)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, hdr)
	binary.Write(buf, binary.LittleEndian, bi)
	bi.BPP=24
	buf.Write(x)
	//ioutil.WriteFile("test.bmp", buf.Bytes(), 0775)
	return bmp.Decode(buf)
}

//go:generate go run $GOROOT/src/syscall/mksyscall_windows.go -output zscreen.go screen.go

//sys	CreateCompatibleDC(dc Handle) (handle Handle, err error) = Gdi32.CreateCompatibleDC
//sys	SelectObject(handle Handle, in Handle) (err error) = Gdi32.SelectObject
//sys	BitBlt(dst Handle, dx int, dy int, dw int, dh int, src Handle, sx int, sy int, rop uint32 ) (ok bool) = Gdi32.BitBlt
//sys	CreateCompatibleBitmap(dc Handle, w int, h int) (handle Handle, err error) = Gdi32.CreateCompatibleBitmap
//sys	GetDC(fd Handle) (handle Handle, err error) = user32.GetDC
//sys	GetObject(handle Handle, size int, in Handle) (err error) = Gdi32.GetObjectA
//sys	GetDIBits(hdc Handle, hbmp Handle, sp uint32, n uint32, data uintptr, info uintptr, usage uint32) (err error) = Gdi32.GetDIBits

//sys	GlobalAlloc(flag int, size int) (gh syscall.Handle, err error) = kernel32.GlobalAlloc
//sys	GlobalLock(gh syscall.Handle) (h syscall.Handle, err error) = kernel32.GlobalLock
//sys	GlobalUnlock(gh syscall.Handle) (err error) [failretval==syscall.InvalidHandle] = kernel32.GlobalUnlock
//sys	GlobalFree(gh syscall.Handle) (err error) = kernel32.GlobalFree
//sys	GlobalSize(gh syscall.Handle) (size int, err error) = kernel32.GlobalSize
