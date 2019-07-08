package main

import (
	"bufio"
	"flag"
	"image"
	"image/color/palette"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"time"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
)

const (
	NWhite = iota
	NBlack
)

var (
	dx    = flag.Int("dx", 0, "image width from (0,0)")
	dy    = flag.Int("dy", 0, "image height from (0,0)")
	delay = flag.Duration("delay", time.Second/100, "delay between frames")
)

func init() {
	flag.Parse()
}

var kern = draw.CatmullRom

type Conv chan *image.Paletted

var fr *image.Paletted
var ctr = 1

func Convert(img image.Image) Conv {
	kern = nil

	ch := make(Conv, 1)
	func(ch Conv) {
		r := img.Bounds()
		if *dx != 0 {
			r.Max.X = *dx
		}
		if *dy != 0 {
			r.Max.Y = *dy
		}
		fr = image.NewPaletted(r, palette.WebSafe)
		if kern != nil {
			kern.Scale(fr, r, img, img.Bounds(), draw.Src, nil)
		} else {
			draw.Draw(fr, fr.Bounds(), img, fr.Bounds().Min, draw.Src)
			//frame.New(fr, image.Rect(0, 0, 100, 100), nil).Insert([]byte(fmt.Sprintf("%d", ctr)), 0)
			ctr++
		}
		ch <- fr
	}(ch)
	return ch
}

func main() {
	anim := &gif.GIF{}

	inc := make(chan image.Image)
	if len(flag.Args()) != 0 {
		go func() {
			sc := bufio.NewScanner(os.Stdin)
			for sc.Scan() {
				n := sc.Text()
				func() {
					fd, err := os.Open(n)
					if err != nil {
						log.Println(err)
						return
					}
					defer fd.Close()
					img, _, err := image.Decode(fd)
					if err != nil {
						log.Println(err)
						return
					}
					inc <- img
				}()
			}
			close(inc)
		}()
	} else {
		go func() {
			defer close(inc)
			dec := bufio.NewReaderSize(os.Stdin, 1024*1024)
			for {
				img, kind, err := image.Decode(dec)
				if err != nil {
					log.Printf("decode: %s: %v\n", kind, err)
					break
				}
				inc <- img
			}
		}()
	}

	line := make([]Conv, 0, 1024)
	for img := range inc {
		log.Printf("image bounds are %s", img.Bounds())
		line = append(line, Convert(img))
	}
	for i, v := range line {
		log.Printf("%d/%d\n", i, len(line))
		anim.Delay = append(anim.Delay, int(*delay/(time.Second*100)))
		anim.Image = append(anim.Image, <-v)
		//anim.Disposal = append(anim.Disposal, gif.DisposalBackground)
	}
	if err := gif.EncodeAll(os.Stdout, anim); err != nil {
		log.Fatal(err)
	}
}
