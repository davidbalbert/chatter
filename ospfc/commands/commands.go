//go:generate goyacc -o parser.go parser.y

package commands

import (
	"fmt"
	"reflect"
)

func willOverwrite(old, new string) bool {
	if old != "" && old != new {
		return true
	} else {
		return false
	}
}

type AutocompleteFunc func(string) ([]string, error)

type Node interface {
	Merge(Node) Node
	SetDescription(string)
	SetHandlerFunc(reflect.Value)
	SetAutocompleteFunc(AutocompleteFunc)
}

type UnaryNode interface {
	Node
	id() string
	Description() string
	OverwriteDescription(UnaryNode, string)
	HandlerFunc() reflect.Value
	OverwriteHandlerFunc(UnaryNode, reflect.Value)
	AutocompleteFunc() AutocompleteFunc
	OverwriteAutocompleteFunc(UnaryNode, AutocompleteFunc)
	Child() Node
	mergeAttributes(UnaryNode, Node) UnaryNode
	withChild(Node) UnaryNode
}

type unaryBase struct {
	description      string
	handlerFunc      reflect.Value
	autocompleteFunc AutocompleteFunc
	child            Node
}

func (u *unaryBase) Description() string {
	return u.description
}

func (u *unaryBase) OverwriteDescription(target UnaryNode, description string) {
	if description != "" {
		if willOverwrite(u.description, description) {
			fmt.Printf("warning: overwriting description for %q: %q -> %q\n", target.id(), u.description, description)
		}
		u.description = description
	}
}

func (u *unaryBase) HandlerFunc() reflect.Value {
	return u.handlerFunc
}

func (u *unaryBase) SetDescription(description string) {
	if u.child != nil {
		u.child.SetDescription(description)
	} else {
		u.description = description
	}
}

func (u *unaryBase) SetHandlerFunc(handlerFunc reflect.Value) {
	if u.child != nil {
		u.child.SetHandlerFunc(handlerFunc)
	} else {
		u.handlerFunc = handlerFunc
	}
}

func (u *unaryBase) OverwriteHandlerFunc(target UnaryNode, handlerFunc reflect.Value) {
	if handlerFunc.IsValid() {
		if u.handlerFunc.IsValid() {
			fmt.Printf("warning: overwriting handler for %q: %v -> %v\n", target.id(), u.handlerFunc, handlerFunc)
		}
		u.handlerFunc = handlerFunc
	}
}

func (u *unaryBase) AutocompleteFunc() AutocompleteFunc {
	return u.autocompleteFunc
}

func (u *unaryBase) SetAutocompleteFunc(autocompleteFunc AutocompleteFunc) {
	if u.child != nil {
		u.child.SetAutocompleteFunc(autocompleteFunc)
	} else {
		u.autocompleteFunc = autocompleteFunc
	}
}

func (u *unaryBase) OverwriteAutocompleteFunc(target UnaryNode, autocompleteFunc AutocompleteFunc) {
	if autocompleteFunc != nil {
		if u.autocompleteFunc != nil {
			fmt.Printf("warning: overwriting autocomplete for %q: %v -> %v\n", target.id(), u.autocompleteFunc, autocompleteFunc)
		}
		u.autocompleteFunc = autocompleteFunc
	}
}

func (u *unaryBase) Child() Node {
	return u.child
}

type literal struct {
	unaryBase
	value string
}

func (l *literal) id() string {
	return "literal:" + l.value
}

func (l *literal) Merge(other Node) Node {
	if u, ok := other.(UnaryNode); ok && l.id() == u.id() {
		return l.mergeAttributes(u, l.Child().Merge(u.Child()))
	} else if u, ok := other.(UnaryNode); ok {
		return newFork(l, u)
	} else if f, ok := other.(*fork); ok {
		return f.Merge(l)
	} else {
		panic(fmt.Sprintf("unexpected type %T", other))
	}
}

func (l *literal) mergeAttributes(u UnaryNode, child Node) UnaryNode {
	newL := *l
	newL.OverwriteDescription(l, u.Description())
	newL.OverwriteHandlerFunc(l, u.HandlerFunc())
	newL.OverwriteAutocompleteFunc(l, u.AutocompleteFunc())
	newL.child = child
	return &newL
}

func (l *literal) withChild(child Node) UnaryNode {
	newL := *l
	newL.child = child
	return &newL
}

type argumentString struct {
	unaryBase
}

func (a *argumentString) id() string {
	return "argument:string"
}

func (a *argumentString) Merge(other Node) Node {
	if u, ok := other.(UnaryNode); ok && u.id() == u.id() {
		return a.mergeAttributes(u, a.Child().Merge(u.Child()))
	} else if u, ok := other.(UnaryNode); ok {
		return newFork(a, u)
	} else if f, ok := other.(*fork); ok {
		return f.Merge(a)
	} else {
		panic(fmt.Sprintf("unexpected type %T", other))
	}
}

func (a *argumentString) mergeAttributes(u UnaryNode, child Node) UnaryNode {
	newA := *a
	newA.OverwriteDescription(a, u.Description())
	newA.OverwriteHandlerFunc(a, u.HandlerFunc())
	newA.OverwriteAutocompleteFunc(a, u.AutocompleteFunc())
	newA.child = child
	return &newA
}

func (a *argumentString) withChild(child Node) UnaryNode {
	newA := *a
	newA.child = child
	return &newA
}

type argumentIPv4 struct {
	unaryBase
}

func (a *argumentIPv4) id() string {
	return "argument:ipv4"
}

func (a *argumentIPv4) Merge(other Node) Node {
	if u, ok := other.(UnaryNode); ok && u.id() == u.id() {
		return a.mergeAttributes(u, a.Child().Merge(u.Child()))
	} else if u, ok := other.(UnaryNode); ok {
		return newFork(a, u)
	} else if f, ok := other.(*fork); ok {
		return f.Merge(a)
	} else {
		panic(fmt.Sprintf("unexpected type %T", other))
	}
}

func (a *argumentIPv4) mergeAttributes(u UnaryNode, child Node) UnaryNode {
	newA := *a
	newA.OverwriteDescription(a, u.Description())
	newA.OverwriteHandlerFunc(a, u.HandlerFunc())
	newA.OverwriteAutocompleteFunc(a, u.AutocompleteFunc())
	newA.child = child
	return &newA
}

func (a *argumentIPv4) withChild(child Node) UnaryNode {
	newA := *a
	newA.child = child
	return &newA
}

type argumentIPv6 struct {
	unaryBase
}

func (a *argumentIPv6) id() string {
	return "argument:ipv6"
}

func (a *argumentIPv6) Merge(other Node) Node {
	if u, ok := other.(UnaryNode); ok && u.id() == u.id() {
		return a.mergeAttributes(u, a.Child().Merge(u.Child()))
	} else if u, ok := other.(UnaryNode); ok {
		return newFork(a, u)
	} else if f, ok := other.(*fork); ok {
		return f.Merge(a)
	} else {
		panic(fmt.Sprintf("unexpected type %T", other))
	}
}

func (a *argumentIPv6) mergeAttributes(u UnaryNode, child Node) UnaryNode {
	newA := *a
	newA.OverwriteDescription(a, u.Description())
	newA.OverwriteHandlerFunc(a, u.HandlerFunc())
	newA.OverwriteAutocompleteFunc(a, u.AutocompleteFunc())
	newA.child = child
	return &newA
}

func (a *argumentIPv6) withChild(child Node) UnaryNode {
	newA := *a
	newA.child = child
	return &newA
}

type fork struct {
	grandchild Node
	children   map[string]UnaryNode
}

func newFork(a, b UnaryNode) *fork {
	grandchild := a.Child().Merge(b.Child())

	return &fork{
		grandchild: grandchild,
		children: map[string]UnaryNode{
			a.id(): a.withChild(grandchild),
			b.id(): b.withChild(grandchild),
		},
	}
}

func (f *fork) Merge(other Node) Node {
	if f2, ok := other.(*fork); ok {
		return f.mergeFork(f2)
	} else if u, ok := other.(UnaryNode); ok {
		return f.mergeUnary(u)
	} else {
		panic(fmt.Sprintf("unexpected type %T", other))
	}
}

func (f *fork) mergeFork(f2 *fork) *fork {
	grandchild := f.grandchild.Merge(f2.grandchild)

	children := make(map[string]UnaryNode, len(f.children))
	for id, child := range f.children {
		children[id] = child.withChild(grandchild)
	}

	for id, c2 := range f2.children {
		if c1, ok := children[id]; ok {
			children[id] = c1.mergeAttributes(c2, grandchild)
		} else {
			children[id] = c2.withChild(grandchild)
		}
	}

	return &fork{
		grandchild: f.grandchild.Merge(f2.grandchild),
		children:   children,
	}
}

func (f *fork) mergeUnary(u UnaryNode) *fork {
	if existing, ok := f.children[u.id()]; ok {
		newChild := existing.Merge(u)
		newUnary, ok := newChild.(UnaryNode)
		if !ok {
			panic(fmt.Sprintf("unexpected type %T", newChild))
		}

		grandchild := f.grandchild.Merge(newUnary.Child())

		children := make(map[string]UnaryNode, len(f.children))
		for id, child := range f.children {
			if id == u.id() {
				children[id] = newUnary.withChild(grandchild)
			} else {
				children[id] = child.withChild(grandchild)
			}
		}

		return &fork{
			grandchild: grandchild,
			children:   children,
		}
	} else {
		grandchild := f.grandchild.Merge(u.Child())

		children := make(map[string]UnaryNode, len(f.children)+1)
		for id, child := range f.children {
			children[id] = child.withChild(grandchild)
		}

		children[u.id()] = u.withChild(grandchild)

		return &fork{
			grandchild: grandchild,
			children:   children,
		}
	}
}

func (f *fork) SetDescription(s string) {
	for _, child := range f.children {
		child.SetDescription(s)
	}
}

func (f *fork) SetHandlerFunc(handlerFunc reflect.Value) {
	for _, child := range f.children {
		child.SetHandlerFunc(handlerFunc)
	}
}

func (f *fork) SetAutocompleteFunc(fn AutocompleteFunc) {
	for _, child := range f.children {
		child.SetAutocompleteFunc(fn)
	}
}

func ParseDeclaration(s string) (Node, reflect.Type, error) {
	l := newLexer(s)
	p := yyNewParser()

	p.Parse(l)
	if l.err != nil {
		return nil, nil, l.err
	}

	return nil, nil, nil
}
