package main

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type autocompleteNode struct {
	leaf     bool
	result   string
	children map[string]*autocompleteNode
}

type cli struct {
	root *autocompleteNode // maps prefixes to autocompleteNodes
}

func newCLI() *cli {
	return &cli{
		root: &autocompleteNode{
			children: make(map[string]*autocompleteNode),
		},
	}
}

func (c *cli) addCommand(cmd, result string) {

}

func (c *cli) execute(line string) string {
	switch line {
	case "show version":
		return "ospfd version 0.1"
	case "show message":
		return "Hello, world!"
	default:
		return "Unknown command"
	}
}

func (c *cli) autocomplete(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	if key != '\t' {
		return line, pos, false
	}

	// if "show version" is the only possible completion for line, then autocomplete to "show version"
	if strings.HasPrefix("show version", line) {
		return "show version ", 13, true
	} else if strings.HasPrefix("show message", line) {
		return "show message ", 13, true
	} else if strings.HasPrefix("show", line) {
		return "show ", 5, true
	} else if strings.HasPrefix("exit", line) {
		return "exit ", 5, true
	}

	// if strings.HasPrefix("show v", line) {
	// 	return "show version ", 13, true
	// } else if strings.HasPrefix("show m", line) {
	// 	return "show message ", 13, true
	// } else if strings.HasPrefix("show", line) {
	// 	return "show ", 5, true
	// } else if strings.HasPrefix("exit", line) {
	// 	return "exit ", 5, true
	// }

	return line, pos, false
}

func client() int {
	fmt.Printf("%d %d\n", '\t', '?')

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println(err)
		return 1
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	c := newCLI()
	t := term.NewTerminal(os.Stdin, "ospfd# ")
	t.AutoCompleteCallback = c.autocomplete

	for {
		line, err := t.ReadLine()
		if err != nil {
			fmt.Println(err)
			return 1
		}

		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		} else if trimmed == "exit" {
			return 0
		}

		fmt.Printf("%s\r\n", c.execute(trimmed))
	}
}

func main() {
	os.Exit(client())
}
