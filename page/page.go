package main

import (
	//	"github.com/as/clip"
	//"golang.org/x/image/font"
	"bufio"
	"fmt"
	"github.com/as/font"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"

	"github.com/as/cursor"

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

var kern = draw2.Interpolator(draw2.CatmullRom)

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
	if r.Max.X > winSize.X || r.Max.Y > winSize.Y-35 {
		img = Scale(img, image.Rectangle{image.ZP, image.Pt(winSize.X, winSize.Y-35)})
	}
	switch img := img.(type) {
	case draw.Image:
		fr.Insert([]byte(path+"\n\n"), 0)
		draw.Draw(img, img.Bounds(), bitmap, bitmap.Bounds().Min, draw.Src)
	}
	return img, nil
}

var bitmap = image.NewRGBA(image.Rect(0, winSize.Y-35, winSize.X, winSize.Y))
var fr = frame.New(bitmap, bitmap.Bounds(), &frame.Config{Font: font.NewGoMono(15), Color: frame.Acme})

func main() {
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
					img, err := readimage(sc.Text())
					if err != nil {
						log.Println(err)
						continue
					}
					win.Send(img)
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
						win.Send(size0)
						size = size0
					}
					win.Send(Scale(img, image.Rectangle{image.ZP, winSize}))
				}
			}
		}()
		dirty := true
		outp := winSize
		for {
			switch e := win.NextEvent().(type) {
			case image.Point:
				outp = e
			case image.Image:
				//log.Println("image")
				draw.Draw(buf.RGBA(), e.Bounds(), e, e.Bounds().Min, draw.Src)
				dirty = true
				win.Send(paint.Event{})
			case key.Event:
				//log.Println("kbd")
				//fmt.Printf("%#v\n", e)
			case mouse.Event:
				//log.Println("mouse")
				x0, y0 := e.X, e.Y
				//x := int(x0*(float32(winSize.X)/float32(outp.X)))
				//y := int(y0*(float32(winSize.Y)/float32(outp.Y)))
				x := int(x0 * (float32(outp.X) / float32(winSize.X)))
				y := int(y0 * (float32(outp.Y) / float32(winSize.Y)))
				//log.Printf("local: (%d,%d) remote: (%d,%d)\n", int(x0), int(y0), x, y)
				btn := int(e.Button)
				if e.Direction == 0 {
					btn = 1
				}
				if e.Direction == 1 && e.Button == 1 {
					btn = 2
				}
				if e.Direction == 2 && e.Button == 1 {
					btn = 4
				}
				v := cursor.WriteString(x, y, btn, 0)
				log.Println("send", v)
				fmt.Println(v)
				//drawBorder(buf.RGBA(), image.Rect(0, 0, 4, 4).Add(image.Pt(int(e.X), int(e.Y))), Red, image.ZP, 1)
			case size.Event:
				//log.Println("size")
				winSize = e.Size()
				dirty = true
			case paint.Event:
				//log.Println("paint")
				if dirty {
					win.Upload(buf.Bounds().Min, buf, buf.Bounds())
					win.Publish()
					dirty = false
				}
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
