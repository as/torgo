package main

import (
	"bufio"
	"image"
	. "image/color"
	"image/color/palette"
	"image/gif"
	"log"
	"os"
	_ "image/png"
	_ "image/jpeg"
	"flag"
	"golang.org/x/image/draw"
)


const (
	NWhite = iota
	NBlack
)

var(
	dx = flag.Int("dx", 0, "image width from (0,0)")
	dy = flag.Int("dy", 0, "image height from (0,0)")
)

func init(){
	flag.Parse()
}

var kern = draw.CatmullRom

type Conv chan *image.Paletted
func Convert(img image.Image) Conv{
	ch := make(Conv)
	go func(ch Conv){
		r := img.Bounds()
		if *dx != 0{
			r.Max.X = *dx
		}
		if *dy != 0{
			r.Max.Y = *dy
		}
		fr := image.NewPaletted(r, palette.Plan9)
		if kern != nil{
			kern.Scale(fr, r, img, img.Bounds(), draw.Src, nil)
		} else {
			
			for y := 0; y < r.Max.Y; y++ {
			for x := 0; x < r.Max.X; x++ {
				fr.Set(x, y, img.At(x, y))
			}
			}
		}
		ch <- fr
	}(ch)
	return ch
}

func main() {
	const (
		delay = 100
	)
	anim := &gif.GIF{}

	inc := make(chan image.Image)
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
	
	line := make([]Conv, 0, 1024)
	for img := range inc {
		line = append(line, Convert(img))
	}
	for i, v := range line{
		log.Printf("%d/%d\n", i, len(line))
		anim.Delay = append(anim.Delay, delay)
		anim.Image = append(anim.Image, <-v)
	}
	if err := gif.EncodeAll(os.Stdout, anim); err != nil {
		log.Fatal(err)
	}
}
