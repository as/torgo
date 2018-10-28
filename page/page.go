package main

import (
	//	"github.com/as/clip"
	//"golang.org/x/image/font"
	"bufio"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"sync/atomic"

	"github.com/as/font"

	"github.com/as/frame"
	draw2 "golang.org/x/image/draw"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
)

var Red = image.NewUniform(color.RGBA{255, 0, 0, 255})
var EggShell = image.NewUniform(color.RGBA{128, 128, 128, 33})
var winSize = image.Pt(1920, 1080)

var kern = draw2.Interpolator(draw2.ApproxBiLinear)

func Scale(img image.Image, r image.Rectangle) draw.Image {
	dst := image.NewRGBA(r)
	kern.Scale(dst, dst.Bounds(), img, img.Bounds(), draw2.Src, nil)
	return dst
}
func readimage(path string) (image.Image, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	img, _, err := image.Decode(fd)
	if err != nil {
		return nil, err
	}
	return process(img, path)
}
func process(img image.Image, path string) (image.Image, error) {

	r := img.Bounds()
	wx := float64(winSize.X)
	wy := float64(winSize.Y)
	min := func(a, b float64) float64 {
		if a < b {
			return a
		}
		return b
	}
	dx := wx / float64(r.Dx())
	dy := wy / float64(r.Dy())
	fact := min(dx, dy)
	r.Max.X = int(float64(r.Max.X) * fact)
	r.Max.Y = int(float64(r.Max.Y) * fact)
	log.Printf("scale %s to %s\n", img.Bounds(), r)
	img = Scale(img, r)
	if false {
		switch img := img.(type) {
		case draw.Image:
			fr.Insert([]byte(path+"\n\n"), 0)
			draw.Draw(img, img.Bounds(), bitmap, bitmap.Bounds().Min, draw.Src)
		}
	}
	return img, nil
}

var bitmap = image.NewRGBA(image.Rect(0, winSize.Y-35, winSize.X, winSize.Y))
var fr = frame.New(bitmap, bitmap.Bounds(), &frame.Config{Face: font.NewGoMono(15), Color: frame.Acme})

func main() {
	var (
		ring [1024 * 1024 * 4]string
		N    = int64(len(ring))
		cur  int64
		hi   int64
	)

	doit := make(chan image.Image, 100)
	driver.Main(func(src screen.Screen) {
		win, _ := src.NewWindow(&screen.NewWindowOptions{winSize.X, winSize.Y, "page"})
		focused := false
		focused = focused
		buf, _ := src.NewBuffer(image.Pt(1920, 1080))
		go func() {
			a := os.Args
			if len(a) > 1 && a[len(a)-1] == "-" {
				sc := bufio.NewScanner(bufio.NewReader(os.Stdin))
				for sc.Scan() {
					i := atomic.AddInt64(&hi, 1)
					ring[i%N] = sc.Text()
				}
			} else {
				fd := bufio.NewReader(os.Stdin)
				size := image.ZP
				for {
					img, _, err := image.Decode(fd)
					if err != nil {
						//log.Println(err)
					}
					if img == nil {
						continue
					}
					size0 := image.Pt(img.Bounds().Dx(), img.Bounds().Dy())
					if size0 != size {
						size = size0
					}
					doit <- (Scale(img, image.Rectangle{image.ZP, winSize}))
				}
			}
		}()
		for {
			switch e := win.NextEvent().(type) {
			case image.Image:
			case key.Event:
				const HIDE = 128
				di := int64(0)
				hi := int64(0)
				i := int64(0)
				switch e.Code {
				case key.CodeSpacebar:
					draw.Draw(buf.RGBA(), buf.Bounds(), image.Black, image.ZP, draw.Src)
					win.SendFirst(paint.Event{})
					continue
				case key.CodeLeftArrow:
					di = -1
				case key.CodeRightArrow:
					di = 1
				case key.CodeUpArrow:
					di = 10
				case key.CodeDownArrow:
					di = -10
				}
				if di == 0 || e.Direction == key.DirRelease {
					continue
				}

				tries := 15
			AGAIN:
				i = atomic.AddInt64(&cur, di)
				if i < 0 {
					i = N - i
				}
				hi = atomic.LoadInt64(&hi)
				if hi == 0 {
					i = 0
				} else {
					i = i % hi % N
				}

				print("read", i, ring[i])
				src, err := readimage(ring[i])
				if err != nil {
					println(err.Error())
					if tries != 0 {
						tries--
						goto AGAIN
					}
					continue
				} else {
					println("ok")
				}

				draw.Draw(buf.RGBA(), buf.Bounds(), src, src.Bounds().Min, draw.Src)
				win.Send(paint.Event{})
			case mouse.Event:
			case size.Event:
				winSize = e.Size()
				buf0, _ := src.NewBuffer(winSize)
				draw.Draw(buf0.RGBA(), buf0.Bounds(), buf.RGBA(), buf.RGBA().Bounds().Min, draw.Src)
				buf.Release()
				buf = buf0
				win.Send(paint.Event{})
			case paint.Event:
				win.Fill(buf.Bounds(), image.Black, draw.Src)
				win.Upload(buf.Bounds().Min, buf, buf.Bounds())
				win.Publish()
			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}
				// NT doesn't repaint the window if another window covers it
				if e.Crosses(lifecycle.StageFocused) == lifecycle.CrossOff {
					focused = false
				} else if e.Crosses(lifecycle.StageFocused) == lifecycle.CrossOn {
					focused = true
				}
			case interface{}:
			}
		}
	})
}

func drawBorder(dst draw.Image, r image.Rectangle, src image.Image, sp image.Point, thick int) {
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+thick), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Max.Y-thick, r.Max.X, r.Max.Y), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Min.X, r.Min.Y, r.Min.X+thick, r.Max.Y), src, sp, draw.Src)
	draw.Draw(dst, image.Rect(r.Max.X-thick, r.Min.Y, r.Max.X, r.Max.Y), src, sp, draw.Src)
}
