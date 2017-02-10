// Copyright 2017 "as". All rights reserved. Torgo is governed
// the same BSD license as the go programming language.

package main

import (
	"fmt"
	"bufio"
	"log"
	"io"
	"os"
)

type parser struct{
	br *bufio.Reader
	tok byte
	n int
	err error
}

func (p *parser) next() (byte, error){
	last := p.tok
	lasterr := p.err
	p.tok, p.err = p.br.ReadByte()
	return last, lasterr
}

func main(){
	parser := &parser{br: bufio.NewReader(os.Stdin)}
	parser.next()
	thresh := 9
	for {
		b, err := parser.next()
		if err != nil {
			if err != io.EOF{
				log.Println(err)
			}
			break
		}
		if b == parser.tok {
			parser.n++
			continue
		}
		if parser.n < thresh{
			for i := 0; i < parser.n+1; i++{
				fmt.Printf("%c",b)
			}
		} else {
			fmt.Printf(`%c{{%d}}`, b, parser.n+1)
			fmt.Println()		
		}
		parser.n = 0
	}
}