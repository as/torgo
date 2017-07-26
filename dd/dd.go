package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/as/mute"
	"io"
	"log"
	"os"
)

var f *flag.FlagSet

var args struct {
	IF, OF             string
	bs, ibs, obs       int
	seek, iseek, oseek int64
	count              uint64
	trunc              bool
	v                  bool
	h, q               bool
}

func init(){
	log.SetFlags(0)
	log.SetPrefix("dd: ")
}

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.StringVar(&args.IF, "if", "", "")
	f.StringVar(&args.OF, "of", "", "")
	f.Int64Var(&args.seek, "seek", 0, "")
	f.Int64Var(&args.iseek, "iseek", 0, "")
	f.Int64Var(&args.oseek, "oseek", 0, "")
	f.IntVar(&args.bs, "bs", -512, "")
	f.IntVar(&args.ibs, "ibs", 512, "")
	f.IntVar(&args.obs, "obs", 512, "")
	f.Uint64Var(&args.count, "count", 1024*1024*1024*1024*1024, "")
	f.BoolVar(&args.v, "v", false, "")
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		log.Fatalln(err)
	}
	if args.h || args.q{
		usage()
		os.Exit(0)
	}
	if args.bs > 0 {
		args.ibs = args.bs
		args.obs = args.bs
	}
	if args.seek > 0 {
		args.iseek = args.seek
		args.oseek = args.seek
	}
	if len(f.Args()) > 0{
		log.Fatalln("extra arguments: %s", f.Args())
	}
}

func main() {
	var (
		ifd, ofd = os.Stdin, os.Stdout
		err      error
		in, out  int64
		flags    = os.O_RDWR
	)
	if args.trunc {
		flags |= os.O_TRUNC
	}
	if args.IF != "" {
		ifd, err = os.OpenFile(args.IF, flags, 0666)
		if err != nil {
			log.Fatalln(err)
		}
	}
	if args.OF != "" {
		ofd, err = os.OpenFile(args.OF, flags, 0666)
		if os.IsNotExist(err) {
			ofd, err = os.Create(args.OF)
		}
		if err != nil {
			log.Fatalln(err)
		}
	}
	ib := make([]byte, args.ibs)
	ibuf := bytes.NewReader(ib)
	obuf := make([]byte, args.obs)
	if args.IF != "" {
		ifd.Seek(args.iseek, 0)
	}
	if args.OF != "" {
		ofd.Seek(args.oseek, 0)
	}
	for nb := uint64(0); nb < args.count; nb++ {
		n, err := io.ReadAtLeast(ifd, ib, len(ib))
		if err == io.ErrUnexpectedEOF {
			err = io.EOF
		}
		if err != nil && err != io.EOF {
			log.Fatalf("iread: nb=%d err=%s\n", nb, err)
		}
		if err != io.EOF && n != len(ib) {
			log.Fatalf("iread: nb=%d err=shortread %d/%d", nb, n, len(ib))
		}
		in += int64(n)
		ibuf.Reset(ib[:n])
		n2, err2 := io.CopyBuffer(ofd, ibuf, obuf)
		if err2 != nil && err2 != io.EOF {
			log.Fatalf("owrite: nb=%d err=%s\n", nb, err2)
		}
		out += int64(n2)
		if err == io.EOF {
			break
		}
	}
	if args.v {
		fmt.Fprintf(os.Stderr, "%d+%d records in\n", in/int64(args.ibs), in%int64(args.ibs))
		fmt.Fprintf(os.Stderr, "%d+%d records out\n", out/int64(args.obs), out%int64(args.obs))
	}
}

func usage() {
	fmt.Println(`
NAME
	dd - copy a file 

SYNOPSIS
	dd [-if file] [-of file] [-bs n] [-count n] [-seek n] [-v]

DESCRIPTION
	dd copies a file with the given block size, count and seek parameters.

	There are a number of options. Defaults are displayed in (parenthesis).

	-if       Input file (stdin)
	-of       Output file (stdout)
	-bs n     copy in n-byte blocks (512)
	-ibs n    read in n-byte blocks (512)
	-seek n   seek forth N blocks (0)
	-iseek n  seek forth N blocks in input file (0)
	-oseek n  seek forth N blocks in output file (0)
	-count n  copy n input blocks before stopping
	-trunc    crush data in existing output file out of existence
	-v        verbose
	
	The seek options are applied before the first io operation
	takes place. On NT, Dd can access low level devices inaccessible
	to other programs.

EXAMPLE
	Back up your boot sector/partition table, from a Windows system.

	dd -bs 512 -if \\.\physicaldrive0 -of backup.dd

	Device files require Admin/elevated execution context.

BUGS
	No ctrl+z for when -if and -of are transposed in the above
	example.

	On some systems, stdin and stdout are not seekable.
`)
}
