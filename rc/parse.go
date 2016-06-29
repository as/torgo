package main

type parser struct {
	cmd chan cmd
}

func parse(items chan item) (*parser, chan cmd) {
	p := &parser{
		items: make(chan item),
	}
	go p.run() // parse
	return p, p.items
}

func (p *parser) run() {
	for state := lexText; state != nil; {
		state = state(l)
	}
	close(l.items)
}
