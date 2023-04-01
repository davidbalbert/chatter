package commands

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

const EOF = 0

type lexer struct {
	scanner      io.RuneScanner
	input        string
	buf          []rune
	line, col    int
	patterns     map[int]regexp.Regexp
	currentToken yySymType
	result       *ast
	err          error
}

func newLexer(s string) *lexer {
	l := &lexer{
		scanner: strings.NewReader(s),
		input:   s,
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
	l.col -= len(l.currentToken.s) // rewind to start of unexpected token
	l.setErr(fmt.Errorf(s))
}

func (l *lexer) setErr(err error) {
	lines := strings.Split(l.input, "\n")
	line := lines[l.line-1]
	marker := strings.Repeat(" ", l.col) + "^"

	l.err = fmt.Errorf("<stdin>:%d:%d: error: %w\n\t%s\n\t%s", l.line, l.col, err, line, marker)
}

func (l *lexer) Lex(lval *yySymType) int {
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

func (l *lexer) nextToken(lval *yySymType) int {
	if len(l.buf) == 0 {
		err := l.readLine()
		if err != nil {
			if err != io.EOF {
				l.setErr(err)
			}
			return EOF
		}
	}

	for t, re := range l.patterns {
		m := re.FindStringSubmatch(string(l.buf))
		if m != nil {
			lval.s = m[0]
			l.consume(len(lval.s))
			l.col += len(lval.s)
			l.currentToken = *lval
			return t
		}
	}

	l.setErr(fmt.Errorf("unexpected character %q", l.buf[0]))
	return EOF
}

func (l *lexer) readLine() error {
	l.line++
	l.col = 0

	for {
		r, _, err := l.scanner.ReadRune()
		if err != nil {
			if err == io.EOF && len(l.buf) > 0 {
				return nil
			}
			return err
		}

		l.buf = append(l.buf, r)

		if r == '\r' {
			return nil
		}
	}
}

func (l *lexer) consume(n int) {
	l.buf = l.buf[n:]
}
