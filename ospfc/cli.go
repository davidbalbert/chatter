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

func (cli *CLI) Run(t *term.Terminal) {
	cli.running = true

	for cli.running {
		line, err := t.ReadLine()
		if err != nil {
			fmt.Fprintf(t, "%% Error reading line: %v\n", err)
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := cli.root.Match(line)
		completeMatches := make([]*commands.Match, 0, len(matches))

		for _, m := range matches {
			if m.IsComplete() {
				completeMatches = append(completeMatches, m)
			}
		}

		if len(completeMatches) == 0 && len(matches) == 0 {
			fmt.Fprintf(t, "%% Unknown command: %s\n", line)
			continue
		} else if len(completeMatches) == 0 {
			fmt.Fprintf(t, "%% Command incomplete: %s\n", line)
			continue
		} else if len(completeMatches) > 1 {
			fmt.Fprintf(t, "%% Ambiguous command: %s\n", line)
			continue
		}

		invoker, err := matches[0].Invoker()
		if err != nil {
			fmt.Fprintf(t, "%% Error running command: %v\n", err)
			continue
		}

		err = invoker.Run(t)
		if err != nil {
			fmt.Fprintf(t, "%% Error running command: %v\n", err)
			continue
		}
	}
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
