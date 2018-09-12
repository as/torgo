package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

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

func main() {
	out := io.Writer(os.Stdout)
	if args.u == "" {
		args.u = "Mozilla/5.0"
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
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			doget(out, sc.Text())
		}
	} else {
		doget(out, arg)
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
