//go:generate goyacc -o parser.go parser.y

package commands

import (
	"fmt"
	"reflect"
)

type Graph interface {
	Name() string
	SetDescription(string)
	SetHandlerFunc(reflect.Value)
	SetAutocompleteFunc(func(string) ([]string, error))
	Merge(Graph) Graph
	Children() []Graph
}

type literal struct {
	value       string
	description string
	handlerFunc reflect.Value
	child       Graph
}

func (l *literal) Name() string {
	return "literal:" + l.value
}

func (l *literal) SetDescription(description string) {
	if l.child == nil {
		l.description = description
	} else {
		l.child.SetDescription(description)
	}
}

func (l *literal) SetHandlerFunc(handlerFunc reflect.Value) {
	if l.child == nil {
		l.handlerFunc = handlerFunc
	} else {
		l.child.SetHandlerFunc(handlerFunc)
	}
}

func (l *literal) SetAutocompleteFunc(autocompleteFunc func(string) ([]string, error)) {
	if l.child == nil {
		panic("(*literal).SetAutocompleteFunc: no child")
	}

	l.child.SetAutocompleteFunc(autocompleteFunc)
}

func (l *literal) Merge(other Graph) Graph {
	// if the other one is a leteral, and has the same value
	// merge its properties into us, printing warnings if we already have the property set,
	// and then merge its child into our child

	if otherLiteral, ok := other.(*literal); ok && otherLiteral.value == l.value {
		if otherLiteral.description != "" && l.description != "" {
			fmt.Printf("warning: overwriting description for literal %q\n", l.value)
			l.description = otherLiteral.description
		}

		if otherLiteral.handlerFunc.IsValid() && l.handlerFunc.IsValid() {
			fmt.Printf("warning: overwriting handler for literal %q\n", l.value)
			l.handlerFunc = otherLiteral.handlerFunc
		}

		if otherLiteral.child != nil && l.child != nil {
			l.child = l.child.Merge(otherLiteral.child)
		} else if otherLiteral.child != nil {
			l.child = otherLiteral.child
		}
	}

	// if the other one is a fork, merge us into the fork
	if fork, ok := other.(*fork); ok {
		return fork.Merge(l)
	}

	// otherwise, create a fork with us and other as children
	return &fork{children: map[string]Graph{l.Name(): l, other.Name(): other}}
}

func (l *literal) Children() []Graph {
	if l.child == nil {
		return nil
	}

	return []Graph{l.child}
}

type argumentType int

const (
	_ argumentType = iota
	argumentTypeString
	argumentTypeIPv4
	argumentTypeIPv6
)

func (t argumentType) String() string {
	switch t {
	case argumentTypeString:
		return "string"
	case argumentTypeIPv4:
		return "ipv4"
	case argumentTypeIPv6:
		return "ipv6"
	default:
		panic("unknown param type")
	}
}

type argument struct {
	t                argumentType
	description      string
	handlerFunc      reflect.Value
	autocompleteFunc func(string) ([]string, error)
	child            Graph
}

func (p *argument) Name() string {
	return "argument:" + p.t.String()
}

func (p *argument) SetDescription(description string) {
	if p.child == nil {
		p.description = description
	} else {
		p.child.SetDescription(description)
	}
}

func (p *argument) SetHandlerFunc(handlerFunc reflect.Value) {
	if p.child == nil {
		p.handlerFunc = handlerFunc
	} else {
		p.child.SetHandlerFunc(handlerFunc)
	}
}

func (p *argument) SetAutocompleteFunc(autocompleteFunc func(string) ([]string, error)) {
	if p.child == nil {
		p.autocompleteFunc = autocompleteFunc
	} else {
		p.child.SetAutocompleteFunc(autocompleteFunc)
	}
}

func (a *argument) Merge(other Graph) Graph {
	if otherArgument, ok := other.(*argument); ok && otherArgument.t == a.t {
		if otherArgument.description != "" && a.description != "" {
			fmt.Printf("warning: overwriting description for argument %q:\nold: %s\nnew: %s", a.t, a.description, otherArgument.description)
			a.description = otherArgument.description
		}

		if otherArgument.handlerFunc.IsValid() && a.handlerFunc.IsValid() {
			fmt.Printf("warning: overwriting handler for argument %q\n", a.t)
			a.handlerFunc = otherArgument.handlerFunc
		}

		if otherArgument.autocompleteFunc != nil && a.autocompleteFunc != nil {
			fmt.Printf("warning: overwriting autocomplete for argument %q\n", a.t)
			a.autocompleteFunc = otherArgument.autocompleteFunc
		}

		if otherArgument.child != nil && a.child != nil {
			a.child = a.child.Merge(otherArgument.child)
		} else if otherArgument.child != nil {
			a.child = otherArgument.child
		}
	}

	if fork, ok := other.(*fork); ok {
		return fork.Merge(a)
	}

	return &fork{children: map[string]Graph{a.Name(): a, other.Name(): other}}
}

func (a *argument) Children() []Graph {
	if a.child == nil {
		return nil
	}

	return []Graph{a.child}
}

type fork struct {
	children map[string]Graph
}

func (f *fork) Name() string {
	return "fork"
}

func (f *fork) SetDescription(description string) {
	for _, child := range f.children {
		child.SetDescription(description)
	}
}

func (f *fork) SetAutocompleteFunc(autocompleteFunc func(string) ([]string, error)) {
	if len(f.children) == 0 {
		panic("no children")
	}

	// We only call this on graphs that haven't been merged yet, so
	// just take the first fork.
	for _, child := range f.children {
		child.SetAutocompleteFunc(autocompleteFunc)
		return
	}
}

func (f *fork) SetHandlerFunc(handlerFunc reflect.Value) {
	if len(f.children) == 0 {
		panic("no children")
	}

	// The handler func gets set on the join, so we just need to find it
	// and set it there. We'll traverse the first child.
	for _, child := range f.children {
		child.SetHandlerFunc(handlerFunc)
		return
	}
}

func (f *fork) Merge(other Graph) Graph {
	// if the other one is a fork, merge its children into us
	if fork, ok := other.(*fork); ok {
		for _, child := range fork.children {
			f.Merge(child)
		}
		return f
	}

	// if the other has the same name of one of our children, merge it into that child
	for name, child := range f.children {
		if child.Name() == other.Name() {
			f.children[name] = child.Merge(other)
			return f
		}
	}

	// otherwise, add it as a new child
	f.children[other.Name()] = other
	return f
}

func (f *fork) Children() []Graph {
	children := make([]Graph, 0, len(f.children))
	for _, child := range f.children {
		children = append(children, child)
	}

	return children
}

type join struct {
	child Graph
}

func (j *join) Name() string {
	return "join:" + j.child.Name()
}

func (j *join) SetDescription(description string) {
	if j.child == nil {
		panic("can't SetDescription on a join with no child")
	}

	j.child.SetDescription(description)
}

func (j *join) SetHandlerFunc(handlerFunc reflect.Value) {
	if j.child == nil {
		panic("can't SetHandlerFunc on a join with no child")
	}

	j.child.SetHandlerFunc(handlerFunc)
}

func (j *join) SetAutocompleteFunc(autocompleteFunc func(string) ([]string, error)) {
	if j.child == nil {
		panic("can't SetAutocompleteFunc on a join with no child")
	}

	j.child.SetAutocompleteFunc(autocompleteFunc)
}

func (j *join) Merge(other Graph) Graph {
	panic("can't merge into a join")
}

func (j *join) Children() []Graph {
	if j.child == nil {
		return nil
	}

	return []Graph{j.child}
}

func ParseDeclaration(s string) (Graph, reflect.Type, error) {
	l := newLexer(s)

	p := yyNewParser()
	p.Parse(l)

	if l.err != nil {
		return nil, nil, l.err
	}

	return nil, nil, nil
}
