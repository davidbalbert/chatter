package main

import (
	"context"
	"fmt"
	"io"

	"github.com/davidbalbert/chatter/api"
)

func registerInterfaceCommands(ctx context.Context, cli *CLI, client *api.Client) {
	cli.MustRegister("show interfaces", "Interface status and configuration", func(w io.Writer) error {
		interfaces, err := client.GetInterfaces(ctx)
		if err != nil {
			return err
		}

		for _, iface := range interfaces {
			fmt.Fprintf(w, "%s\n", iface.Name)
		}

		return nil

	})
}
