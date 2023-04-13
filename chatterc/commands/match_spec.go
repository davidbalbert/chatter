package commands

import (
	"fmt"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// A specification for matches returned by matching an input string against a command definition.
//
// For example, given this command definition and associated graph:
//
// 	show <ip|interface>
//  show -> choice -> ip
//  				\ interface
//
// The input string "sh i" should return two matches: "show ip" and "show interface".
// This can be represented by the following match specification:
//
// 	show ip
// 	show interface
//
//
// Match specs also support parameters. Given the following command definition:
//  show bgp neighbors <A.B.C.D|X:X:X::X|all>
//
// The input "show bgp neighbors 1.2.3.4" should match the following spec:
//  show bgp neighbors ipv4:1.2.3.4

// Rough grammar:
//
// specs <- (ws|eol)* match+ (ws|eol)* eof
// spec <- part (ws+ part)* ws* eol
// part <- ipv4Param / ipv6Param / stringParam / literal
// ipv4Param <- "ipv4:" ipv4addr
// ipv6Param <- "ipv6:" ipv6addr
// stringParam <- "string:" [^ \t\n]+
// literal <- [a-zA-Z][a-zA-Z0-9_-]*
// ipv4addr <- [0-9.]+        // validated by netip.ParseAddr
// ipv6addr <- [0-9a-fA-F:.]+ // validated by netip.ParseAddr
// ws <- [ \t]
// eol <- ("\n" | eof)
// eof <- !.

type matchSpecPart struct {
	t    nodeType
	s    string
	addr netip.Addr
}

type matchSpec struct {
	parts []*matchSpecPart
}

func (s *matchSpec) String() string {
	var b strings.Builder

	for i, part := range s.parts {
		if i > 0 {
			b.WriteString(" ")
		}

		switch part.t {
		case ntLiteral:
			b.WriteString(part.s)
		case ntParamString:
			b.WriteString("string:")
			b.WriteString(part.s)
		case ntParamIPv4:
			b.WriteString("ipv4:")
			b.WriteString(part.addr.String())
		case ntParamIPv6:
			b.WriteString("ipv6:")
			b.WriteString(part.addr.String())
		default:
			panic("unreachable")
		}
	}

	return b.String()
}

func (s *matchSpec) match(m *Match) error {
	if len(s.parts) != m.length() {
		return fmt.Errorf("%s: expected %d parts, got %d", s.String(), len(s.parts), m.length())
	}

	for _, part := range s.parts {
		switch part.t {
		case ntLiteral:
			if m.node.t != ntLiteral || m.node.value != part.s {
				return fmt.Errorf("%s: expected literal %s, got %s", s.String(), part.s, m.node.value)
			}
		case ntParamString:
			if m.node.t != ntParamString || m.input != part.s {
				return fmt.Errorf("%s: expected string param %s, got %s", s.String(), part.s, m.input)
			}
		case ntParamIPv4:
			if m.node.t != ntParamIPv4 {
				return fmt.Errorf("%s: expected ipv4 param, got %s", s.String(), m.node.t)
			}

			var actual netip.Addr
			argType := reflect.TypeOf(actual)
			for _, arg := range m.args {
				if arg.Type() == argType {
					actual = arg.Interface().(netip.Addr)
					break
				}
			}

			if m.node.t != ntParamIPv4 || actual.Compare(part.addr) != 0 {
				return fmt.Errorf("%s: expected ipv4 param %s, got %s", s.String(), part.addr.String(), actual.String())
			}
		case ntParamIPv6:
			if m.node.t != ntParamIPv6 {
				return fmt.Errorf("%s: expected ipv4 param, got %s", s.String(), m.node.t)
			}

			var actual netip.Addr
			argType := reflect.TypeOf(actual)
			for _, arg := range m.args {
				if arg.Type() == argType {
					actual = arg.Interface().(netip.Addr)
					break
				}
			}

			if m.node.t != ntParamIPv6 || actual.Compare(part.addr) != 0 {
				return fmt.Errorf("%s: expected ipv6 param %s, got %s", s.String(), part.addr.String(), actual.String())
			}
		default:
			panic("unreachable")
		}

		m = m.next
	}

	return nil
}

type matchSpecs struct {
	specs []*matchSpec
}

func (s *matchSpecs) match(m []*Match) error {
	if len(s.specs) != len(m) {
		return fmt.Errorf("expected %d matches, got %d", len(s.specs), len(m))
	}

	for i, spec := range s.specs {
		if err := spec.match(m[i]); err != nil {
			return err
		}
	}

	return nil
}

type matchSpecParser struct {
	s    string
	pos  int
	line int
	col  int
}

func (p *matchSpecParser) parseSpecs() (*matchSpecs, error) {
	p.skipWhitespaceIncludingNewlines()

	spec, err := p.parseSpec()
	if err != nil {
		return nil, err
	}

	specs := &matchSpecs{[]*matchSpec{spec}}

	for {
		p.skipWhitespaceIncludingNewlines()

		if p.isEOF() {
			return specs, nil
		}

		spec, err := p.parseSpec()
		if err != nil {
			return nil, err
		}

		specs.specs = append(specs.specs, spec)
	}
}

func (p *matchSpecParser) parseSpec() (*matchSpec, error) {
	part, err := p.parsePart()
	if err != nil {
		return nil, err
	}

	spec := &matchSpec{[]*matchSpecPart{part}}

	for {
		p.skipWhitespace()

		if p.isSpecEnd() {
			return spec, nil
		}

		part, err := p.parsePart()
		if err != nil {
			return nil, err
		}

		spec.parts = append(spec.parts, part)
	}
}

func (p *matchSpecParser) parsePart() (*matchSpecPart, error) {
	if p.consume("ipv4:") {
		return p.parseIPv4Param()
	}

	if p.consume("ipv6:") {
		return p.parseIPv6Param()
	}

	if p.consume("string:") {
		return p.parseStringParam()
	}

	return p.parseLiteral()
}

func (p *matchSpecParser) parseIPv4Param() (*matchSpecPart, error) {
	s, err := p.parseIPv4Addr()
	if err != nil {
		return nil, err
	}

	addr, err := netip.ParseAddr(s)
	if err != nil {
		return nil, p.errorf("invalid IPv4 address %s: %s", s, err)
	}

	return &matchSpecPart{ntParamIPv4, s, addr}, nil
}

func (p *matchSpecParser) parseIPv6Param() (*matchSpecPart, error) {
	s, err := p.parseIPv6Addr()
	if err != nil {
		return nil, err
	}

	addr, err := netip.ParseAddr(s)
	if err != nil {
		return nil, p.errorf("invalid IPv6 address %s: %s", s, err)
	}

	return &matchSpecPart{ntParamIPv6, s, addr}, nil
}

func (p *matchSpecParser) parseStringParam() (*matchSpecPart, error) {
	if p.isStringEnd() {
		return nil, p.errorf("unexpected end of string parameter")
	}

	runes := []rune{p.next()}

	for !p.isStringEnd() {
		runes = append(runes, p.next())
	}

	return &matchSpecPart{ntParamString, string(runes), netip.Addr{}}, nil
}

func (p *matchSpecParser) parseLiteral() (*matchSpecPart, error) {
	if !p.isLitStart() {
		return nil, p.errorf("unexpected character while reading literal: %c", p.peek())
	}

	runes := []rune{p.next()}

	for p.isLitRest() {
		runes = append(runes, p.next())
	}

	return &matchSpecPart{ntLiteral, string(runes), netip.Addr{}}, nil
}

func (p *matchSpecParser) parseIPv4Addr() (string, error) {
	if !p.isIPv4() {
		return "", p.errorf("unexpected character while reading IPv4 address: %c", p.peek())
	}

	runes := []rune{p.next()}

	for p.isIPv4() {
		runes = append(runes, p.next())
	}

	return string(runes), nil
}

func (p *matchSpecParser) parseIPv6Addr() (string, error) {
	if !p.isIPv6() {
		return "", p.errorf("unexpected character while reading IPv6 address: %c", p.peek())
	}

	runes := []rune{p.next()}

	for p.isIPv6() {
		runes = append(runes, p.next())
	}

	return string(runes), nil
}

func (p *matchSpecParser) isLitStart() bool {
	return (p.peek() >= 'a' && p.peek() <= 'z') || (p.peek() >= 'A' && p.peek() <= 'Z')
}

func (p *matchSpecParser) isLitRest() bool {
	return p.isLitStart() || (p.peek() >= '0' && p.peek() <= '9') || p.peek() == '_' || p.peek() == '-'
}

func (p *matchSpecParser) isStringEnd() bool {
	return p.isEOF() || p.peek() == ' ' || p.peek() == '\t' || p.peek() == '\n'
}

func (p *matchSpecParser) isIPv4() bool {
	return (p.peek() >= '0' && p.peek() <= '9') || p.peek() == '.'
}

func (p *matchSpecParser) isIPv6() bool {
	return (p.peek() >= '0' && p.peek() <= '9') || (p.peek() >= 'a' && p.peek() <= 'f') || (p.peek() >= 'A' && p.peek() <= 'F') || p.peek() == ':' || p.peek() == '.'
}

func (p *matchSpecParser) isSpecEnd() bool {
	return p.isEOF() || p.peek() == '\n'
}

func (p *matchSpecParser) skipWhitespaceIncludingNewlines() {
	for !p.isEOF() && (p.peek() == ' ' || p.peek() == '\t' || p.peek() == '\n') {
		p.next()
	}
}

func (p *matchSpecParser) skipWhitespace() {
	for !p.isEOF() && (p.peek() == ' ' || p.peek() == '\t') {
		p.next()
	}
}

func (p *matchSpecParser) isEOF() bool {
	return p.pos >= len(p.s)
}

func (p *matchSpecParser) consume(s string) bool {
	if strings.HasPrefix(p.s[p.pos:], s) {
		for range s {
			p.next()
		}

		return true
	}

	return false
}

func (p *matchSpecParser) peek() rune {
	r, _ := utf8.DecodeRuneInString(p.s[p.pos:])

	return r
}

func (p *matchSpecParser) next() rune {
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

func (p *matchSpecParser) errorf(format string, args ...any) error {
	lines := strings.Split(p.s, "\n")
	line := lines[p.line-1]
	marker := strings.Repeat(" ", p.col) + "^"

	return fmt.Errorf("%d:%d: %s\n\t%s\n\t%s", p.line, p.col, fmt.Sprintf(format, args...), line, marker)
}

func parseMatchSpecs(s string) (*matchSpecs, error) {
	p := &matchSpecParser{
		s:    s,
		line: 1,
	}

	return p.parseSpecs()
}

func AssertMatchesMatchSpec(t *testing.T, s string, matches []*Match) {
	t.Helper()

	specs, err := parseMatchSpecs(s)
	if err != nil {
		t.Fatal(err)
	}

	err = specs.match(matches)
	if err != nil {
		t.Fatal(err)
	}
}
