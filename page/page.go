package main

import (
	//	"github.com/as/clip"
	//"golang.org/x/image/font"
	draw2 "golang.org/x/image/draw"
	"github.com/as/frame"
	"bufio"
	"image"
	"image/color"
	"image/draw"
	"log"
	"time"
	"os"

	_ "image/png"
	_ 	"image/jpeg"
	_ 	"image/gif"
	_ 	"golang.org/x/image/bmp"
	_ 	"golang.org/x/image/tiff"

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

var kern = draw2.Interpolator(draw2.NearestNeighbor)

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
	if err != nil{
		return nil, err
	}
	r := img.Bounds()
	if r.Max.X > winSize.X || r.Max.Y > winSize.Y-35{
		img = Scale(img, image.Rectangle{image.ZP, image.Pt(winSize.X, winSize.Y-35)})
	}
	switch img := img.(type){
	case draw.Image:
		fr.Insert([]byte(path+"\n\n"), 0)
		draw.Draw(img, img.Bounds(), bitmap, bitmap.Bounds().Min, draw.Src)
	}
	return img, err
}

var bitmap = image.NewRGBA(image.Rect(0,winSize.Y-35,winSize.X,winSize.Y))
var fr = frame.New(bitmap.Bounds(), frame.NewGoMono(15), bitmap, frame.Acme)

func main() {
	driver.Main(func(src screen.Screen) {
		win, _ := src.NewWindow(&screen.NewWindowOptions{winSize.X, winSize.Y, "thing"})
		focused := false
		focused = focused
		buf, _ := src.NewBuffer(winSize)

		go func() {
			sc := bufio.NewScanner(bufio.NewReader(os.Stdin))
			for sc.Scan() {
				img, err := readimage(sc.Text())
				if err != nil {
					log.Println(err)
					continue
				}
				win.Send(img)
			}
		}()
		for {
			switch e := win.NextEvent().(type) {
			case image.Image:
				draw.Draw(buf.RGBA(), e.Bounds(), e, e.Bounds().Min, draw.Src)
				win.Send(paint.Event{})
			case key.Event:
				//fmt.Printf("%#v\n", e)
			case mouse.Event:
				//drawBorder(buf.RGBA(), image.Rect(0, 0, 4, 4).Add(image.Pt(int(e.X), int(e.Y))), Red, image.ZP, 1)
				//win.Send(paint.Event{})
			case size.Event:
				winSize = e.Size()
				win.SendFirst(paint.Event{})
			case paint.Event:
				win.Upload(buf.Bounds().Min, buf, buf.Bounds())
				win.Publish()
				time.Sleep(time.Second/24)
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
