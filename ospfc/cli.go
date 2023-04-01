package main

import (
	"fmt"
	"reflect"

	"github.com/davidbalbert/ospfd/ospfc/commands"
)

type CLI struct {
	graph commands.Graph
}

func (cli *CLI) MustRegister(command string, description string, handlerFunc any) {
	g, parsedType, err := commands.ParseDeclaration(command)
	if err != nil {
		panic(err)
	}

	handler := reflect.ValueOf(handlerFunc)
	givenType := handler.Type()

	if parsedType != givenType {
		s := fmt.Sprintf("command %q expects %s, but handler is %s", command, parsedType, givenType)
		panic(s)
	}

	cli.graph = cli.graph.Merge(g)

}

func (cli *CLI) MustDocument(command string, description string) {
	g, _, err := commands.ParseDeclaration(command)
	if err != nil {
		panic(err)
	}

	g.SetDescription(description)

	cli.graph = cli.graph.Merge(g)
}

func (cli *CLI) MustAutocomplete(command string, autocompleteFunc func(string) ([]string, error)) {
	g, _, err := commands.ParseDeclaration(command)
	if err != nil {
		panic(err)
	}

	g.SetAutocompleteFunc(autocompleteFunc)

	cli.graph = cli.graph.Merge(g)
}
