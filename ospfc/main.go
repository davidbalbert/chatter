package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/davidbalbert/ospfd/api"
	"github.com/davidbalbert/ospfd/rpc"
)

type node struct {
	help     string
	execute  func() string
	children map[string]*node
}

func namespace(help string) *node {
	return &node{
		help:     help,
		execute:  incompleteCommand,
		children: make(map[string]*node),
	}
}

func command(help string, execute func() string) *node {
	return &node{
		help:     help,
		execute:  execute,
		children: make(map[string]*node),
	}
}

type cli struct {
	root *node
}

func incompleteCommand() string {
	return "Incomplete command"
}

func newCLI() *cli {
	c := &cli{
		root: namespace(""),
	}

	return c
}

func (c *cli) registerNamespace(path, help string) error {
	return c.registerNode(path, namespace(help))
}

func (c *cli) registerCommand(path, help string, execute func() string) error {
	return c.registerNode(path, command(help, execute))
}

func (c *cli) registerNode(path string, n *node) error {
	components := strings.Fields(path)
	ncomps := len(components)

	current := c.root
	for i, name := range components[:ncomps-1] {
		current = current.children[name]
		if current == nil {
			return fmt.Errorf("cannot register \"%s\", missing node at \"%s\"", path, strings.Join(components[:i+1], " "))
		}
	}

	name := components[ncomps-1]
	if current.children[name] != nil {
		return fmt.Errorf("cannot register \"%s\", node already exists", path)
	}

	current.children[name] = n

	return nil
}

func (c *cli) eval(s string) error {
	components := strings.Fields(s)

	current := c.root
	for i, name := range components {
		current = current.children[name]
		if current == nil {
			return fmt.Errorf("unknown command \"%s\"", strings.Join(components[:i+1], " "))
		}
	}

	fmt.Println(current.execute())

	return nil
}

func readLine() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
}

func main() {
	client, err := api.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to dial: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := newCLI()

	if err := c.registerNamespace("show", "Show running system information"); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if err := c.registerNamespace("show rand", "Show random information"); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	err = c.registerCommand("show rand int", "Show a random integer", func() string {
		resp, err := client.GetRandInt(ctx, &rpc.Empty{})
		if err != nil {
			return fmt.Sprintf("failed to show random int: %v", err)
		}

		return fmt.Sprintf("%d", resp.Value)
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	err = c.registerCommand("show rand string", "Show a random string", func() string {
		resp, err := client.GetRandString(ctx, &rpc.Empty{})
		if err != nil {
			return fmt.Sprintf("failed to show random string: %v", err)
		}

		return resp.Value
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	stdin := make(chan string)

	go func() {
		for {
			line, err := readLine()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}

				fmt.Println(err)
				cancel()
				return
			}

			if line == "" {
				continue
			}

			stdin <- line
		}
	}()

	for {
		fmt.Print("ospfd# ")
		select {
		case line := <-stdin:
			if line == "exit" {
				cancel()
				return
			}

			err := c.eval(line)
			if err != nil {
				fmt.Println(err)
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		case <-ctx.Done():
			return
		}
	}

}
