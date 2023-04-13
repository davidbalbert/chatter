package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/davidbalbert/chatter/api"
	"github.com/davidbalbert/chatter/rpc"
	"golang.org/x/term"
)

func main() {
	ctx := context.Background()

	client, err := api.NewClient()
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}

	cli := NewCLI()

	cli.MustDocument("show", "Show running system information")
	cli.MustRegister("show version", "Show version", func(w io.Writer) error {
		v, err := client.GetVersion(ctx, &rpc.GetVersionRequest{})
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "v%s\n", v.Version)

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
