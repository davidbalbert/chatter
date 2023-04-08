package main

import (
	"github.com/davidbalbert/ospfd/ospfc/commands"
)

type CLI struct {
	root *commands.Node
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
