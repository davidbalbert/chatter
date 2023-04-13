package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func main() {
	cli := NewCLI()

	cli.MustDocument("show", "Show running system information")
	cli.MustRegister("show version", "Show Chatter version", func(w io.Writer) error {
		fmt.Fprintln(w, "v0.0.1")

		return nil
	})

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("Failed to make terminal raw: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	cli.Run(os.Stdin)
}
