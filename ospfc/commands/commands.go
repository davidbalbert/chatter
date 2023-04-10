package commands

import (
	"fmt"
	"io"
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

func (t nodeType) paramType(inChoice bool) reflect.Type {
	switch t {
	case ntParamString:
		return reflect.TypeOf("")
	case ntParamIPv4:
		return reflect.TypeOf(netip.Addr{})
	case ntParamIPv6:
		return reflect.TypeOf(netip.Addr{})
	case ntLiteral:
		if inChoice {
			return reflect.TypeOf(false)
		} else {
			return nil
		}
	default:
		return nil
	}
}

type AutocompleteFunc (func(string) ([]string, error))

type Node struct {
	t                nodeType
	value            string
	description      string
	handlerFunc      reflect.Value
	autocompleteFunc AutocompleteFunc
	children         []*Node
	paramTypes       []reflect.Type // the types of all parameters in this node and its parents

	// true if this node was explicitly declared as a choice (e.g. "<A.B.C.D|X:X:X::X>") as
	// compared to a choice that was implictly created by merging two literals
	explicitChoice bool
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

func (n *Node) String() string {
	switch n.t {
	case ntLiteral, ntParamString:
		return n.value
	case ntParamIPv4:
		return "A.B.C.D"
	case ntParamIPv6:
		return "X:X:X::X"
	case ntChoice:
		var b strings.Builder
		b.WriteString("<")
		for i, child := range n.children {
			if i > 0 {
				b.WriteString("|")
			}
			b.WriteString(child.String())
		}
		b.WriteString(">")
		return b.String()
	default:
		panic("unreachable")
	}
}

func containsType(types []reflect.Type, t reflect.Type) bool {
	for _, t2 := range types {
		if t2 == t {
			return true
		}
	}
	return false
}

func (n *Node) updateParamTypesWithTypes(types []reflect.Type) {
	clonedTypes := make([]reflect.Type, len(types))
	copy(clonedTypes, types)

	if n.t == ntChoice {
		for _, child := range n.children {
			t := child.t.paramType(true)
			// Only add the type if it's not already in the list, but boolean types,
			// which represent literals, are always added.
			//
			// Examples:
			// 1. "show <A.B.C.D|X:X:X::X|all>" -> func(addr netip.Addr, all bool)
			// 2. "show <A.B.C.D|X:X:X::X>" -> func(addr netip.Addr)
			// 3. "show <A.B.C.D|all|X:X:X::X>" -> func(addr netip.Addr, all bool)
			// 4. "show <ip|ipv6>" -> func(ip, ipv6 bool)
			if t != nil && (!containsType(clonedTypes, t) || t == reflect.TypeOf(false)) {
				clonedTypes = append(clonedTypes, t)
			}
		}

		for _, child := range n.children {
			child.paramTypes = clonedTypes

			for _, child2 := range child.children {
				child2.updateParamTypesWithTypes(clonedTypes)
			}
		}
	} else {
		t := n.t.paramType(false)
		if t != nil {
			clonedTypes = append(clonedTypes, t)
		}

		n.paramTypes = clonedTypes

		for _, child := range n.children {
			child.updateParamTypesWithTypes(clonedTypes)
		}
	}
}

func (n *Node) updateParamTypes() {
	n.updateParamTypesWithTypes([]reflect.Type{reflect.TypeOf((*io.Writer)(nil)).Elem()})
}

func findIndex(n *Node, ns []*Node) int {
	for i, child := range ns {
		if child.id() == n.id() {
			return i
		}
	}
	return -1
}

func (n1 *Node) mergeAttributes(path string, n2 *Node) {
	if n1.description != "" && n2.description != "" {
		fmt.Printf("Warning: overwriting description for %s from %q to %q\n", path+"/"+n1.id(), n1.description, n2.description)
	}
	if n2.description != "" {
		n1.description = n2.description
	}

	if n1.autocompleteFunc != nil && n2.autocompleteFunc != nil {
		fmt.Printf("Warning: overwriting autocomplete function for %s\n", path+"/"+n1.id())
	}
	if n2.autocompleteFunc != nil {
		n1.autocompleteFunc = n2.autocompleteFunc
	}

	if n1.handlerFunc.IsValid() && n2.handlerFunc.IsValid() {
		fmt.Printf("Warning: overwriting handler function for %s\n", path+"/"+n1.id())
	}
	if n2.handlerFunc.IsValid() {
		n1.handlerFunc = n2.handlerFunc
	}
}

func (n1 *Node) mergeWithPath(path string, n2 *Node) (*Node, error) {
	if n1 == nil {
		return n2, nil
	}

	if n2 == nil {
		return n1, nil
	}

	if n1.t == ntChoice && n2.t == ntChoice {
		if n1.explicitChoice && n2.explicitChoice {
			if len(n1.children) != len(n2.children) {
				return nil, fmt.Errorf("%s: cannot merge explicit choice %q with %q", path, n1, n2)
			}

			for i, child1 := range n1.children {
				child2 := n2.children[i]

				if child1.id() != child2.id() {
					return nil, fmt.Errorf("%s: cannot merge explicit choice %q with %q", path, n1, n2)
				}
			}
		} else if n1.explicitChoice {
			return nil, fmt.Errorf("%s: cannot merge explicit choice %q with %q", path, n1, n2)
		} else if n2.explicitChoice {
			return nil, fmt.Errorf("%s: cannot merge explicit choice %q into %q", path, n2, n1)
		}

		for _, child2 := range n2.children {
			i := findIndex(child2, n1.children)

			if i == -1 {
				n1.children = append(n1.children, child2)
			} else {
				n1.children[i].mergeWithPath(path+"/"+n1.children[i].id(), child2)
			}

			var grandchild *Node
			if len(child2.children) == 1 && i != -1 {
				grandchild = n1.children[i].children[0]
			} else if len(child2.children) == 1 {
				for _, child1 := range n1.children {
					if len(child1.children) > 0 && child1.children[0].id() == child2.children[0].id() {
						gc, err := child1.children[0].mergeWithPath(path+"/"+child1.id(), child2.children[0])
						if err != nil {
							return nil, err
						}

						grandchild = gc
						break
					}
				}
			}

			if grandchild != nil {
				for _, child1 := range n1.children {
					if len(child1.children) > 0 && child1.children[0].id() == grandchild.id() {
						child1.children[0] = grandchild
					}
				}
			}
		}

		return n1, nil
	} else if n1.t == ntChoice {
		if n1.explicitChoice {
			return nil, fmt.Errorf("%s: cannot merge explicit choice %q with %q", path, n1, n2)
		}

		c2 := &Node{t: ntChoice, children: []*Node{n2}}
		return n1.mergeWithPath(path, c2)
	} else if n2.t == ntChoice {
		if n2.explicitChoice {
			return nil, fmt.Errorf("%s: cannot merge explicit choice %q into %q", path, n2, n1)
		}

		c1 := &Node{t: ntChoice, children: []*Node{n1}}
		return c1.mergeWithPath(path, n2)
	} else { // n1 and n2 are non-choice nodes
		if n1.id() == n2.id() {
			n1.mergeAttributes(path+"/"+n1.id(), n2)

			if len(n1.children) > 1 || len(n2.children) > 1 {
				panic("non-choice nodes should have at most one child")
			}

			if len(n1.children) == 1 && len(n2.children) == 1 {
				c, err := n1.children[0].mergeWithPath(path+"/"+n1.id(), n2.children[0])
				if err != nil {
					return nil, err
				}
				n1.children[0] = c
			} else if len(n2.children) == 1 {
				n1.children = append(n1.children, n2.children[0])
			}

			return n1, nil
		} else {
			c1 := &Node{t: ntChoice, children: []*Node{n1}}
			c2 := &Node{t: ntChoice, children: []*Node{n2}}

			return c1.mergeWithPath(path, c2)
		}
	}
}

func (n1 *Node) Merge(nodes ...*Node) (*Node, error) {
	for _, n2 := range nodes {
		var err error
		n1, err = n1.mergeWithPath("", n2)
		if err != nil {
			return nil, err
		}
	}

	return n1, nil
}

func containsNode(ns []*Node, n *Node) bool {
	for _, child := range ns {
		if child.id() == n.id() {
			return true
		}
	}

	return false
}

func (n *Node) Leaves() []*Node {
	if n.t == ntChoice {
		var leaves []*Node
		for _, child := range n.children {
			ls := child.Leaves()

			for _, l := range ls {
				if !containsNode(leaves, l) {
					leaves = append(leaves, l)
				}
			}
		}

		return leaves
	} else {
		if len(n.children) == 0 {
			return []*Node{n}
		} else {
			return n.children[0].Leaves()
		}
	}
}

func (n *Node) SetHandlerFunc(f any) error {
	if n.t == ntChoice {
		return fmt.Errorf("cannot set handler function for choice node")
	}

	in := n.paramTypes
	out := []reflect.Type{reflect.TypeOf((*error)(nil)).Elem()}
	expected := reflect.FuncOf(in, out, false)

	if reflect.TypeOf(f) != expected {
		return fmt.Errorf("handler function has wrong type: expected %s, got %s", expected, reflect.TypeOf(f))
	}

	n.handlerFunc = reflect.ValueOf(f)

	return nil
}

func (n *Node) SetDescription(desc string) error {
	if n.t == ntChoice {
		return fmt.Errorf("cannot set description for choice node")
	}

	n.description = desc

	return nil
}

func (n *Node) SetAutocompleteFunc(fn AutocompleteFunc) error {
	if n.t == ntChoice {
		return fmt.Errorf("cannot set autocomplete function for choice node")
	} else if n.t == ntLiteral {
		return fmt.Errorf("cannot set autocomplete function for literal node")
	}

	n.autocompleteFunc = fn

	return nil
}

type Match struct {
	node       *Node
	next       *Match
	isComplete bool            // leaf node has a valid handler function
	input      string          // the input that matched this node
	args       []reflect.Value // arguments for the handler function
}

func (m *Match) IsComplete() bool {
	return m.isComplete
}

func (m *Match) Invoker() (*Invoker, error) {
	if !m.isComplete {
		return nil, fmt.Errorf("match is not complete")
	}

	var args []reflect.Value
	var handlerFunc reflect.Value
	for {
		args = append(args, m.args...)

		if m.next == nil {
			handlerFunc = m.node.handlerFunc
			break
		}

		m = m.next
	}

	if !handlerFunc.IsValid() {
		return nil, fmt.Errorf("invariant violation: handler function is not valid but isComplete is true")
	}

	return &Invoker{
		args:        args,
		handlerFunc: handlerFunc,
	}, nil
}

type Invoker struct {
	args        []reflect.Value
	handlerFunc reflect.Value
}

func (i *Invoker) Run(w io.Writer) error {
	args := make([]reflect.Value, len(i.args)+1)
	args[0] = reflect.ValueOf(w)
	copy(args[1:], i.args)

	results := i.handlerFunc.Call(args)
	err := results[0].Interface()
	if err != nil {
		return err.(error)
	}

	return nil
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
		for i, child := range n.children {
			ms := child.matchTokens(tokens)

			for _, m := range ms {
				if n.explicitChoice {
					var args []reflect.Value
					argCount := 0
					firstIndexOfMatchType := -1

					for j, child := range n.children {
						argType := child.t.paramType(true)

						if argType == m.node.t.paramType(true) && firstIndexOfMatchType == -1 {
							firstIndexOfMatchType = argCount
						}

						// <A.B.C.D|all> -> [1.2.3.4, false] or [netip.Addr{}, true]
						// <A.B.C.D|X:X:X::X> -> [netip.Addr{...}]
						// <A.B.C.D|X:X:X::X|all> -> [netip.Addr{...}, false] or [netip.Addr{}, true]
						// <A.B.C.D|all|X:X:X::X> -> [netip.Addr{...}, false] or [netip.Addr{}, true]

						shouldAppend := false
						if firstIndexOfMatchType != -1 {
							if j == firstIndexOfMatchType {
								shouldAppend = true
							} else if j > firstIndexOfMatchType {
								if child.t == ntLiteral {
									shouldAppend = true
								}
							}
						}

						if i == j {
							var arg reflect.Value
							if m.node.t == ntLiteral {
								arg = reflect.ValueOf(true)
							} else {
								arg = m.args[0]
							}

							if j == firstIndexOfMatchType {
								args = append(args, arg)
							} else {
								args[firstIndexOfMatchType] = arg
							}

							argCount++
						} else if shouldAppend {
							args = append(args, reflect.Zero(argType))
							argCount++
						}
					}

					m.args = args
				}

				matches = append(matches, m)
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
				args:  []reflect.Value{reflect.ValueOf(tokens[0])},
			}
		} else if n.t == ntParamIPv4 {
			addr, err := netip.ParseAddr(tokens[0])
			if err == nil && addr.Is4() {
				match = &Match{
					node:  n,
					input: tokens[0],
					args:  []reflect.Value{reflect.ValueOf(addr)},
				}
			}
		} else if n.t == ntParamIPv6 {
			addr, err := netip.ParseAddr(tokens[0])
			if err == nil && addr.Is6() {
				match = &Match{
					node:  n,
					input: tokens[0],
					args:  []reflect.Value{reflect.ValueOf(addr)},
				}
			}
		} else {
			panic("unreachable")
		}

		if match == nil {
			return nil
		}

		if len(tokens) == 1 && n.handlerFunc.IsValid() {
			match.isComplete = true
		}

		if len(tokens) == 1 {
			return []*Match{match}
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

			if childMatch.isComplete {
				dupedMatch.isComplete = true
			}

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

func ParseDeclaration(s string) (*Node, error) {
	p := &commandParser{s: s}
	n, err := p.parseCommand()
	if err != nil {
		return nil, err
	}

	n.updateParamTypes()

	return n, nil
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
		t:              ntChoice,
		children:       []*Node{child},
		explicitChoice: true,
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
