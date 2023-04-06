package commands

import (
	"errors"
	"fmt"
	"net/netip"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// An example spec:
// literal:show?"Show system information"[
//		literal:version!Hfunc()?"Show version information",
//		literal:ip[
//			literal:route[
//				paramter:ipv4!Hfunc(addr)?"Show route for IPv4 address",
//			]
//		],
//		literal:bgp[
//			literal:neighbors!Hfunc()[
//				parameter:ipv4!A!Hfunc(addr)?"Show BGP neighbor",
//			]
//		]
// ]
//
// Whitespace between children is not significant. A rough grammar is:
//
// spec <- ws? name id? autocomplete? handler? description? children? ws?
// name <- "choice" / "literal:" word / "param:" paramType
// id <- '.' [1-9][0-9]*
// autocomplete <- "!A"
// handler <- "!H" signature
// description <- '?' '"' [^"]* '"'
// children <- '[' spec (',' spec)* ','? ws? ']'
// signature <- "func(" handlerParam (ws? ',' handlerParam)* ")"
// word <- [a-zA-Z0-9]+
// paramType <- "string" / "ipv4" / "ipv6"
// handlerParam <- "string" / "addr"
// ws <- [ \t\r\n]*
//
// All nodes of the same type and ID (.1, .2, etc.) must be reference equal to each other.
// You only need to specify other attributes on the first node of a given type and ID.

type commandSpec struct {
	t               nodeType
	value           string
	id              int
	handler         *reflect.Type
	hasAutocomplete bool
	description     string
	children        []*commandSpec
}

type commandSpecParser struct {
	s    string
	pos  int
	line int
	col  int
}

func (p *commandSpecParser) parseSpec() (*commandSpec, error) {
	s := &commandSpec{}

	p.skipWhitespace()

	if err := p.parseName(s); err != nil {
		return nil, err
	}

	if p.peek() == '.' {
		if err := p.parseID(s); err != nil {
			return nil, err
		}
	}

	if p.startsWith("!A") {
		if err := p.parseAutocomplete(s); err != nil {
			return nil, err
		}
	}

	if p.peek() == '!' {
		if err := p.parseHandler(s); err != nil {
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

func (p *commandSpecParser) parseName(s *commandSpec) error {
	if p.peek() == 'c' {
		if !p.consume("choice") {
			return p.errorf("expected 'fork'")
		}

		s.t = ntChoice
		return nil
	}

	if p.peek() == 'l' {
		if !p.consume("literal:") {
			return p.errorf("expected 'literal:'")
		}

		s.t = ntLiteral
		if err := p.parseWord(s); err != nil {
			return err
		}

		return nil
	}

	if p.peek() == 'p' {
		if !p.consume("param:") {
			return p.errorf("expected 'param:'")
		}

		paramType, err := p.parseParamType()
		if err != nil {
			return err
		}

		switch paramType {
		case "ipv4":
			s.t = ntParamIPv4
		case "ipv6":
			s.t = ntParamIPv6
		case "string":
			s.t = ntParamString
		default:
			return p.errorf("invalid parameter type %s", paramType)
		}

		return nil
	}

	return p.errorf("expected 'fork', 'join', 'literal:', or 'argument:'")
}

func (p *commandSpecParser) parseID(s *commandSpec) error {
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

func (p *commandSpecParser) parseHandler(s *commandSpec) error {
	p.next() // consume the '!'
	p.next() // consume the 'H'

	if err := p.parseSignature(s); err != nil {
		return err
	}

	return nil
}

func (p *commandSpecParser) parseAutocomplete(s *commandSpec) error {
	p.next() // consume the '!'
	p.next() // consume the 'A'

	s.hasAutocomplete = true

	return nil
}

func (p *commandSpecParser) parseDescription(s *commandSpec) error {
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

func (p *commandSpecParser) parseChildren(s *commandSpec) error {
	if p.next() != '[' {
		return p.errorf("expected '['")
	}

	child, err := p.parseSpec()
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

		child, err := p.parseSpec()
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

func (p *commandSpecParser) parseSignature(s *commandSpec) error {
	if !p.consume("func(") {
		return p.errorf("expected 'func('")
	}

	args := make([]string, 0)
	arg, err := p.parseHandlerParam()
	if err != nil {
		return err
	}

	args = append(args, arg)

	for {
		if p.peek() == ',' {
			p.next()
		}

		p.skipWhitespace()

		if p.peek() == ')' {
			break
		}

		arg, err := p.parseHandlerParam()
		if err != nil {
			return err
		}

		args = append(args, arg)
	}

	if p.next() != ')' {
		return p.errorf("expected ')'")
	}

	types := make([]reflect.Type, len(args))
	for i, arg := range args {
		switch arg {
		case "string":
			types[i] = reflect.TypeOf("")
		case "addr":
			types[i] = reflect.TypeOf(netip.Addr{})
		default:
			return p.errorf("invalid argument type %s", arg)
		}
	}

	ret := []reflect.Type{reflect.TypeOf(errors.New(""))}
	handler := reflect.FuncOf(types, ret, false)
	s.handler = &handler

	return nil
}

func (p *commandSpecParser) parseWord(s *commandSpec) error {
	var word string
	for p.peek() >= 'a' && p.peek() <= 'z' || p.peek() >= 'A' && p.peek() <= 'Z' || p.peek() >= '0' && p.peek() <= '9' {
		word += string(p.next())
	}

	if word == "" {
		return p.errorf("expected word")
	}

	s.value = word

	return nil
}

func (p *commandSpecParser) parseParamType() (string, error) {
	if p.peek() == 's' {
		if !p.consume("string") {
			return "", p.errorf("expected 'string'")
		}

		return "string", nil
	}

	if p.peek() == 'i' {
		if !p.consume("ipv") {
			return "", p.errorf("expected 'ipv4' or 'ipv6'")
		}

		if p.peek() == '4' {
			p.next()
			return "ipv4", nil
		}

		if p.peek() == '6' {
			p.next()
			return "ipv6", nil
		}

		return "", p.errorf("expected 'ipv4' or 'ipv6'")
	}

	return "", p.errorf("expected 'string', 'ipv4', or 'ipv6'")
}

func (p *commandSpecParser) parseHandlerParam() (string, error) {
	if p.peek() == 's' {
		if !p.consume("string") {
			return "", p.errorf("expected 'string'")
		}

		return "string", nil
	}

	if p.peek() == 'a' {
		if !p.consume("addr") {
			return "", p.errorf("expected 'addr'")
		}

		return "addr", nil
	}

	return "", p.errorf("expected 'string' or 'addr'")
}

func (p *commandSpecParser) skipWhitespace() {
	for p.peek() == ' ' || p.peek() == '\t' || p.peek() == '\r' || p.peek() == '\n' {
		p.next()
	}
}

func (p *commandSpecParser) peek() rune {
	r, _ := utf8.DecodeRuneInString(p.s[p.pos:])

	return r
}

func (p *commandSpecParser) next() rune {
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

func (p *commandSpecParser) startsWith(s string) bool {
	return strings.HasPrefix(p.s[p.pos:], s)
}

func (p *commandSpecParser) consume(s string) bool {
	if !strings.HasPrefix(p.s[p.pos:], s) {
		return false
	}

	for range s {
		p.next()
	}

	return true
}

func (p *commandSpecParser) errorf(format string, args ...interface{}) error {
	lines := strings.Split(p.s, "\n")
	line := lines[p.line-1]
	marker := strings.Repeat(" ", p.col) + "^"

	return fmt.Errorf("%d:%d: %s\n\t%s\n\t%s", p.line, p.col, fmt.Sprintf(format, args...), line, marker)
}

func parseSpec(s string) (*commandSpec, error) {
	p := &commandSpecParser{
		s:    s,
		line: 1,
	}

	return p.parseSpec()
}

func (s *commandSpec) pathComponent() string {
	var name string

	switch s.t {
	case ntLiteral:
		name = "literal:" + s.value
	case ntParamString:
		name = "param:string"
	case ntParamIPv4:
		name = "param:ipv4"
	case ntParamIPv6:
		name = "param:ipv6"
	case ntChoice:
		name = "choice"
	default:
		panic("unreachable")
	}

	if s.id != 0 {
		name += "." + strconv.Itoa(s.id)
	}

	return name
}

type commandSpecMatcher struct {
	references map[string]*Node
}

func newCommandSpecMatcher() *commandSpecMatcher {
	return &commandSpecMatcher{
		references: make(map[string]*Node),
	}
}

func (m *commandSpecMatcher) match(path string, n *Node, s *commandSpec) error {
	var ref *Node

	if s.id != 0 {
		key := s.pathComponent()
		var ok bool
		if ref, ok = m.references[key]; ok {
			if n != ref {
				return fmt.Errorf("%s: expected %p to be equal to %p", path, n, ref)
			}

			return nil
		} else {
			m.references[key] = n
		}
	}

	if s.t != n.t {
		return fmt.Errorf("%s: expected type %v, got %v", path, s.t, n.t)
	}

	switch s.t {
	case ntLiteral:
		if n.value != s.value {
			return fmt.Errorf("%s: expected literal:%s, got literal:%s", path, s.value, n.value)
		}

		if s.description != n.description {
			return fmt.Errorf("%s: expected description %q, got %q", path, s.description, n.description)
		}

		if s.handler == nil && n.handlerFunc.IsValid() {
			return fmt.Errorf("%s: expected no handler, got %v", path, n.handlerFunc.Type())
		} else if s.handler != nil && (!n.handlerFunc.IsValid() || *s.handler != n.handlerFunc.Type()) {
			return fmt.Errorf("%s: expected handler %v, got %v", path, s.handler, n.handlerFunc.Type())
		}
	case ntParamString, ntParamIPv4, ntParamIPv6:
		if s.description != n.description {
			return fmt.Errorf("%s: expected description %q, got %q", path, s.description, n.description)
		}

		if s.handler == nil && n.handlerFunc.IsValid() {
			return fmt.Errorf("%s: expected no handler, got %v", path, n.handlerFunc.Type())
		} else if s.handler != nil && (!n.handlerFunc.IsValid() || *s.handler != n.handlerFunc.Type()) {
			return fmt.Errorf("%s: expected handler %v, got %v", path, s.handler, n.handlerFunc.Type())
		}

		if s.hasAutocomplete && n.autocompleteFunc == nil {
			return fmt.Errorf("%s: expected autocomplete, got none", path)
		} else if !s.hasAutocomplete && n.autocompleteFunc != nil {
			return fmt.Errorf("%s: expected no autocomplete, got %T", path, n.autocompleteFunc)
		}
	case ntChoice:
		// noop, children are checked below
	default:
		return fmt.Errorf("%s: unknown type %q", path, s.t)
	}

	if len(s.children) != len(n.children) {
		return fmt.Errorf("%s: expected %d children, got %d", path, len(s.children), len(n.children))
	}

	for i, child := range s.children {
		err := m.match(path+"/"+child.pathComponent(), n.children[i], child)
		if err != nil {
			return err
		}
	}

	return nil
}

func AssertMatchesCommandSpec(t *testing.T, s string, n *Node) {
	t.Helper()

	spec, err := parseSpec(s)
	if err != nil {
		t.Fatal(err)
	}

	m := newCommandSpecMatcher()
	err = m.match("/"+spec.pathComponent(), n, spec)
	if err != nil {
		t.Fatal(err)
	}
}
