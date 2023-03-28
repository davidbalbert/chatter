package main

import (
	"errors"
	"fmt"
	"strings"
)

// 1. lcp is shorter than both the existing label and the key we're inserting
// therefore, we split

// insert("show version")
// 's' -> "show version"

// insert("show veronica")
// 's' -> "show ver"
//        'o' -> "onica"
//        's' -> "sion"

// 2. lcp is the same length as the existing label (which means the prefix is the same string as the label)
// therefore, we insert another node with the suffix

// insert("show version")
// 's' -> "show version"

// insert("show version detail")
// 's' -> "show version"
//        ' ' -> " detail"

// 3. lcp is the same length as they key we're inserting, but shorter than the existing label
// therefore, we split, but only end up with one child

// insert("show version")
// 's' -> "show version"

// insert("show")
// 's' -> "show"
//        ' ' -> " version"

// 4. same as above, I think? lcp is the same length as they key we're inserting, but shorter than the existing label, so we split the existing label and add a single child with the existing node's children.

// insert("show version")
// insert("show name")

// 's' -> "show "
//        'v' -> "version"
//        'n' -> "name"

// insert("show")

// 's' -> "show"
//        ' ' -> " "
//               'v' -> "version"
//               'n' -> "name"

// 5. Same as above. I wonder if I'll run into issues with treating whitespace specially?

// insert("show version")
// insert("show veronica")

// 's' -> "show ver"
//        'o' -> "onica"
//        's' -> "sion"

// insert("show")

// 's' -> "show"
//        ' ' -> " ver"
//               'o' -> "onica"
//               's' -> "sion"

// The algorithm
//
// insert("show version")
// insert("show version detail")
// insert("show name")
//
// 's' -> "show "
//        'v' -> "version"
//               ' ' -> " detail"
//        'n' -> "name"
//
// insert("show version funny")
//
// n = root
//
// lcp = longestCommonPrefix(s "show version funny", n.label "") // => ""
// s = s[len(lcp):] //=> "show version funny"
//
// // lcp == n.label, therefore get the node corresponding to the first byte of s ('s')
//
// b = s[0] //=> 's'
// n = n[b] || newNode() // n['s']
//
// lcp = longestCommonPrefix(s "show version funny", n.label "show ") // => "show "
// s = s[len(lcp):] //=> "version funny"
//
// // lcp == label, therefore, get the next byte ('v') and see if there's a node for that (creating one if there isn't)
//
// b = s[0] //=> 'v'
// n = n['v'] || newNode()
//
// lcp = longestCommonPrefix(s "version funny", n.label "version") //=> "version"
// s = s[len(lcp):] // " funny"
//
// // lcp == label, therefore, get the next byte of s (' ') and see if there's a node for that, (creating one if there isn't)
//
// b = s[0] //=> ' '
// n = n[b] || newNode()
//
// lcp = longestCommonPrefix(s " funny", n.label " detail") //=> " "
// s = s[len(lcp):] //=> "funny"
//
// // len(lcp) < len(label), therefore, split n at len(lcp)
//
// nleft = newLabel(n.label([:len(lcp)]))
// parent.children[nleft.label[0]] = left
//
// n.label = n.label[len(lcp):]
// nleft.children[n.label[len(lcp)]] = n
//
// 's' -> "show "
//        'v' -> "version"
//               ' ' -> " "
//                      'd' -> "detail"
//        'n' -> "name"
//
//
// nleft.children[s[0]] = newNode(s "funny")
//
// 's' -> "show "
//        'v' -> "version"
//               ' ' -> " "
//                      'd' -> "detail"
//                      'f' -> "funny"
//        'n' -> "name"

func commonPrefixLen(a, b string) int {
	i := 0
	for ; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}

type node struct {
	label    string
	children map[byte]*node
	terminal bool
	value    any
}

type cursor struct {
	node   *node
	offset int
}

type walkBytesFunc func(prefix string, c cursor) error

var errSkipAll = errors.New("skip all")
var errSkipPrefix = errors.New("skip prefix")

// calls fn for every byte. it doesn't matter whether we're terminal or not
func (c cursor) walkBytes(fn walkBytesFunc) error {
	var walk func(prefix string, c cursor, fn walkBytesFunc) error
	walk = func(prefix string, c cursor, fn walkBytesFunc) error {
		n := c.node
		offset := c.offset

		for offset < len(n.label) {
			prefix = prefix + string(n.label[offset])

			if err := fn(prefix, cursor{node: n, offset: offset}); err != nil {
				if err == errSkipPrefix {
					return nil
				}
				return err
			}
			offset++
		}

		if n.children == nil {
			return nil
		}

		for _, child := range n.children {
			if err := walk(prefix, cursor{node: child}, fn); err != nil {
				return err
			}
		}

		return nil
	}

	if err := walk("", c, fn); err != nil && err != errSkipAll {
		return err
	}

	return nil
}

func (c cursor) next() []cursor {
	if c.offset < len(c.node.label)-1 {
		return []cursor{{c.node, c.offset + 1}}
	}

	if c.node.children == nil {
		return nil
	}

	var next []cursor
	for _, child := range c.node.children {
		next = append(next, cursor{node: child})
	}

	return next
}

func (c cursor) value() (any, bool) {
	if c.offset < len(c.node.label) {
		return nil, false
	}

	return c.node.value, c.node.terminal
}

func (c cursor) String() string {
	return fmt.Sprintf("{%#v %d}", c.node.label, c.offset)
}

func (root *node) walkBytes(fn walkBytesFunc) error {
	return cursor{node: root}.walkBytes(fn)
}

func (root *node) store(s string, value any) {
	parent := root
	for {
		if parent.children == nil {
			parent.children = make(map[byte]*node)
		}

		n := parent.children[s[0]]

		if n == nil {
			parent.children[s[0]] = &node{label: s, terminal: true, value: value}
			return
		}

		prefixLen := commonPrefixLen(s, n.label)

		if prefixLen == len(s) {
			n.value = value
			n.terminal = true
			return
		} else if prefixLen == len(n.label) {
			s = s[prefixLen:]
			parent = n
		} else { // prefixLen < len(n.label) && prefixLen < len(s)
			// split
			prefixNode := &node{label: n.label[:prefixLen], children: make(map[byte]*node)}
			n.label = n.label[prefixLen:]

			parent.children[prefixNode.label[0]] = prefixNode
			prefixNode.children[n.label[0]] = n

			s = s[prefixLen:]
			parent = prefixNode
		}
	}
}

func (root *node) load(s string) (value any, ok bool) {
	n := root
	for {
		if n == nil {
			return nil, false
		}

		prefixLen := commonPrefixLen(s, n.label)

		if s == n.label {
			return n.value, n.terminal
		} else if prefixLen == len(n.label) {
			s = s[prefixLen:]
			n = n.children[s[0]]
		} else {
			return nil, false
		}
	}
}

func (root *node) cursors() []cursor {
	if len(root.label) > 0 {
		return []cursor{{node: root}}
	}

	var cursors []cursor
	for _, child := range root.children {
		cursors = append(cursors, cursor{node: child})
	}

	return cursors
}

func (root *node) fuzzyLoad(q string) ([]any, error) {
	cursors := root.cursors()
	words := strings.Fields(q)
	var values []any

	for i, word := range words {
		last := i == len(words)-1
		var next []cursor

		fmt.Printf("1. cursors: %v\n", cursors)

		for _, b := range []byte(word) {
			for _, c := range cursors {
				err := c.walkBytes(func(prefix string, c1 cursor) error {
					fmt.Printf("b: %c, %c\n", b, prefix[len(prefix)-1])

					if b == prefix[len(prefix)-1] {
						next = append(next, c1.next()...)
					}

					return errSkipPrefix
				})

				if err != nil {
					return nil, err
				}
			}

			cursors = next
			next = nil
		}

		fmt.Printf("2. cursors: %v\n", cursors)

		// right now there's a bug. if we have a query like "a b" and we have a two keys "aay bee" and "aay bee cee"
		// we expect to load the value for "aay bee", but we get no values because the following code reads through
		// a space. If we're on the last iteration, we need to read up to a space, but not through it.

		for _, c := range cursors {
			err := c.walkBytes(func(prefix string, c1 cursor) error {
				b := prefix[len(prefix)-1]
				fmt.Printf("b: %c\n", b)

				if last && b == ' ' {
					value, ok := c1.value()
					if ok {
						values = append(values, value)
					}

					return errSkipPrefix
				} else if b == ' ' {
					next = append(next, c1.next()...)
					return errSkipPrefix
				}

				return nil
			})

			if err != nil {
				return nil, err
			}
		}

		cursors = next

		fmt.Printf("3. cursors: %v\n", cursors)
	}

	return values, nil
}

func main() {
	t := &node{}
	// t.insert("show ip ospf")
	// t.insert("show version")
	// t.insert("show ip ospf neighbor")
	// t.insert("show ipsec sa")
	// t.insert("show")

	t.store("show version", 1)
	t.store("show version detail", 2)
	t.store("show name", 3)
	t.store("show version funny", 4)

	values, err := t.fuzzyLoad("sh v")
	if err != nil {
		panic(err)
	}

	fmt.Println(values)

	// t.walk(func(s string) {
	// 	fmt.Printf("%#v\n", s)
	// })

	// fmt.Println(t.hasPrefix("show ip ospf"))
	// fmt.Println(t.hasPrefix("show ip ospf neighbor"))
	// fmt.Println(t.hasPrefix("show ip ospf neighbor detail"))
	// fmt.Println(t.hasPrefix("sh"))
}
