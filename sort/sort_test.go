package main

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/as/xo"
)

func run(t *testing.T, in, fl, x, k, ex string) {
	r, err := xo.NewReaderString(strings.NewReader(in), fl, x)
	if err != nil {
		t.Logf("fail: xo: %s", err)
	}
	K := &Keys{}
	for {
		b, _, err := r.Structure()
		K.Dots = append(K.Dots, Dot(b))
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Log(err)
			t.Fail()
		}
	}
	K.lessfn = func(i, j int) bool {
		return bytes.Compare(K.Dots[i], K.Dots[j]) == -1
	}
	if k != "" {
		K.o = regexp.MustCompile(k)
		K.lessfn = K.LessO
	}
	sort.Stable(K)
	ac := ""
	for _, v := range K.Dots {
		ac += fmt.Sprintf("%s", v)
	}
	if ac != ex {
		t.Logf("fail: ac != ex: \n%q != \n%q", ac, ex)
		t.Fail()
	}
}

func TestGoFunc(t *testing.T)    { run(t, DPAT, "i", `/func/,/\n\}\n/`, "", ADPT) }
func TestGoFuncKey(t *testing.T) { run(t, DPAT, "i", `/func/,/\n\}\n/`, `ln\(".`, DTPA) }
func TestAlphabet(t *testing.T) {
	run(t, "dfcsjqerptyuiahlgkzxvmbnwo", "i", `/./`, ``, "abcdefghijklmnopqrstuvwxyz")
}

func TestSurname(t *testing.T) {
	in := `John Smith	100	2016/10/23
Gupta Brown 954 2016/09/27
Gorge Xavier 10 2015/04/20
Jessica Smith 1 2012/02/14
`
	ex := `Gupta Brown 954 2016/09/27
John Smith	100	2016/10/23
Jessica Smith 1 2012/02/14
Gorge Xavier 10 2015/04/20
`
	run(t, in, "i", `/.*/,/\n/`, ` [A-Z]+`, ex)
}

func TestYear(t *testing.T) {
	in := `John Smith	100	2016/10/23
Gupta Brown 954 2016/09/27
Gorge Xavier 10 2015/04/20
Jessica Smith 1 2012/02/14
`
	ex := `Jessica Smith 1 2012/02/14
Gorge Xavier 10 2015/04/20
John Smith	100	2016/10/23
Gupta Brown 954 2016/09/27
`
	run(t, in, "i", `/.*/,/\n/`, `..../`, ex)
}

var DPAT = `func D() {
	fmt.Println("Isn't")
}
func P() {
	fmt.Println("This")
}
func A() {
	fmt.Println("Well")
}
func T() {
	fmt.Println("Obvious")
}
`

var ADPT = `func A() {
	fmt.Println("Well")
}
func D() {
	fmt.Println("Isn't")
}
func P() {
	fmt.Println("This")
}
func T() {
	fmt.Println("Obvious")
}
`

var DTPA = `func D() {
	fmt.Println("Isn't")
}
func T() {
	fmt.Println("Obvious")
}
func P() {
	fmt.Println("This")
}
func A() {
	fmt.Println("Well")
}
`
