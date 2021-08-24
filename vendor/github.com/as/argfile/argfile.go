package argfile

import (
	"os"
	"io"
	"bufio"
	"fmt"
	"sync"
)

var (
	MaxFD   = 1024
	ticket = make(chan struct{}, MaxFD)
)

type File struct {
	io.ReadCloser
	Name           string
	closefn        func() error
	wg *sync.WaitGroup
	cap chan bool
}

func (fd *File) Close() {
	select{
	case clean := <-fd.cap:
		if clean{
			fd.closefn()
			fd.wg.Done()
			<- ticket
		}
	default:
	}
}

func emit(to chan *File, args ...string) {
	var wg = new(sync.WaitGroup)
	if len(args) == 0 {
		ticket <- struct{}{}
		wg.Add(1)
		ff := &File{os.Stdin, "/dev/stdin", os.Stdin.Close, wg, make(chan bool, 1)}
		ff.cap <- true // capcability to call close and clean up resources exactly once
		to <- ff
		close(to)
		return
	}

	emitfd := func(n string) {
		ticket <- struct{}{}
		wg.Add(1)
		fd, err := os.Open(n)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			fd.Close()
		} else {
			ff := &File{Name: n, ReadCloser: fd, closefn: fd.Close, wg: wg, cap: make(chan bool, 1)}
			ff.cap <- true // capcability to call close and clean up resources exactly once
			to <- ff
		}
	}

	for _, v := range args {
		if v != "-" {
			emitfd(v)
		} else {
			in := bufio.NewScanner(os.Stdin)
			for in.Scan() {
				emitfd(in.Text())
			}
		}
	}
	wg.Wait()
	close(ticket)
	close(to)
}

func Next(args ...string) (to chan *File) {
	to = make(chan *File)
	go emit(to, args...)
	return to
}
