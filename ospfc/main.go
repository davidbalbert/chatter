package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/davidbalbert/ospfd/vty"
	"github.com/peterh/liner"
)

var commands = []string{"show version", "show interfaces", "exit"}

func autocomplete(line string) []string {
	options := []string{}

	for _, cmd := range commands {
		if strings.HasPrefix(cmd, line) {
			options = append(options, cmd)
		}
	}

	return options
}

func main() {
	client, err := vty.Dial()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer client.Close()

	line := liner.NewLiner()
	defer line.Close()

	if f, err := os.Open(".ospfc_history"); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	line.SetCompleter(autocomplete)
	line.SetTabCompletionStyle(liner.TabPrints)

	for {
		cmd, err := line.Prompt("ospfd# ")
		if err != nil {
			break
		}

		if cmd == "" {
			continue
		}

		line.AppendHistory(cmd)

		if strings.TrimSpace(cmd) == "exit" {
			break
		}

		resp, err := client.Send(cmd)
		if err != nil {
			fmt.Println(err)
			continue
		}

		// Server only sends an empty response when the connection is closed.
		if resp == "" {
			break
		}

		fmt.Println(resp)
	}

	if f, err := os.Create(".ospfc_history"); err != nil {
		fmt.Println(err)
	} else {
		line.WriteHistory(f)
		f.Close()
	}
}
