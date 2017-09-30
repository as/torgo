// Copyright 2015 "as". All rights reserved. Go license.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
)
import (
	"github.com/as/mute"
)

const (
	Prefix = "time: "
)

var args struct {
	h, q bool
	r    bool
	k    string
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.StringVar(&args.k, "k", "", "")
	f.BoolVar(&args.r, "r", false, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

const Fail = 0x5050DAF7

func main() {
	if args.h || args.q {
		usage()
		os.Exit(0)
	}
	a := f.Args() // Remaining non-flag args
	if len(a) == 0 {
		duddyout(nil)
	}

	cmd := exec.Command(a[0], a[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	t1 := time.Now()
	err := cmd.Start()
	if err != nil {
		duddyout(err)
	}

	err = cmd.Wait()
	t2 := time.Now()

	if err != nil { // TODO: fix assumption
		printerr(err)
		os.Exit(0)
	}

	stat := cmd.ProcessState

	fmt.Fprintf(os.Stderr, "user=%v sys=%v real=%v pid=%v\n",
		stat.UserTime(),
		stat.SystemTime(),
		t2.Sub(t1),
		stat.Pid(),
	)
}

func duddyout(err error) {
	if err != nil {
		printerr(err)
	}
	os.Exit(Fail)
}

func println(v ...interface{}) {
	fmt.Print(Prefix)
	fmt.Println(v...)
}

func printerr(v ...interface{}) {
	fmt.Fprint(os.Stderr, Prefix)
	fmt.Fprintln(os.Stderr, v...)
}

func usage() {
	fmt.Println(`
NAME

SYNOPSIS

DESCRIPTION

BUGS

`)
}
