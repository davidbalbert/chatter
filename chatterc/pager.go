package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type pager struct {
	strings.Builder
	w writeFder
}

func (p *pager) print() {
	s := p.String()

	_, height, err := term.GetSize(int(p.w.Fd()))
	if err != nil {
		fmt.Fprint(p.w, s)
		return
	}

	lines := strings.Split(s, "\n")

	i := 0
	n := height
	for {
		if i >= len(lines) {
			break
		}

		end := i + n
		if end > len(lines) {
			end = len(lines)
		}

		fmt.Fprint(p.w, strings.Join(lines[i:end], "\n"))

		if end < len(lines) {
			fmt.Fprint(p.w, "\n--More--")
		}

		i += n

		if i >= len(lines) {
			break
		}

		b := make([]byte, 1)
		_, err := os.Stdin.Read(b)
		if err != nil {
			fmt.Fprint(p.w, s)
			return
		}

		fmt.Fprint(p.w, "\r"+strings.Repeat(" ", len("--More--"))+"\r")

		switch b[0] {
		case 'q':
			return
		case ' ':
			n = height
		case '\r', 'j':
			n = 1
		}
	}
}
