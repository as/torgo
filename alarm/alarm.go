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
	Prefix = "alarm: "
)

var args struct {
	h, q bool
	t    int
}

var f *flag.FlagSet

func init() {
	f = flag.NewFlagSet("main", flag.ContinueOnError)
	f.BoolVar(&args.h, "h", false, "")
	f.BoolVar(&args.q, "?", false, "")
	f.IntVar(&args.t, "t", 1, "")
	err := mute.Parse(f, os.Args[1:])
	if err != nil {
		printerr(err)
		os.Exit(1)
	}
}

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
	err := cmd.Start()
	if err != nil {
		duddyout(err)
	}
	time.AfterFunc(time.Duration(args.t)*time.Second, func() {
		cmd.Process.Kill()
		os.Exit(1)
	})
	err = cmd.Wait()
	if err != nil { // TODO: fix assumption
		printerr(err)
		os.Exit(1)
	}
}

func duddyout(err error) {
	if err != nil {
		printerr(err)
	}
	os.Exit(1)
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
	alarm - terminate process after a deadline

SYNOPSIS
	alarm [-t sec] cmd args

DESCRIPTION
	-t n, timeout in seconds (default n=1)

BUGS

`)
}
