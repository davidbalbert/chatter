package commands

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// An example spec:
// fork[
//	  	literal:foo?"This is a description"[
//			join.1
//		],
//		literal:bar[
//			join.1
//		]
// ]
//
// Whitespace between children is not significant. A rough grammar is:
//
// spec <- ws? name id? description? children? ws?
// name <- "fork" / "join" / "literal:" word / "argument:" argtype
// id <- '.' [1-9][0-9]*
// description <- '?' '"' [^"]* '"'
// children <- '[' spec (',' spec)* ']'
// children <- '[' spec (',' spec)* ','? ws? ']'
// word <- [a-zA-Z0-9]+
// argtype <- "string" / "ipv4" / "ipv6"
// ws <- [ \t\r\n]*

type spec struct {
	typeName    string
	arg         string
	id          int
	description string
	children    []*spec
}

type specParser struct {
	s    string
	pos  int
	line int
	col  int
}

func (p *specParser) parse() (*spec, error) {
	s := &spec{}

	p.skipWhitespace()

	if err := p.parseName(s); err != nil {
		return nil, err
	}

	if p.peek() == '.' {
		if err := p.parseID(s); err != nil {
			return nil, err
		}
	}

	if p.peek() == '?' {
		if err := p.parseDescription(s); err != nil {
			return nil, err
		}
	}

	if p.peek() == '[' {
		if err := p.parseChildren(s); err != nil {
			return nil, err
		}
	}

	p.skipWhitespace()

	return s, nil
}

func (p *specParser) parseName(s *spec) error {
	if p.peek() == 'f' {
		if !p.consume("fork") {
			return p.errorf("expected 'fork'")
		}

		s.typeName = "fork"
		return nil
	}

	if p.peek() == 'j' {
		if !p.consume("join") {
			return p.errorf("expected 'join'")
		}

		s.typeName = "join"
		return nil
	}

	if p.peek() == 'l' {
		if !p.consume("literal:") {
			return p.errorf("expected 'literal:'")
		}

		s.typeName = "literal"
		if err := p.parseWord(s); err != nil {
			return err
		}

		return nil
	}

	if p.peek() == 'a' {
		if !p.consume("argument:") {
			return p.errorf("expected 'argument:'")
		}

		s.typeName = "argument"
		if err := p.parseArgType(s); err != nil {
			return err
		}

		return nil
	}

	return p.errorf("expected 'fork', 'join', 'literal:', or 'argument:'")
}

func (p *specParser) parseID(s *spec) error {
	p.next() // consume the '.'

	var num string
	if p.peek() == '0' {
		return p.errorf("invalid id 0")
	} else if p.peek() < '1' || p.peek() > '9' {
		return p.errorf("expected digit")
	}

	for p.peek() >= '0' && p.peek() <= '9' {
		num += string(p.next())
	}

	id, err := strconv.Atoi(num)
	if err != nil {
		return p.errorf("invalid id %s", num)
	}

	s.id = id

	return nil
}

func (p *specParser) parseDescription(s *spec) error {
	p.next() // consume the '?'

	if p.next() != '"' {
		return p.errorf("expected '\"'")
	}

	var desc string
	for p.peek() != '"' {
		if p.peek() == utf8.RuneError {
			return p.errorf("unexpected EOF")
		}

		desc += string(p.next())
	}

	if p.next() != '"' {
		return p.errorf("expected '\"'")
	}

	s.description = desc

	return nil
}

func (p *specParser) parseChildren(s *spec) error {
	if p.next() != '[' {
		return p.errorf("expected '['")
	}

	child, err := p.parse()
	if err != nil {
		return err
	}

	s.children = append(s.children, child)

	for {
		if p.peek() == ',' {
			p.next()
		}

		p.skipWhitespace()

		if p.peek() == ']' {
			break
		}

		child, err := p.parse()
		if err != nil {
			return err
		}

		s.children = append(s.children, child)
	}

	if p.next() != ']' {
		return p.errorf("expected ']'")
	}

	return nil
}

func (p *specParser) parseWord(s *spec) error {
	var word string
	for p.peek() >= 'a' && p.peek() <= 'z' || p.peek() >= 'A' && p.peek() <= 'Z' || p.peek() >= '0' && p.peek() <= '9' {
		word += string(p.next())
	}

	if word == "" {
		return p.errorf("expected word")
	}

	s.arg = word

	return nil
}

func (p *specParser) parseArgType(s *spec) error {
	if p.peek() == 's' {
		if !p.consume("string") {
			return p.errorf("expected 'string'")
		}

		s.arg = "string"
		return nil
	}

	if p.peek() == 'i' {
		if !p.consume("ipv") {
			return p.errorf("expected 'ipv4' or 'ipv6'")
		}

		if p.peek() == '4' {
			p.next()
			s.arg = "ipv4"
			return nil
		}

		if p.peek() == '6' {
			p.next()
			s.arg = "ipv6"
			return nil
		}

		return p.errorf("expected 'ipv4' or 'ipv6'")
	}

	return p.errorf("expected 'string', 'ipv4', or 'ipv6'")
}

func (p *specParser) skipWhitespace() {
	for p.peek() == ' ' || p.peek() == '\t' || p.peek() == '\r' || p.peek() == '\n' {
		p.next()
	}
}

func (p *specParser) peek() rune {
	r, _ := utf8.DecodeRuneInString(p.s[p.pos:])

	return r
}

func (p *specParser) next() rune {
	r, size := utf8.DecodeRuneInString(p.s[p.pos:])
	p.pos += size

	if r == '\n' {
		p.line++
		p.col = 0
	} else {
		p.col++
	}

	return r
}

func (p *specParser) consume(s string) bool {
	if !strings.HasPrefix(p.s[p.pos:], s) {
		return false
	}

	for range s {
		p.next()
	}

	return true
}

func (p *specParser) errorf(format string, args ...interface{}) error {
	lines := strings.Split(p.s, "\n")
	line := lines[p.line-1]
	marker := strings.Repeat(" ", p.col) + "^"

	return fmt.Errorf("%d:%d: %s\n\t%s\n\t%s", p.line, p.col, fmt.Sprintf(format, args...), line, marker)
}

func parseSpec(s string) (*spec, error) {
	p := &specParser{
		s:    s,
		line: 1,
	}

	return p.parse()
}
