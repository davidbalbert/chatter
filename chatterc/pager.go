package main

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/term"
)

type pager struct {
	strings.Builder
	wfd writeFder
	r   io.Reader
}

func (p *pager) print() {
	s := p.String()

	_, height, err := term.GetSize(int(p.wfd.Fd()))
	if err != nil {
		fmt.Fprint(p.wfd, s)
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
			fmt.Fprintf(p.wfd, "%s", strings.Join(lines[i:end], "\n"))
		}

		i += n

		if i >= len(lines) {
			break
		}

		fmt.Fprintf(p.wfd, "\n")

	ReadChar:
		for {
			fmt.Fprintf(p.wfd, "--More--")

			b := make([]byte, 1)
			_, err := p.r.Read(b)
			if err != nil {
				fmt.Fprint(p.wfd, s)
				return
			}

			fmt.Fprintf(p.wfd, "%s", "\r"+strings.Repeat(" ", len("--More--"))+"\r")

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
			fmt.Fprintf(p.wfd, "\a")
		}
	}
}
