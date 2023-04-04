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
}

type UnaryNode interface {
	Node
	id() string
	Description() string
	OverwriteDescription(string)
	HandlerFunc() reflect.Value
	OverwriteHandlerFunc(reflect.Value)
	Child() Node
	mergeAttributes(UnaryNode, Node) UnaryNode
	withChild(Node) UnaryNode
}

type literal struct {
	value       string
	description string
	handlerFunc reflect.Value
	child       Node
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

func (l *literal) Description() string {
	return l.description
}

func (l *literal) HandlerFunc() reflect.Value {
	return l.handlerFunc
}

func (l *literal) Child() Node {
	return l.child
}

func (l *literal) mergeAttributes(u UnaryNode, child Node) UnaryNode {
	newL := *l
	newL.OverwriteDescription(u.Description())
	newL.OverwriteHandlerFunc(u.HandlerFunc())
	newL.child = child
	return &newL
}

func (l *literal) OverwriteDescription(description string) {
	if description != "" {
		if willOverwrite(l.description, description) {
			fmt.Printf("warning: overwriting description for %q: %q -> %q\n", l.id(), l.description, description)
		}
		l.description = description
	}
}

func (l *literal) OverwriteHandlerFunc(handlerFunc reflect.Value) {
	if handlerFunc.IsValid() {
		if l.handlerFunc.IsValid() {
			fmt.Printf("warning: overwriting handler for %q: %v -> %v\n", l.id(), l.handlerFunc, handlerFunc)
		}
		l.handlerFunc = handlerFunc
	}
}

func (l *literal) withChild(child Node) UnaryNode {
	newL := *l
	newL.child = child
	return &newL
}

type argumentString struct {
	description string
	handlerFunc reflect.Value
	child       Node
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

func (a *argumentString) Child() Node {
	return a.child
}

func (a *argumentString) mergeAttributes(u UnaryNode, child Node) UnaryNode {
	newA := *a
	newA.OverwriteDescription(u.Description())
	newA.OverwriteHandlerFunc(u.HandlerFunc())
	newA.child = child
	return &newA
}

func (a *argumentString) Description() string {
	return a.description
}

func (a *argumentString) OverwriteDescription(description string) {
	if description != "" {
		if willOverwrite(a.description, description) {
			fmt.Printf("warning: overwriting description for %q: %q -> %q\n", a.id(), a.description, description)
		}
		a.description = description
	}
}

func (a *argumentString) HandlerFunc() reflect.Value {
	return a.handlerFunc
}

func (a *argumentString) OverwriteHandlerFunc(handlerFunc reflect.Value) {
	if handlerFunc.IsValid() {
		if a.handlerFunc.IsValid() {
			fmt.Printf("warning: overwriting handler for %q: %v -> %v\n", a.id(), a.handlerFunc, handlerFunc)
		}
		a.handlerFunc = handlerFunc
	}
}

func (a *argumentString) withChild(child Node) UnaryNode {
	newA := *a
	newA.child = child
	return &newA
}

type argumentIPv4 struct {
	description string
	handlerFunc reflect.Value
	child       Node
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

func (a *argumentIPv4) Child() Node {
	return a.child
}

func (a *argumentIPv4) mergeAttributes(u UnaryNode, child Node) UnaryNode {
	newA := *a
	newA.OverwriteDescription(u.Description())
	newA.OverwriteHandlerFunc(u.HandlerFunc())
	newA.child = child
	return &newA
}

func (a *argumentIPv4) Description() string {
	return a.description
}

func (a *argumentIPv4) OverwriteDescription(description string) {
	if description != "" {
		if willOverwrite(a.description, description) {
			fmt.Printf("warning: overwriting description for %q: %q -> %q\n", a.id(), a.description, description)
		}
		a.description = description
	}
}

func (a *argumentIPv4) HandlerFunc() reflect.Value {
	return a.handlerFunc
}

func (a *argumentIPv4) OverwriteHandlerFunc(handlerFunc reflect.Value) {
	if handlerFunc.IsValid() {
		if a.handlerFunc.IsValid() {
			fmt.Printf("warning: overwriting handler for %q: %v -> %v\n", a.id(), a.handlerFunc, handlerFunc)
		}
		a.handlerFunc = handlerFunc
	}
}

func (a *argumentIPv4) withChild(child Node) UnaryNode {
	newA := *a
	newA.child = child
	return &newA
}

type argumentIPv6 struct {
	description string
	handlerFunc reflect.Value
	child       Node
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

func (a *argumentIPv6) Child() Node {
	return a.child
}

func (a *argumentIPv6) mergeAttributes(u UnaryNode, child Node) UnaryNode {
	newA := *a
	newA.OverwriteDescription(u.Description())
	newA.OverwriteHandlerFunc(u.HandlerFunc())
	newA.child = child
	return &newA
}

func (a *argumentIPv6) Description() string {
	return a.description
}

func (a *argumentIPv6) OverwriteDescription(description string) {
	if description != "" {
		if willOverwrite(a.description, description) {
			fmt.Printf("warning: overwriting description for %q: %q -> %q\n", a.id(), a.description, description)
		}
		a.description = description
	}
}

func (a *argumentIPv6) HandlerFunc() reflect.Value {
	return a.handlerFunc
}

func (a *argumentIPv6) OverwriteHandlerFunc(handlerFunc reflect.Value) {
	if handlerFunc.IsValid() {
		if a.handlerFunc.IsValid() {
			fmt.Printf("warning: overwriting handler for %q: %v -> %v\n", a.id(), a.handlerFunc, handlerFunc)
		}
		a.handlerFunc = handlerFunc
	}
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
