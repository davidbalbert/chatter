package main

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"golang.org/x/term"
)

// A simple pager like more(1). Implements io.Writer. If r is a terminal,
// it must be put in raw mode with term.MakeRaw(). If r is not a terminal,
// pager is a no-op and all output is directly written to w.
type pager struct {
	fd         int
	w          io.Writer
	r          *bufio.Reader
	buf        bytes.Buffer
	shouldPage bool
	line       int
	stopped    bool
}

var _ io.Writer = &pager{}

func newPager(r io.Reader, w io.Writer) *pager {
	shouldPage := false
	fd := -1

	f, ok := r.(interface{ Fd() uintptr }) // usually *os.File
	if ok {
		fd = int(f.Fd())
		shouldPage = term.IsTerminal(fd)
	}

	return &pager{
		fd:         fd,
		w:          w,
		r:          bufio.NewReader(r),
		buf:        bytes.Buffer{},
		shouldPage: shouldPage,
	}
}

func (p *pager) Write(b []byte) (n int, err error) {
	if !p.shouldPage {
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
	for written < len(b) {
		if p.line >= height-1 {
			// We're at the bottom of the screen, so wait for user input
			// before continuing. It's height-1 rather than height to
			// leave room for "--More--".
			err := p.paginate()
			if err != nil {
				return written, err
			}

			// We just pressed 'G' to go to the end. Pass through the rest
			// of the buffer.
			if !p.shouldPage {
				n, err := p.w.Write(p.buf.Bytes())
				written += n
				return written, err
			}
		}

		line, err := p.buf.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return written, err
		}
		n, err := p.w.Write(line)
		written += n
		if err != nil {
			return written, err
		}
		p.line++
	}
	return written, nil
}

func (p *pager) paginate() error {
	more := []byte("--More--")
	clear := []byte("\r" + strings.Repeat(" ", len(more)) + "\r")

	for {
		_, err := p.w.Write(more)
		if err != nil {
			return err
		}

		b, err := p.r.ReadByte()
		if err != nil {
			return err
		}

		_, err = p.w.Write(clear)
		if err != nil {
			return err
		}

		switch b {
		case 'q':
			p.stopped = true
			return io.EOF
		case ' ':
			p.line = 0
			return nil
		case '\r', 'j':
			p.line--
			return nil
		case 'G':
			p.shouldPage = false
			return nil
		case '\x1b': // escape sequence, read next two bytes
			var b [2]byte
			_, err := p.r.Read(b[:])
			if err != nil {
				return err
			}
			if b[0] == '[' && b[1] == 'B' {
				// down arrow
				p.line--
				return nil
			}
		}
	}
}
