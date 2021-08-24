package xo

import (
	"fmt"
	"os"
)

type level int

var verb level

func (v level) Printf(fm string, i ...interface{}) (n int, err error) {
	if verb < 1 {
		return
	}
	return fmt.Fprintf(os.Stderr, fm, i...)
}
func (v level) Println(i ...interface{}) (n int, err error) {
	if verb < 1 {
		return
	}
	return fmt.Fprintln(os.Stderr, i...)
}
func (v level) Print(i ...interface{}) (n int, err error) {
	if verb < 1 {
		return
	}
	return fmt.Fprint(os.Stderr, i...)
}
