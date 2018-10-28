// Copyright 2015 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.
//

package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func init() {
	log.SetPrefix("crawl: ")
}

var absurl = regexp.MustCompile(`^[a-z]+://`)

type ec chan error

var (
	limit = flag.Int("l", -1, "limit num pages processed")
	dir   = flag.String("d", "./", "storage directory (disabled) ")
	ua = flag.String("ua", "Mozilla Firefox Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:53.0) Gecko/20100101 Firefox/53.0.", "user agent string")
)

var (
	Maxhosts               = 3
	in                     = make(chan [][]byte, 1024)
	get0, get1, get2       = make(chan string, 128), make(chan Page, 128), make(ec, 128)
	link0, link1, link2    = make(chan Page, 128), make(chan [][]byte, 128), make(ec, 128)
	store0, store1, store2 = make(chan Page, 128), make(chan [][]byte, 128), make(ec, 128)
)

type Page struct {
	Origin string
	Body   []byte
}

func (e ec) ck(ctx string, err error) (ok bool) {
	if err == nil {
		return true
	}
	select {
	case e <- err:
	}
	return false
}

func (e ec) readclose(dst *[]byte, rc io.ReadCloser) (ok bool) {
	defer rc.Close()
	b, err := ioutil.ReadAll(rc)
	*dst = b
	return get2.ck("readclose", err)
}
func store(done chan bool) {
	tick := mktick()
	defer close(store0)
	for {
		select {
		case <-tick:
			log.Println("FIN: store: waiting")
		case <-done:
			return
		case page, more := <-store0:
			if !more {
			}
			u, _ := url.Parse(page.Origin)
			fp := filepath.Join(*dir, filepath.Join(u.Host, u.Path))
			if !strings.HasPrefix(fp, *dir) {
				store2.ck("store", fmt.Errorf("bad/malicious file name: %q", fp))
				continue
			}
			log.Println("FILE: ", fp)
			//ioutil.WriteFile(fp, page.Body, 0600)
		}
	}

}
func link(done chan bool) {
	list := [][]byte{}
	origin := &url.URL{}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					u, err := url.Parse(a.Val)
					if err != nil {
						break
					}
					v := origin.ResolveReference(u)
					list = append(list, []byte(v.String()))
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	tick := mktick()
	defer close(link1)
	defer log.Println("FIN: link(exit)")

	for {
		select {
		case <-tick:
			log.Println("FIN: link: waiting")
		case <-done:
			return
		case page, more := <-link0:
			if !more {
				println("FIN: link0(consumer): no more data")
				close(link1)
				return
			}
			doc, err := html.Parse(bytes.NewReader(page.Body))
			if !link2.ck("html", err) {
				continue
			}
			list = list[:0]
			origin, err = url.Parse(page.Origin)
			if !link2.ck("url.parse", err) {
				continue
			}
			f(doc)

			all := append([][]byte{}, list...)
		Transmit:
			select {
			case link1 <- all:
			case <-tick:
				println("INF: link: cant link1<-url list (no consumers on link1)")
				goto Transmit
			}
		}
	}
}

type Hasher struct {
	io.Reader
	io.Closer
	hash.Hash
	filter
}

func (h *Hasher) Reset(rc io.ReadCloser) {
	if h.Hash == nil {
		h.Hash = md5.New()
	} else {
		h.Hash.Reset()
	}
	h.Closer = rc
	h.Reader = io.TeeReader(rc, h.Hash)
}
func (h *Hasher) Seen() bool {
	if h.filter == nil {
		h.filter = filter{}
	}
	return h.filter.seen(h.Sum(nil))
}

func get(done chan bool) {
	c := http.Client{
		Timeout: time.Second * 5,
	}
	httpget := func(u string) (*http.Response, error){
		r, err := http.NewRequest("GET", u, nil)
		if err != nil{
			return nil, err
		}
		r.Header.Add("User-Agent", *ua)
		return c.Do(r)
	}
	tick := mktick()
	hasher := &Hasher{}
	defer log.Println("FIN: get(exit)")
	defer close(get1)
	for {
		select {
		case <-tick:
			log.Println("INF: get: waiting")
		case <-done:
			return
		case uri, more := <-get0:
			if !more {
				println("FIN: get: get0: no more data")
				return
			}
			url0, err := url.Parse(uri)
			if !get2.ck("url.parse", err) {
				continue
			}
			if !hostallowed[url0.Host] {
				if len(hostallowed) < Maxhosts {
					hostallowed[url0.Host] = true
					log.Println("adding host:", url0.Host)
				} else {
					get2.ck("url.host", errors.New("url host differs"))
					continue
				}
			}

			b := []byte{}
			resp, err := httpget(uri)
			if !get2.ck("get", err) {
				continue
			}
			hasher.Reset(resp.Body)
			if !get2.readclose(&b, hasher) {
				continue
			}
			if hasher.Seen() {
				get2.ck("hash", errors.New("duplicate content"))
				continue
			}
		Transmit:
			select {
			case get1 <- Page{Origin: uri, Body: b}:
			case <-tick:
				println("INF: get: cant get1<-page (no consumers on get1)")
				goto Transmit
			}
		}
	}
}

func mktick() <-chan time.Time {
	return time.NewTicker(time.Second * 5).C
}

type filter map[string]struct{}

func (f filter) seen(url []byte) bool {
	_, ok := f[string(url)]
	if !ok {
		f[string(url)] = struct{}{}
		return false
	}
	return true
}

var urlfilter = filter{}

var firsturl *url.URL
var hostallowed = map[string]bool{}

func main() {
	flag.Parse()
	*dir = filepath.Clean(*dir)
	done := make(chan bool)
	go get(done)
	go link(done)
	// go store(done)
	go func() {
		tick := mktick()
		defer close(link0)
		defer log.Println("FIN: pipe(exit)")
		for {
			select {
			case <-tick:
				log.Println("INF: waiting")
			case <-done:
				return
			case page, more := <-get1:
				if !more {
					return
				}
				log.Println("pipe", len(page.Body))

			Transmit:
				select {
				case link0 <- page:
				case <-tick:
					println("INF: cant link0<-page (no consumers)")
					goto Transmit
				}
			}
		}
	}()
	n := 1

	if len(flag.Args()) < 1 {
		log.Fatal("usage: crawl url")
	}

	var err error
	first := flag.Args()[0]
	firsturl, err = url.Parse(first)
	if err != nil {
		log.Fatal(err)
	}
	hostallowed[firsturl.Host] = true
	get0 <- first
	urlfilter.seen([]byte(first))
	tick := mktick()

	go func() {
		for url := range in {
			for _, url := range url {
				if url == nil {
					continue
				}
				get0 <- string(url)
			}
		}
	}()

Loop:
	for {
		if n == 0 {
			println("FIN: done?")
			break Loop

		}
		select {
		case <-tick:
			println("INF: scheduler: nothing to do: n =", n)
		case err := <-link2:
			log.Printf("ERR: link: %v", err)
			n--
		case err := <-get2:
			log.Printf("ERR: get: %v", err)
			n--
		case err := <-store2:
			log.Printf("ERR: store: %v", err)
		case url, more := <-link1:
			if !more {
				println("FIN: link1(consumer): no more data")
				break Loop
			}
			*limit--
			n--
			for i, u := range url {
				if !urlfilter.seen(u) {
					n++
					fmt.Println(string(u))
				} else {
					url[i] = nil
				}
			}
			if *limit == 0 {
				break Loop
			}
			in <- url
		}
	}

	close(get0)
	close(done)
}
