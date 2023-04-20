package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// A simple pager like more(1). Implements io.Writer. If r is a terminal,
// it must be put in raw mode with term.MakeRaw(). If r is not a terminal,
// Pager is a no-op and all output is directly written to w.

type Pager struct {
	fd      int
	w       io.Writer
	r       io.Reader
	buf     bytes.Buffer
	isTerm  bool
	line    int
	stopped bool
}

func NewPager(r io.Reader, w io.Writer) *Pager {
	isTerm := false
	fd := 0

	f, ok := r.(*os.File)
	if ok {
		fd = int(f.Fd())
		isTerm = term.IsTerminal(fd)
	}

	return &Pager{
		fd:     fd,
		w:      w,
		r:      r,
		buf:    bytes.Buffer{},
		isTerm: isTerm,
	}
}

func (p *Pager) Write(b []byte) (n int, err error) {
	if !p.isTerm {
		return p.w.Write(b)
	}

	if p.stopped {
		return 0, io.EOF
	}

	_, height, err := term.GetSize(p.fd)
	if err != nil {
		return p.w.Write(b)
	}

	p.buf.Write(b)

	written := 0
	for {
		if p.line >= height-1 {
		ReadChar:
			for {
				fmt.Fprintf(p.w, "--More--")

				b := make([]byte, 1)
				_, err := p.r.Read(b)
				if err != nil {
					return written, err
				}

				fmt.Fprintf(p.w, "%s", "\r"+strings.Repeat(" ", len("--More--"))+"\r")

				switch b[0] {
				case 'q':
					p.stopped = true
					return written, io.EOF
				case ' ':
					p.line = 0
					break ReadChar
				case '\r', 'j':
					p.line--
					break ReadChar
				case '\x1b':
					// escape sequence, read next two bytes
					b := make([]byte, 2)
					_, err := p.r.Read(b)
					if err != nil {
						return written, err
					}

					if b[0] == '[' && b[1] == 'B' {
						// down arrow
						p.line--
						break ReadChar
					}
				}

				// invalid character, ring bell
				fmt.Fprintf(p.w, "\a")
			}
		}

		line, err := p.buf.ReadString('\n')
		if err != nil && err != io.EOF {
			return written, err
		}

		n, err = fmt.Fprintf(p.w, "%s", line)
		written += n
		if err != nil {
			return written, err
		}

		p.line++

		if written == len(b) {
			return written, nil
		}
	}
}
