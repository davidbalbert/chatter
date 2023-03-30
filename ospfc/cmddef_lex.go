package main

import (
	"fmt"
	"io"
	"strings"
	"unicode"
)

const eof = 0

type cmddefLex struct {
	s      io.RuneScanner
	result []string
	err    error
}

func (l *cmddefLex) Error(s string) {
	l.err = fmt.Errorf("%s", s)
}

func (l *cmddefLex) Lex(lval *cmddefSymType) int {
	for {
		r := l.next()

		switch {
		case r == eof:
			return eof
		case unicode.IsLetter(r):
			return l.literal(r, lval)
		case unicode.IsSpace(r):
		default:
			l.err = fmt.Errorf("unexpected character: %c", r)
			return eof
		}
	}
}

func (l *cmddefLex) literal(r rune, lval *cmddefSymType) int {
	var s string

	for {
		s += string(r)
		r = l.peek()

		if r == eof || !unicode.IsLetter(r) {
			break
		}

		l.next()
	}

	lval.s = s
	return LITERAL
}

func (l *cmddefLex) next() rune {
	r, _, err := l.s.ReadRune()
	if err == io.EOF {
		return eof
	} else if err != nil {
		l.err = err
		return eof
	}

	return r
}

func (l *cmddefLex) peek() rune {
	r := l.next()
	if r == eof {
		return eof
	}

	err := l.s.UnreadRune()
	if err != nil {
		l.err = err
		return eof
	}

	return r
}

func parseCommandDefinition(s string) (tokens []string, err error) {
	cmddefErrorVerbose = true

	l := &cmddefLex{
		s: strings.NewReader(s),
	}

	p := cmddefNewParser()
	p.Parse(l)

	if l.err != nil {
		return nil, l.err
	}

	return l.result, nil
}
