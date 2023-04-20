package main

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/term"
)

type pager struct {
	strings.Builder
	w writeFder
	r io.Reader
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

		if end > i {
			fmt.Fprintf(p.w, "%s", strings.Join(lines[i:end], "\n"))
		}

		i += n

		if i >= len(lines) {
			break
		}

		fmt.Fprintf(p.w, "\n")

	ReadChar:
		for {
			fmt.Fprintf(p.w, "--More--")

			b := make([]byte, 1)
			_, err := p.r.Read(b)
			if err != nil {
				fmt.Fprint(p.w, s)
				return
			}

			fmt.Fprintf(p.w, "%s", "\r"+strings.Repeat(" ", len("--More--"))+"\r")

			switch b[0] {
			case 'q':
				return
			case ' ':
				n = height
				break ReadChar
			case '\r', 'j':
				n = 1
				break ReadChar
			}

			// invalid character, ring bell
			fmt.Fprintf(p.w, "\a")
		}
	}
}
