// loop is a temporary program for Windows systems that executes a looping command
// this is because (for /l %a in () do (somecommand.exe)) | filter.exe crashes command prompt
package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
)

var (
	checkStatus = flag.Bool("ck", false, "check exit status and terminate loop if non-zero")
)

func init() {
	log.SetFlags(0)
	log.SetPrefix("loop:")
	flag.Parse()
}

func main() {
	a := flag.Args()
	if len(a) < 1 {
		log.Println("usage: loop cmd [args...]")
	}
	for {
		out, err := exec.Command(a[0], a[1:]...).Output()
		fmt.Print(string(out))
		if err != nil {
			e, _ := err.(*exec.ExitError)
			if e != nil && !*checkStatus {
				continue
			}
			log.Println(e)
			break
		}
	}
}
