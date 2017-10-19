package main

import (
	"compress/zlib"
	"compress/lzw"
	"compress/gzip"
	"compress/bzip2"
	"compress/flate"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/as/mute"
)

type rw struct{
	NewReader func(r io.Reader) io.ReadCloser
	NewWriter func(w io.Writer) io.Writer
}


func main() {
	var (
		in io.Reader
		out io.Writer
	)
	alg := "zlib"
	if len(f.Args()) > 0{
		alg=f.Args()[0]
	}
	if args.d {
		in, out = decompress(alg)
	} else {
		in, out = compress(alg)
	}
	_, err := io.Copy(out, in)
	ck(err)
	flush(in, out)
}

func flush(fd ...interface{}){
	for _,fd := range fd{
		if fd, ok := fd.(io.Closer); ok{
			fd.Close()
		}
	}
}

func compress(alg string) (in io.Reader, out io.Writer){
	var err error
	in, out = os.Stdin, os.Stdout
	switch alg{
	case "bzip2":
		err=fmt.Errorf("bzip2: writer not implemented\n")
	case "gzip":
		out,err =  gzip.NewWriterLevel(out, args.l)
	case "zlib":
		out ,err= zlib.NewWriterLevel(out, args.l)
	case "flate": 
		out, err = flate.NewWriter(out, args.l)
		ck(err)
	case "lzw":
		out= lzw.NewWriter(out,lzw.LSB,8)
	default:
		err=fmt.Errorf("bad algorithm: %q\n", alg)
	}
	ck(err)
	return in, out
}

func decompress(alg string) (in io.Reader, out io.Writer){
	var err error
	in, out = os.Stdin, os.Stdout
	switch alg{
	case "bzip2":
		in =  bzip2.NewReader(in)
	case "gzip":
		in, err =  gzip.NewReader(in)
	case "zlib":
		in,err =  zlib.NewReader(in)
	case "flate": 
		in= flate.NewReader(in)
	case "lzw":
		in= lzw.NewReader(in,lzw.LSB,8)
	default:
		err=fmt.Errorf("bad algorithm: %q\n", alg)
	}
	ck(err)
	return in, out
}

var args struct {
	h, q bool
	d    bool
	z bool
	l int
}
var f *flag.FlagSet

func level(){
	if len(os.Args) == 1{
		return
	}
	for i, v := range os.Args[1:]{
		if len(v) > 1{
			v=v[1:]
		}
		n, err := strconv.Atoi(v)
		if err != nil{
			continue
		}
		if n>=0 && n<= 9{
			args.l=n
			os.Args[i+1]="-z"
		}
	}
}
func init() {
	log.SetFlags(0)
	log.SetPrefix("z: ")
	level()
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.BoolVar(&args.d, "d", false, "")
	f.BoolVar(&args.z, "z", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
}
func ck(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Println(`
NAME
	z - compress or decompress io

SYNOPSIS
	z [-d] [alg]

DESCRIPTION
	Z compresses or decompresses stdin using
	the algorithm, alg, and writes the result to stdout

	Supported algorithms (alg). Default flate.
        flate	DEFLATE, LZ77 + Huffman coding
        gzip	flate with archive header
        zlib	flate with a smaller header than gzip
        lzw	UNIX Compress, GIF/TIFF, Adaptive codebook
        bzip2	Burrows-Wheeler block sorting (-d only)

	Flags
        -d    Decompress
        -n    Let n specify compression level (0-9)

EXAMPLES
    # prints hello
    echo hello | z | z -d

`)
}
