package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/as/mute"
)

const Prefix = "hget: "

var args struct {
	h, q bool
	v    bool
	a    bool
	u    string
	f    bool
}
var f *flag.FlagSet

type Page struct {
	URL  string
	Body []byte
	err  error
}

func (p *Page) Filename() string {
	u, err := url.Parse(p.URL)
	if err != nil {
		p.err = err
		return ""
	}
	_, file := filepath.Split(u.Path)
	return filepath.Clean(file)
}

func main() {
	out := io.Writer(os.Stdout)
	if args.u == "" {
		args.u = "hget"
	}
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
	if len(f.Args()) == 0 {
		os.Exit(1)
	}
	arg := f.Args()[0]
	if arg == "-" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		cwd, err = filepath.Abs(cwd)
		if err != nil {
			log.Fatal(err)
		}
		getc := make(chan *Page)
		putc := make(chan *Page)
		finc := make(chan *Page)
		for i := 0; i < 4; i++ {
			c := client{
				ua: args.u,
				Client: &http.Client{
					Timeout: time.Second * 5,
				},
				in:  getc,
				out: putc,
				err: finc,
			}
			s := store{
				base: cwd,
				in:   putc,
				err:  finc,
			}
			go c.run()
			go s.run()
		}
		sc := bufio.NewScanner(os.Stdin)
		n := 0

		for sc.Scan() {
			select {
			case getc <- &Page{URL: sc.Text()}:
				n++
			case p := <-finc:
				println(p.URL)
				n--
				if p.err != nil {
					log.Println(p.err)
				}
			}

		}
		for n > 0 {
			<-finc
		}
		close(getc)
	} else {
		doget(out, arg)
	}
}

func ok(ctx string, err error) bool {
	if err != nil {
		log.Printf("%s: %v", ctx, err)
		return false
	}
	return true
}

type client struct {
	*http.Client
	ua  string
	in  <-chan *Page
	out chan<- *Page
	err chan<- *Page
}

func (c *client) ok(p *Page, err error) bool {
	if err != nil {
		p.err = err
		c.err <- p
		return false
	}
	return true
}

func (c *client) get(p *Page) bool {
	rq, err := http.NewRequest("GET", p.URL, nil)
	if !c.ok(p, err) {
		return false
	}
	rq.Header.Set("User-Agent", c.ua)
	resp, err := c.Do(rq)
	if !c.ok(p, err) {
		return false
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	p.Body = b
	return c.ok(p, err)

}

func (c *client) run() {
	defer close(c.out)
	for {
		select {
		case p, more := <-c.in:
			if !more {
				return
			}
			if c.get(p) {
				c.out <- p
			}
		}
	}
}

type store struct {
	base string
	in   <-chan *Page
	err  chan<- *Page
}

func (s *store) Safepath(name string) (string, error) {
	name = filepath.Clean(filepath.Join(s.base, name))
	if !strings.HasPrefix(name, s.base) {
		return "", fmt.Errorf("bad file path: %q: missing base: %q", name, s.base)
	}
	if _, err := os.Stat(name); err == nil {
		return "", fmt.Errorf("existing file: %q", name)
	}
	return name, nil
}

func (s *store) run() {
	for {
		select {
		case p, more := <-s.in:
			if !more {
				return
			}
			file := ""
			file, p.err = s.Safepath(p.Filename())
			if p.err == nil {
				println("write file", file)
				p.err = ioutil.WriteFile(file, p.Body, 0600)
			}
			s.err <- p
		}

	}
}

func doget(out io.Writer, ur string) {
	if args.a {
		u, err := url.Parse(ur)
		_, file := path.Split(u.Path)
		if args.f {
			_, err := os.Stat(file)
			if err != nil {
				log.Println("skip file", file)
			}
			return
		}
		fd, err := os.Create(file)
		no(err)
		defer fd.Close()
		out = fd
	}
	req, err := http.NewRequest("GET", ur, nil)
	no(err)
	req.Header.Set("User-Agent", args.u)
	resp, err := http.DefaultClient.Do(req)
	if args.v {
		fmt.Print(Response{resp})
	}
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	if _, err = io.Copy(out, resp.Body); err != nil {
		log.Println(err)
	}
}

func no(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.a, "a", false, "")
	f.StringVar(&args.u, "u", "Mozilla/5.0", "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}
func usage() {
	fmt.Println(`
NAME
	hget - http get url

SYNOPSIS
	hget url

DESCRIPTION
	Hget writes the contents of an http url to stdout

	Options:
	-a    auto name file and save to current directory
	-u    user agent
	-v    verbose, print response headers to stderr

EXAMPLE
	hget http://example.com

`)
}

type Response struct {
	*http.Response
}

func (r Response) String() string {
	if r.Response == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s Status: %s\n%s\n", r.Proto, r.Status, r.Header)
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}
