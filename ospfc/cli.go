package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/davidbalbert/ospfd/ospfc/commands"
	"golang.org/x/term"
)

func commonPrefixLen(ss ...string) int {
	if len(ss) == 0 {
		return 0
	}

	prefixLen := len(ss[0])
	for _, s := range ss[1:] {
		i := 0
		for ; i < len(s) && i < prefixLen; i++ {
			if s[i] != ss[0][i] {
				break
			}
		}
		prefixLen = i
	}

	return prefixLen
}

type CLI struct {
	running bool
	root    *commands.Node
	prompt  string
	lastKey rune
}

func NewCLI() *CLI {
	cli := &CLI{prompt: "ospfc# "}

	cli.MustRegister("exit", "Exit the CLI", func(w io.Writer) error {
		cli.running = false
		return nil
	})

	cli.MustRegister("quit", "Exit the CLI", func(w io.Writer) error {
		cli.running = false
		return nil
	})

	return cli
}

func (cli *CLI) runLine(line string, w io.Writer) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	matches := cli.root.Match(line)
	completeMatches := make([]*commands.Match, 0, len(matches))

	for _, m := range matches {
		if m.IsComplete() {
			completeMatches = append(completeMatches, m)
		}
	}

	if len(completeMatches) == 0 && len(matches) == 0 {
		fmt.Fprintf(w, "%% Unknown command: %s\n", line)
		return
	} else if len(completeMatches) == 0 {
		fmt.Fprintf(w, "%% Command incomplete: %s\n", line)
		return
	} else if len(completeMatches) > 1 {
		fmt.Fprintf(w, "%% Ambiguous command: %s\n", line)
		return
	}

	invoker, err := matches[0].Invoker()
	if err != nil {
		fmt.Fprintf(w, "%% Error running command: %v\n", err)
		return
	}

	err = invoker.Run(w)
	if err != nil {
		fmt.Fprintf(w, "%% Error running command: %v\n", err)
		return
	}
}

func (cli *CLI) autocompleteWithTab(w io.Writer, line string, pos int) (newLine string, newPos int, ok bool) {
	prefix := line[:pos]
	rest := line[pos:]

	options, offset, err := cli.root.GetAutocompleteOptions(w, prefix)
	if err != nil {
		fmt.Fprintf(w, "%s%s\n", cli.prompt, line)
		fmt.Fprintf(w, "%% Error getting autocomplete options: %v\n", err)
		return "", 0, false
	}

	if len(options) == 0 {
		fmt.Fprintf(w, "\a")
		return "", 0, false
	} else if len(options) == 1 {
		new := prefix + options[0][offset:]

		if !strings.HasPrefix(rest, " ") {
			new += " "
		}

		return new + rest, len(new), true
	} else if cli.lastKey != '\t' {
		prefixLen := commonPrefixLen(options...)
		new := prefix + options[0][offset:prefixLen]

		fmt.Fprintf(w, "\a")

		return new + rest, len(new), true
	} else {
		fmt.Fprintf(w, "%s%s\n", cli.prompt, line)

		// TODO: tabulate output based on width of terminal
		for _, o := range options {
			fmt.Fprintf(w, "%s\n", o)
		}

		return "", 0, false
	}
}

type NodeSlice []*commands.Node

func (ns NodeSlice) Len() int {
	return len(ns)
}

func (ns NodeSlice) Less(i, j int) bool {
	return ns[i].String() < ns[j].String()
}

func (ns NodeSlice) Swap(i, j int) {
	ns[i], ns[j] = ns[j], ns[i]
}

func (cli *CLI) autocompleteWithQuestionMark(w io.Writer, line string, pos int) (newLine string, newPos int, ok bool) {
	nodes, err := cli.root.GetAutocompleteNodes(line)
	if err != nil {
		fmt.Fprintf(w, "%s%s\n", cli.prompt, line)
		fmt.Fprintf(w, "%% Error getting autocomplete nodes: %v\n", err)
		return "", 0, false
	}

	if len(nodes) == 0 {
		fmt.Fprintf(w, "%% There is no matched command.\n")
		return "", 0, false
	}

	longestTokenLen := 0
	for _, n := range nodes {
		if len(n.String()) > longestTokenLen {
			longestTokenLen = len(n.String())
		}
	}

	sort.Sort(NodeSlice(nodes))

	fmt.Fprintf(w, "%s%s\n", cli.prompt, line)

	for _, n := range nodes {
		fmt.Fprintf(w, "  %-*s  %s\n", longestTokenLen, n.String(), n.Description())
	}

	return "", 0, false
}

func (cli *CLI) autocomplete(w io.Writer, line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	defer func() {
		cli.lastKey = key
	}()

	if key == '\t' {
		return cli.autocompleteWithTab(w, line, pos)
	} else if key == '?' {
		return cli.autocompleteWithQuestionMark(w, line, pos)
	}

	return "", 0, false
}

func (cli *CLI) Run(rw io.ReadWriter) {
	t := term.NewTerminal(rw, cli.prompt)

	t.AutoCompleteCallback = func(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
		return cli.autocomplete(t, line, pos, key)
	}

	cli.running = true

	for cli.running {
		line, err := t.ReadLine()
		if err != nil {
			fmt.Fprintf(t, "%% Error reading line: %v\n", err)
			break
		}

		cli.runLine(line, t)
	}

	t.AutoCompleteCallback = nil
}

func (cli *CLI) Register(command string, description string, handlerFunc any) error {
	n, err := commands.ParseDeclaration(command)
	if err != nil {
		return err
	}

	for _, l := range n.Leaves() {
		err := l.SetHandlerFunc(handlerFunc)
		if err != nil {
			return err
		}
	}

	newRoot, err := cli.root.Merge(n)
	if err != nil {
		return err
	}

	cli.root = newRoot

	return nil
}

func (cli *CLI) MustRegister(command string, description string, handlerFunc any) {
	err := cli.Register(command, description, handlerFunc)
	if err != nil {
		panic(err)
	}
}

func (cli *CLI) Document(command string, description string) error {
	n, err := commands.ParseDeclaration(command)
	if err != nil {
		return err
	}

	for _, l := range n.Leaves() {
		l.SetDescription(description)
	}

	newRoot, err := cli.root.MergeWithoutExplicitChoiceRestrictions(n)
	if err != nil {
		return err
	}

	cli.root = newRoot

	return nil
}

func (cli *CLI) MustDocument(command string, description string) {
	err := cli.Document(command, description)
	if err != nil {
		panic(err)
	}
}

func (cli *CLI) RegisterAutocomplete(command string, autocompleteFunc commands.AutocompleteFunc) error {
	n, err := commands.ParseDeclaration(command)
	if err != nil {
		return err
	}

	for _, l := range n.Leaves() {
		l.SetAutocompleteFunc(autocompleteFunc)
	}

	newRoot, err := cli.root.MergeWithoutExplicitChoiceRestrictions(n)
	if err != nil {
		return err
	}

	cli.root = newRoot

	return nil
}

func (cli *CLI) MustRegisterAutocomplete(command string, autocompleteFunc commands.AutocompleteFunc) {
	err := cli.RegisterAutocomplete(command, autocompleteFunc)
	if err != nil {
		panic(err)
	}
}
