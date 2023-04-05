package commands

import (
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"
)

// An example command definition
// show bgp neighbors <A.B.C.D|X:X:X::X|all> detail
//
// This should return a graph with the following structure:
//
// literal:show -> literal:bgp -> literal:neighbors -> choice -> parameter:ipv4 -> literal:detail
// 															   \ parameter:ipv6 /
// 															   \ literal:all    /
//
//
// Rough grammar for command definitions:
//
// command <- ws (element ws)+ eol
// element <- choice / ipv4Param / ipv6Param / stringParam / literal
// choice <- "<" ws element (ws "|" ws element)* ws ">"
// ipv4Param <- "A.B.C.D"
// ipv6Param <- "X:X:X::X"
// stringParam <- [A-Z]+
// literal <- [a-zA-Z][a-zA-Z0-9_-]*
// ws <- " "*

type nodeType int

const (
	ntLiteral nodeType = iota
	ntParamString
	ntParamIPv4
	ntParamIPv6
	ntChoice
)

func (t nodeType) String() string {
	switch t {
	case ntLiteral:
		return "literal"
	case ntParamString:
		return "param:string"
	case ntParamIPv4:
		return "param:ipv4"
	case ntParamIPv6:
		return "param:ipv6"
	case ntChoice:
		return "choice"
	default:
		panic("unreachable")
	}
}

type AutocompleteFunc (func(string) []string)

type Node struct {
	t                nodeType
	value            string
	description      string
	handlerFunc      reflect.Value
	autocompleteFunc AutocompleteFunc
	children         map[string]*Node
}

func (n *Node) Children() []*Node {
	var children []*Node
	for _, child := range n.children {
		children = append(children, child)
	}
	return children
}

func (n *Node) id() string {
	switch n.t {
	case ntLiteral:
		return "literal:" + n.value
	case ntParamString:
		return "param:string"
	case ntParamIPv4:
		return "param:ipv4"
	case ntParamIPv6:
		return "param:ipv6"
	case ntChoice:
		return "choice"
	default:
		panic("unreachable")
	}
}

type commandParser struct {
	s   string
	pos int
}

func parseCommand(s string) (*Node, error) {
	p := &commandParser{s: s}
	return p.parseCommand()
}

func (p *commandParser) parseCommand() (*Node, error) {
	p.skipWhitespace()

	root, err := p.parseElement()
	if err != nil {
		return nil, err
	}

	n := root

	for {
		p.skipWhitespace()

		if p.isEOL() {
			break
		}

		child, err := p.parseElement()
		if err != nil {
			return nil, err
		}

		if n.children == nil {
			n.children = make(map[string]*Node)
		}

		n.children[child.id()] = child
		n = child
	}

	return root, nil
}

func (p *commandParser) parseElement() (*Node, error) {
	if p.hasPrefix("<") {
		return p.parseChoice()
	}

	if p.hasPrefix("A.B.C.D") {
		n := p.parseIPv4Param()
		if n != nil {
			return n, nil
		}
	}

	if p.hasPrefix("X:X:X::X") {
		n := p.parseIPv6Param()
		if n != nil {
			return n, nil
		}
	}

	if p.peek() >= 'A' && p.peek() <= 'Z' {
		n := p.parseStringParam()
		if n != nil {
			return n, nil
		}
	}

	if p.isLitStart() {
		return p.parseLiteral()
	}

	return nil, p.errorf("unexpected character while parsing element: %c", p.peek())
}

func (p *commandParser) parseChoice() (*Node, error) {
	pos := p.mark()
	p.next() // consume '<'

	p.skipWhitespace()

	child, err := p.parseElement()
	if err != nil {
		return nil, err
	}

	n := &Node{
		t:        ntChoice,
		children: map[string]*Node{child.id(): child},
	}

	for {
		p.skipWhitespace()

		if !p.consume("|") {
			break
		}

		p.skipWhitespace()

		child, err := p.parseElement()
		if err != nil {
			return nil, err
		}

		n.children[child.id()] = child
	}

	p.skipWhitespace()

	if !p.consume(">") {
		p.reset(pos)
		return nil, p.errorf("expected '>'")
	}

	if !p.isElementEnd() {
		p.reset(pos)
		return nil, p.errorf("unexpected character after choice: %c", p.peek())
	}

	return n, nil
}

func (p *commandParser) parseIPv4Param() *Node {
	pos := p.mark()

	p.consume("A.B.C.D")

	if !p.isElementEnd() {
		p.reset(pos)
		return nil
	}

	return &Node{
		t: ntParamIPv4,
	}
}

func (p *commandParser) parseIPv6Param() *Node {
	pos := p.mark()

	p.consume("X:X:X::X")

	if !p.isElementEnd() {
		p.reset(pos)
		return nil
	}

	return &Node{
		t: ntParamIPv6,
	}
}

func (p *commandParser) parseStringParam() *Node {
	pos := p.mark()

	runes := []rune{p.next()}

	for p.peek() >= 'A' && p.peek() <= 'Z' {
		runes = append(runes, p.next())
	}

	if !p.isElementEnd() {
		p.reset(pos)
		return nil
	}

	return &Node{
		t:     ntParamString,
		value: string(runes),
	}
}

func (p *commandParser) parseLiteral() (*Node, error) {
	pos := p.mark()

	runes := []rune{p.next()}

	for p.isLitRest() {
		runes = append(runes, p.next())
	}

	if !p.isElementEnd() {
		p.reset(pos)
		return nil, p.errorf("unexpected character while parsing literal: %c", p.peek())
	}

	return &Node{
		t:     ntLiteral,
		value: string(runes),
	}, nil
}

func (p *commandParser) isLitStart() bool {
	return (p.peek() >= 'a' && p.peek() <= 'z') || (p.peek() >= 'A' && p.peek() <= 'Z')
}

func (p *commandParser) isLitRest() bool {
	return p.isLitStart() || (p.peek() >= '0' && p.peek() <= '9') || p.peek() == '_' || p.peek() == '-'
}

func (p *commandParser) isElementEnd() bool {
	return p.isEOL() || p.peek() == ' ' || p.peek() == '|' || p.peek() == '>'
}

func (p *commandParser) skipWhitespace() {
	for p.peek() == ' ' {
		p.next()
	}
}

func (p *commandParser) consume(s string) bool {
	if strings.HasPrefix(p.s[p.pos:], s) {
		p.pos += len(s)

		return true
	}

	return false
}

func (p *commandParser) hasPrefix(s string) bool {
	return strings.HasPrefix(p.s[p.pos:], s)
}

func (p *commandParser) isEOL() bool {
	return p.isEOF() || p.peek() == '\n'
}

func (p *commandParser) isEOF() bool {
	return p.pos >= len(p.s)
}

func (p *commandParser) peek() rune {
	r, _ := utf8.DecodeRuneInString(p.s[p.pos:])

	return r
}

func (p *commandParser) next() rune {
	r, size := utf8.DecodeRuneInString(p.s[p.pos:])
	p.pos += size

	return r
}

func (p *commandParser) mark() int {
	return p.pos
}

func (p *commandParser) reset(pos int) {
	p.pos = pos
}

func (p *commandParser) errorf(format string, args ...interface{}) error {
	line := strings.Split(p.s, "\n")[0]
	marker := strings.Repeat(" ", p.pos) + "^"

	return fmt.Errorf("%d: %s\n\t%s\n\t%s", p.pos, fmt.Sprintf(format, args...), line, marker)
}
