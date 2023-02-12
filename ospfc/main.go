package main

import "fmt"

type node struct {
	prefix   string
	children map[rune]*node
}

func newNode(prefix string) *node {
	return &node{prefix: prefix, children: make(map[rune]*node)}
}

func (n *node) walk(f func(string)) {
	f(n.prefix)

	for _, v := range n.children {
		v.walk(f)
	}
}

type radixTree struct {
	root *node
}

func newRadixTree() *radixTree {
	return &radixTree{root: newNode("")}
}

func (t *radixTree) insert(s string) {
	n := t.root

	for i, r := range s {
		if _, ok := n.children[r]; !ok {
			n.children[r] = newNode(s[:i+1])
		}

		n = n.children[r]
	}
}

func (t *radixTree) walk(f func(string)) {
	t.root.walk(f)
}

func (t *radixTree) hasPrefix(s string) bool {
	n := t.root

	for _, r := range s {
		if _, ok := n.children[r]; !ok {
			return false
		}

		n = n.children[r]
	}

	return true
}

func main() {
	t := newRadixTree()
	t.insert("show ip ospf")
	t.insert("show version")
	t.insert("show ip ospf neighbor")

	t.walk(func(s string) {
		fmt.Println(s)
	})

	fmt.Println(t.hasPrefix("show ip ospf"))
	fmt.Println(t.hasPrefix("show ip ospf neighbor"))
	fmt.Println(t.hasPrefix("show ip ospf neighbor detail"))
	fmt.Println(t.hasPrefix("sh"))
}
