package commands

import (
	"fmt"
	"net/netip"
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
	children         []*Node
}

func (n1 *Node) mergeWithPath(path string, n2 *Node) *Node {
	if n1 == nil {
		return n2
	}

	if n2 == nil {
		return n1
	}

	return nil
}

func (n1 *Node) Merge(n2 *Node) *Node {
	return n1.mergeWithPath("", n2)
}

type Match struct {
	node  *Node
	next  *Match
	input string     // the input that matched this node
	addr  netip.Addr // address for IPv4/IPv6 parameters
}

func (m *Match) length() int {
	if m == nil {
		return 0
	}

	return 1 + m.next.length()
}

func (n *Node) matchTokens(tokens []string) []*Match {
	if n.t == ntChoice {
		var matches []*Match
		for _, child := range n.children {
			ms := child.matchTokens(tokens)
			if len(ms) > 0 {
				matches = append(matches, ms...)
			}
		}

		return matches
	} else {
		var match *Match
		if n.t == ntLiteral {
			if strings.HasPrefix(n.value, tokens[0]) {
				match = &Match{
					node:  n,
					input: tokens[0],
				}
			}
		} else if n.t == ntParamString {
			match = &Match{
				node:  n,
				input: tokens[0],
			}
		} else if n.t == ntParamIPv4 {
			addr, err := netip.ParseAddr(tokens[0])
			if err == nil && addr.Is4() {
				match = &Match{
					node:  n,
					input: tokens[0],
					addr:  addr,
				}
			}
		} else if n.t == ntParamIPv6 {
			addr, err := netip.ParseAddr(tokens[0])
			if err == nil && addr.Is6() {
				match = &Match{
					node:  n,
					input: tokens[0],
					addr:  addr,
				}
			}
		} else {
			panic("unreachable")
		}

		if match == nil {
			return nil
		}

		if len(tokens) == 1 && n.handlerFunc.IsValid() {
			return []*Match{match}
		} else if len(tokens) == 1 {
			return nil
		}

		if len(n.children) == 0 {
			return nil
		} else if len(n.children) > 1 {
			panic("non-choice node with multiple children")
		}

		var matches []*Match
		child := n.children[0]
		for _, childMatch := range child.matchTokens(tokens[1:]) {
			dupedMatch := *match
			dupedMatch.next = childMatch
			matches = append(matches, &dupedMatch)
		}

		return matches
	}
}

func (n *Node) Match(input string) []*Match {
	tokens := strings.Fields(input)

	if len(tokens) == 0 {
		return nil
	}

	matches := n.matchTokens(tokens)

	if len(matches) <= 1 {
		return matches
	}

	// If there is more than one match, then our matches are ambiguous. There's one special case disambiguation
	// that we perform: if a literal token matches exactly, we pick that match.
	//
	// Some examples:
	// "show <ip|ipv6>"
	// "sh i" -> ["show ip", "show ipv6"]
	// "sh ip" -> ["show ip"]
	// "sh ipv" -> ["show ipv6"]
	// "sh ipv6" -> ["show ipv6"]
	//
	// "show <ip|ipv6> <route|routes>"
	// "sh i r" -> ["show ip route", "show ip routes", "show ipv6 route", "show ipv6 routes"]
	// "sh ip r" -> ["show ip route", "show ip routes"]
	// "sh ipv r" -> ["show ipv6 route", "show ipv6 routes"]
	// "sh ipv6 r" -> ["show ipv6 route", "show ipv6 routes"]
	// "sh ipv6 rout" -> ["show ipv6 route", "show ipv6 routes"]
	// "sh ipv6 route" -> ["show ipv6 route"]
	// "sh ipv6 routes" -> ["show ipv6 routes"]
	// "sh i rout" -> ["show ip route", "show ip routes", "show ipv6 route", "show ipv6 routes"]
	// "sh i route" -> ["show ip route", "show ipv6 route"]
	// "sh i routes" -> ["show ip routes", "show ipv6 routes"]
	// "sh ip route" -> ["show ip route"]
	// "sh ip routes" -> ["show ip routes"]

	var disambiguated []*Match

	ms := make([]*Match, len(matches))
	copy(ms, matches)

	for _, token := range tokens {
		var exactRoots []*Match
		var nonExactRoots []*Match
		var exactMatches []*Match
		var nonExactMatches []*Match

		for i, m := range ms {
			if m.node.t == ntLiteral && m.node.value == token {
				exactRoots = append(exactRoots, matches[i])
				exactMatches = append(exactMatches, m.next)
			} else {
				nonExactRoots = append(nonExactRoots, matches[i])
				nonExactMatches = append(nonExactMatches, m.next)
			}
		}

		if len(exactRoots) > 0 {
			disambiguated = exactRoots
			ms = exactMatches
		} else {
			disambiguated = nonExactRoots
			ms = nonExactMatches
		}

		if len(disambiguated) <= 1 {
			break
		}
	}

	return disambiguated
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

		if n.t == ntChoice {
			for _, option := range n.children {
				option.children = append(option.children, child)
			}
		} else {
			n.children = append(n.children, child)
		}

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
		children: []*Node{child},
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

		n.children = append(n.children, child)
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

func (p *commandParser) errorf(format string, args ...any) error {
	line := strings.Split(p.s, "\n")[0]
	marker := strings.Repeat(" ", p.pos) + "^"

	return fmt.Errorf("%d: %s\n\t%s\n\t%s", p.pos, fmt.Sprintf(format, args...), line, marker)
}
