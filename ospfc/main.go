package main

import (
	"errors"
	"fmt"
	"sort"
)

func commonPrefixLen(a, b string) int {
	i := 0
	for ; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}

type edge struct {
	label string
	node  *node
}

type node struct {
	hasValue  bool
	value     any
	edgeIndex []byte
	edges     []*edge
}

func (n *node) store(key string, value any) {
	if len(key) == 0 {
		n.hasValue = true
		n.value = value
		return
	}

	for {
		i := sort.Search(len(n.edgeIndex), func(i int) bool {
			return n.edgeIndex[i] >= key[0]
		})

		if i < len(n.edgeIndex) && n.edgeIndex[i] == key[0] {
			// edge found
			e := n.edges[i]
			prefixLen := commonPrefixLen(e.label, key)

			if prefixLen == len(e.label) && prefixLen == len(key) {
				// exact match, overwrite
				e.node.hasValue = true
				e.node.value = value
				return
			} else if prefixLen == len(e.label) {
				// e.label is a prefix of key
				key = key[prefixLen:]
				n = e.node
			} else {
				// prefixLen < len(n.label) && prefixLen < len(key)

				// split
				intermediateNode := &node{
					edgeIndex: []byte{e.label[prefixLen]},
					edges:     []*edge{{label: e.label[prefixLen:], node: e.node}},
				}

				e.label = e.label[:prefixLen]
				e.node = intermediateNode

				key = key[prefixLen:]
				n = intermediateNode
			}
		} else if i < len(n.edgeIndex) {
			// insert edge
			n.edgeIndex = append(n.edgeIndex, 0)
			copy(n.edgeIndex[i+1:], n.edgeIndex[i:])
			n.edgeIndex[i] = key[0]

			n.edges = append(n.edges, nil)
			copy(n.edges[i+1:], n.edges[i:])
			n.edges[i] = &edge{label: key, node: &node{hasValue: true, value: value}}

			return
		} else {
			// append edge
			n.edgeIndex = append(n.edgeIndex, key[0])
			n.edges = append(n.edges, &edge{label: key, node: &node{hasValue: true, value: value}})
			return
		}
	}
}

func (n *node) load(key string) (any, bool) {
	if len(key) == 0 {
		return n.value, n.hasValue
	}

	for {
		i := sort.Search(len(n.edgeIndex), func(i int) bool {
			return n.edgeIndex[i] >= key[0]
		})

		if i < len(n.edgeIndex) && n.edgeIndex[i] == key[0] {
			// edge found
			e := n.edges[i]
			prefixLen := commonPrefixLen(e.label, key)

			if prefixLen == len(e.label) && prefixLen == len(key) {
				// exact match
				return e.node.value, e.node.hasValue
			} else if prefixLen == len(e.label) {
				// e.label is a prefix of key
				key = key[prefixLen:]
				n = e.node
			} else {
				// prefixLen < len(n.label) && prefixLen < len(key)
				return nil, false
			}
		} else {
			// no edge found
			return nil, false
		}
	}
}

var (
	errSkipAll    = errors.New("skip all")
	errSkipPrefix = errors.New("skip prefix")
)

type walkFunc func(key string, value any) error

func (n *node) walk(fn walkFunc) error {
	var walk func(key string, n *node) error
	walk = func(key string, n *node) error {
		if n.hasValue {
			err := fn(key, n.value)
			if err != nil {
				return err
			}
		}

		for _, e := range n.edges {
			err := walk(key+e.label, e.node)
			if err != nil {
				return err
			}
		}

		return nil
	}

	err := walk("", n)
	if err == errSkipAll {
		return nil
	}

	return err
}

type walkBytesFunc func(s string) error

// calls fn for each byte in the tree, even when the byte falls inside an edge
func (n *node) walkBytes(fn walkBytesFunc) error {
	if !n.hasValue && len(n.edges) == 0 {
		return nil
	}

	var walk func(key string, n *node) error
	walk = func(key string, n *node) error {
		if err := fn(key); err != nil {
			return err
		}

	Edges:
		for _, e := range n.edges {
			for i := 0; i < len(e.label)-1; i++ {
				err := fn(key + string(e.label[:i+1]))
				if err == errSkipPrefix {
					continue Edges
				} else if err != nil {
					return err
				}
			}

			err := walk(key+e.label, e.node)
			if err == errSkipPrefix {
				continue
			} else if err != nil {
				return err
			}
		}

		return nil
	}

	err := walk("", n)
	if err == errSkipAll {
		return nil
	}

	return err
}

func main() {
	n := &node{}
	n.store("show version", 1)
	n.store("show version detail", 2)
	n.store("show name", 3)
	n.store("show version funny", 4)

	// fmt.Println(n.load("show version"))
	// fmt.Println(n.load("show version detail"))
	// fmt.Println(n.load("show name"))
	// fmt.Println(n.load("show version funny"))
	// fmt.Println(n.load("show ver"))
	// fmt.Println(n.load("nothing going on here"))

	err := n.walk(func(key string, value any) error {
		fmt.Printf("%#v: %#v\n", key, value)
		return nil
	})

	if err != nil {
		panic(err)
	}

	err = n.walkBytes(func(s string) error {
		fmt.Printf("%#v\n", s)
		return nil
	})

	if err != nil {
		panic(err)
	}

	// t.walk(func(s string) {
	// 	fmt.Printf("%#v\n", s)
	// })

	// fmt.Println(t.hasPrefix("show ip ospf"))
	// fmt.Println(t.hasPrefix("show ip ospf neighbor"))
	// fmt.Println(t.hasPrefix("show ip ospf neighbor detail"))
	// fmt.Println(t.hasPrefix("sh"))
}
