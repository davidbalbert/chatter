package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/davidbalbert/ospfd/ospfc/commands"
	"golang.org/x/term"
)

type CLI struct {
	running bool
	root    *commands.Node
}

func NewCLI() *CLI {
	cli := &CLI{}

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

func (cli *CLI) autocomplete(w io.Writer, line string, pos int, key rune) (newLine string, newPos int, ok bool) {
	// For now, only autocomplete at the end of the line
	if pos != len(line) {
		return "", 0, false
	}

	if key != '\t' {
		return "", 0, false
	}

	options, offset, err := cli.root.GetAutocompleteOptions(line)
	if err != nil {
		fmt.Fprintf(w, "%% Error getting autocomplete options: %v\n", err)
		return "", 0, false
	}

	if len(options) == 0 {
		fmt.Fprintf(w, "no options\n")
		return "", 0, false
	} else if len(options) == 1 {
		new := line + options[0][offset:] + " "

		return new, len(new), true
	} else {
		for _, o := range options {
			fmt.Fprintf(w, "%s\n", o)
		}

		return "", 0, false
	}
}

func (cli *CLI) Run(t *term.Terminal) {
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

	newRoot, err := cli.root.Merge(n)
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

func (cli *CLI) Autocomplete(command string, autocompleteFunc commands.AutocompleteFunc) error {
	n, err := commands.ParseDeclaration(command)
	if err != nil {
		return err
	}

	for _, l := range n.Leaves() {
		l.SetAutocompleteFunc(autocompleteFunc)
	}

	newRoot, err := cli.root.Merge(n)
	if err != nil {
		return err
	}

	cli.root = newRoot

	return nil
}

func (cli *CLI) MustAutocomplete(command string, autocompleteFunc func(string) ([]string, error)) {
	err := cli.Autocomplete(command, autocompleteFunc)
	if err != nil {
		panic(err)
	}
}
