package main

import (
	"bufio"
	"bytes"
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
)

const Prefix = "hget: "

var (
	h1, h2   = flag.Bool("h", false, "show help"), flag.Bool("?", false, "show help")
	verb     = flag.Bool("v", false, "verbose")
	auto     = flag.Bool("a", false, "auto save file (no pipe)")
	ua       = flag.String("ua", "hget", "user agent")
	force    = flag.Bool("f", false, "overwrite existing files (if using -a)")
	method   = flag.String("X", "GET", "request method")
	header   = flag.String("H", "", "request headers")
	data     = flag.String("d", "", "request data")
	nofollow = flag.Bool("nofollow", false, "dont follow redirects")
)

type Page struct {
	URL string
	http.Header
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

var (
	getc = make(chan *Page)
	putc = make(chan *Page)
	finc = make(chan *Page)
)

func init() {

}

func main() {
	out := io.Writer(os.Stdout)
	if *ua == "" {
		*ua = "hget"
	}
	if *h1 || *h2 {
		usage()
		os.Exit(0)
	}
	if !*nofollow {
		ckredirect = nil
	}
	args := flag.Args()
	if len(args) == 0 {
		os.Exit(1)
	}
	arg := args[0]
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
				ua: *ua,
				Client: &http.Client{
					Timeout:       time.Second * 5,
					CheckRedirect: ckredirect,
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

func (c *client) do(m string, p *Page, head ...string) bool {
	if *verb {
		log.Printf("do: %s %s", m, p.URL)
	}
	rq, err := http.NewRequest(m, p.URL, nil)
	if !c.ok(p, err) {
		return false
	}
	rq.Header.Set("User-Agent", c.ua)
	var k string
	for i, v := range head[:len(head)-len(head)%2] {
		if i%2 == 0 {
			k = v
		} else {
			rq.Header.Set(k, v)
		}
	}
	if p.Body != nil {
		rq.ContentLength = int64(len(p.Body))
		rq.Body = ioutil.NopCloser(bytes.NewReader(p.Body))
	}

	resp, err := c.Do(rq)
	if !c.ok(p, err) {
		p.Body = nil
		return false
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	p.Body = b
	p.Header = resp.Header
	if *verb {
		log.Printf("reply: %+v", resp)
	}
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
			if c.do("GET", p) {
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
				log.Println("write file", file)
				p.err = ioutil.WriteFile(file, p.Body, 0600)
			}
			s.err <- p
		}

	}
}

var ckredirect = noredirects
var noredirects = func(req *http.Request, via []*http.Request) error {
	log.Println("not following")
	return http.ErrUseLastResponse
}

func doget(out io.Writer, ur string) {
	if *auto {
		u, err := url.Parse(ur)
		_, file := path.Split(u.Path)
		if *force {
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

	getc = make(chan *Page, 10)
	putc = make(chan *Page, 10)
	finc = make(chan *Page, 10)
	c := &client{
		Client: &http.Client{
			CheckRedirect: ckredirect,
			Timeout:       time.Second * 5,
		},
		ua:  *ua,
		in:  getc,
		out: putc,
		err: finc,
	}
	hdr := strings.Split(*header, `,`)
	hdr0 := []string{}
	for _, v := range hdr {
		for _, v := range strings.Split(v, ": ") {
			hdr0 = append(hdr0, strings.TrimSpace(v))
		}
	}
	pg := &Page{URL: ur}
	var err error
	buf := []byte(*data)
	if *verb {
		log.Printf("body: %q\n", *data)
	}
	if *data == "-" {
		buf, err = ioutil.ReadAll(os.Stdin)
		no(err)
	}
	pg.Body = buf
	c.do(*method, pg, hdr0...)
	if *verb {
		log.Println(pg.Header)
	}
	if err != nil {
		log.Println(err)
		return
	}
	if _, err = io.Copy(out, bytes.NewReader(pg.Body)); err != nil {
		log.Println(err)
	}
}

func no(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func init() {
	flag.Parse()
	if *h1 || *h2 {
		usage()
		os.Exit(0)
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
