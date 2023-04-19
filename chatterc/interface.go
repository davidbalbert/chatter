package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"sort"

	"github.com/davidbalbert/chatter/api"
	"github.com/davidbalbert/chatter/system"
)

type interfaceSlice []system.Interface

func (s interfaceSlice) Len() int {
	return len(s)
}

func (s interfaceSlice) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

func (s interfaceSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func registerInterfaceCommands(ctx context.Context, cli *CLI, client *api.Client) {
	cli.MustRegister("show interfaces", "Interface status and configuration", func(w io.Writer) error {
		interfaces, err := client.GetInterfaces(ctx)
		if err != nil {
			return err
		}

		sort.Sort(interfaceSlice(interfaces))

		table, err := tabulate(interfaces, []string{"Name", "State", "MTU"}, func(iface system.Interface) []string {
			state := "down"
			if iface.Flags&net.FlagUp != 0 {
				state = "up"
			}

			return []string{
				iface.Name,
				state,
				fmt.Sprintf("%d", iface.MTU),
			}
		})
		if err != nil {
			return err
		}

		for _, row := range table {
			fmt.Fprintf(w, "%s\n", row)
		}

		return nil

	})
}
