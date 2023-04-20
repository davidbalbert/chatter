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
	nlines := strings.Count(s, "\n")

	_, height, err := term.GetSize(int(p.w.Fd()))
	if err != nil {
		fmt.Fprint(p.w, s)
		return
	}

	if nlines < height {
		fmt.Fprint(p.w, s)
		return
	}

	lines := strings.Split(s, "\n")

	i := 0
	for {
		end := i + height
		if end > len(lines) {
			end = len(lines)
		}

		fmt.Fprint(p.w, strings.Join(lines[i:end], "\n"))

		if end < len(lines) {
			fmt.Fprint(p.w, "\n--More--")
		}

		i += height

		if i >= len(lines) {
			break
		}

		b := make([]byte, 1)
		_, err := os.Stdin.Read(b)
		if err != nil {
			fmt.Fprint(p.w, s)
			return
		}
		c := b[0]

		fmt.Fprint(p.w, "\r"+strings.Repeat(" ", len("--More--"))+"\r")

		if c == 'q' {
			break
		}

		if i >= len(lines) {
			break
		}
	}
}
