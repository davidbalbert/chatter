package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/davidbalbert/chatter/api"
	"golang.org/x/term"
)

var socketPath string

func main() {
	flag.StringVar(&socketPath, "socket", "/var/run/chatterd.sock", "path to chatterd socket")

	flag.Parse()

	ctx := context.Background()

	client, err := api.NewClient(socketPath)
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	cli := NewCLI()

	cli.MustDocument("show", "Show running system information")

	cli.MustRegister("show version", "Show version", func(w io.Writer) error {
		version, err := client.GetVersion(ctx)
		if err != nil {
			return err
		}

		fmt.Fprintf(w, "v%s\n", version)

		return nil
	})

	cli.MustRegister("shutdown", "Shutdown chatterd", func(w io.Writer) error {
		err := client.Shutdown(ctx)
		if err != nil {
			return err
		}

		cli.running = false

		return nil
	})

	registerInterfaceCommands(ctx, cli, client)

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Printf("Failed to make terminal raw: %v\n", err)
		os.Exit(1)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	cli.Run(os.Stdin)
}
