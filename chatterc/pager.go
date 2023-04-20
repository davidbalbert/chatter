package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"golang.org/x/term"
)

type pager struct {
	wfd     writeFder
	r       io.Reader
	buf     bytes.Buffer
	isTerm  bool
	line    int
	stopped bool
}

func newPager(wfd writeFder, r io.Reader) *pager {
	return &pager{
		wfd:    wfd,
		r:      r,
		buf:    bytes.Buffer{},
		isTerm: term.IsTerminal(int(wfd.Fd())),
	}
}

func (p *pager) Write(b []byte) (n int, err error) {
	if !p.isTerm {
		return p.wfd.Write(b)
	}

	if p.stopped {
		return 0, io.EOF
	}

	_, height, err := term.GetSize(int(p.wfd.Fd()))
	if err != nil {
		return p.wfd.Write(b)
	}

	p.buf.Write(b)

	written := 0

	for {
		if p.line >= height-1 {
		ReadChar:
			for {
				fmt.Fprintf(p.wfd, "--More--")

				b := make([]byte, 1)
				_, err := p.r.Read(b)
				if err != nil {
					return written, err
				}

				fmt.Fprintf(p.wfd, "%s", "\r"+strings.Repeat(" ", len("--More--"))+"\r")

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
				}

				// invalid character, ring bell
				fmt.Fprintf(p.wfd, "\a")
			}
		}

		line, err := p.buf.ReadString('\n')
		if err != nil && err != io.EOF {
			return written, err
		}

		n, err = fmt.Fprintf(p.wfd, "%s", line)
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

func (p *pager) Close() error {
	if p.stopped {
		return nil
	}

	br := bytes.NewReader(p.buf.Bytes())
	_, err := io.Copy(p.wfd, br)
	return err
}
