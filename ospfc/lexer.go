package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

const EOF = 0

var tokenNames map[int]string = map[int]string{
	EOF:      "EOF",
	LITERAL:  "LITERAL",
	VARIABLE: "VARIABLE",
	WS:       "WS",
}

type astType int

const (
	_ astType = iota
	astCommand
	astTokens
	astToken
)

type ast struct {
	_type    astType
	value    string
	children []*ast
}

func newNode(_type astType, value string, children ...*ast) *ast {
	return &ast{_type: _type, value: value, children: children}
}

type lexer struct {
	s            io.RuneScanner
	buf          []rune
	line, col    int
	patterns     map[int]regexp.Regexp
	currentToken cmddefSymType
	result       *ast
	err          error
}

func newLexer(s string) *lexer {
	l := &lexer{
		s: strings.NewReader(s),
	}

	regexps := map[string]int{
		`[A-Z]+`:                 VARIABLE,
		`[a-zA-Z][a-zA-Z0-9_-]*`: LITERAL,
		`\s+`:                    WS,
	}

	l.patterns = make(map[int]regexp.Regexp, len(regexps))
	for r, t := range regexps {
		l.patterns[t] = *regexp.MustCompile("^" + r)
	}

	return l
}

func (l *lexer) Error(s string) {
	l.err = fmt.Errorf("%s", s)
}

func (l *lexer) Lex(lval *cmddefSymType) int {
	for {
		t := l.nextToken(lval)
		if t == EOF {
			return EOF
		}

		if t != WS {
			return t
		}
	}
}

func (l *lexer) nextToken(lval *cmddefSymType) int {
	if len(l.buf) == 0 {
		err := l.readLine()
		if err != nil {
			if err != io.EOF {
				l.err = err
			}
			return EOF
		}
	}

	for t, re := range l.patterns {
		m := re.FindStringSubmatch(string(l.buf))
		if m != nil {
			lval.s = m[0]
			l.consume(len(lval.s))
			fmt.Printf("token %s %q\n", tokenNames[t], lval.s)
			l.currentToken = *lval
			return t
		}
	}

	l.err = fmt.Errorf("unexpected character %q", l.buf[0])
	return EOF
}

func (l *lexer) readLine() error {
	for {
		r, _, err := l.s.ReadRune()
		if err != nil {
			if err == io.EOF && len(l.buf) > 0 {
				return nil
			}
			return err
		}

		l.buf = append(l.buf, r)

		if r == '\n' {
			l.line++
			l.col = 0
			return nil
		} else {
			l.col++
		}
	}
}

func (l *lexer) consume(n int) {
	l.buf = l.buf[n:]
}

func parseCommandDefinition(s string) (*ast, error) {
	cmddefErrorVerbose = true

	l := newLexer(s)

	p := cmddefNewParser()
	p.Parse(l)

	if l.err != nil {
		return nil, l.err
	}

	return l.result, nil
}
