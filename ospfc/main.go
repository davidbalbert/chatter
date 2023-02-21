package main

import "fmt"

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

type node struct {
	label    string
	children map[byte]*node
}

func newNode(label string) *node {
	return &node{
		label:    label,
		children: make(map[byte]*node),
	}
}

func newRadixTree() *node {
	return newNode("")
}

func commonPrefixLen(a, b string) int {
	i := 0
	for ; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			break
		}
	}
	return i
}

func (root *node) insert(s string) {
	parent := root
	for {
		n := parent.children[s[0]]

		if n == nil {
			parent.children[s[0]] = newNode(s)
			return
		}

		prefixLen := commonPrefixLen(s, n.label)

		if prefixLen == len(s) {
			return
		} else if prefixLen == len(n.label) {
			s = s[prefixLen:]
			parent = n
		} else { // prefixLen < len(n.label) && prefixLen < len(s)
			// split
			nprefix := newNode(n.label[:prefixLen])
			n.label = n.label[prefixLen:]

			parent.children[nprefix.label[0]] = nprefix
			nprefix.children[n.label[0]] = n

			// TODO: this seems logically correct but when
			// I uncomment it, "show version funny" gets
			// printed as "show version  funny"
			// parent = nprefix
		}
	}
}

func (root *node) walk(f func(string)) {
	var walk func(*node, string)
	walk = func(n *node, s string) {
		f(s)

		for _, n := range n.children {
			walk(n, s+n.label)
		}
	}
	walk(root, "")
}

func main() {
	t := newRadixTree()
	// t.insert("show ip ospf")
	// t.insert("show version")
	// t.insert("show ip ospf neighbor")
	// t.insert("show ipsec sa")
	// t.insert("show")

	t.insert("show version")
	t.insert("show version detail")
	t.insert("show name")
	t.insert("show version funny")

	t.walk(func(s string) {
		fmt.Printf("%#v\n", s)
	})

	// fmt.Println(t.hasPrefix("show ip ospf"))
	// fmt.Println(t.hasPrefix("show ip ospf neighbor"))
	// fmt.Println(t.hasPrefix("show ip ospf neighbor detail"))
	// fmt.Println(t.hasPrefix("sh"))
}
