package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type Unit int64

func (u Unit) String() string {
	const (
		G  = 1 << 30
		M  = 1 << 20
		K  = 1 << 10
		B  = 1 
		fm = "%.3f %s"
	)
	switch {
	case u >= G:
		return fmt.Sprintf(fm, float64(u)/G, "G")
	case u >= M:
		return fmt.Sprintf(fm, float64(u)/M, "M")
	case u >= K:
		return fmt.Sprintf(fm, float64(u)/K, "K")
	default:
		return fmt.Sprintf(fm, float64(u)/B, "B")
	}
}

func init() {
	log.SetFlags(0)
	if len(os.Args) > 1 {
		log.SetPrefix(os.Args[1] + ": ")
	} else {
		log.SetPrefix("pv: ")
	}
}

func main() {
	var (
		back       [1024 * 1024]byte
		sec, total int64
	)
	done := make(chan bool)
	nchan := make(chan int64, 1)
	pipefd := io.TeeReader(bufio.NewReader(os.Stdin), os.Stdout)
	tick := time.NewTicker(time.Second * 1)

	go func() {
		// Copy data from the pipe to back buffer and send the
		// write size through nchan.
		for {
			n, err := pipefd.Read(back[:])
			nchan <- int64(n)
			if err != nil {
				if err != io.EOF {
					log.Println(err)
				}
				done <- true
				return
			}
		}
	}()

	start := Start(time.Now())
	for {
		select {
		case t := <-tick.C:
			ts := start.Fmt(t)
			log.Printf("t=%d   b=%s   Δ=%s/s  a=%s/s\n", ts, Unit(total), Unit(sec), Unit(total/int64(ts+1)))
			sec = 0
		case n := <-nchan:
			sec += n
			total += n
		case <-done:
			ts := start.Fmt(time.Now())
			log.Printf("t=%d   b=%s   Δ=%s/s  a=%s/s\n", ts, Unit(total), Unit(sec), Unit(total/int64(ts+1)))
			close(nchan)
			os.Exit(0)
		}
	}
}

type Start time.Time

func (s Start) Fmt(t time.Time) int {
	return int(time.Since(time.Time(s)).Seconds())
}
