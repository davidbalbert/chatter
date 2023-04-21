package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"sort"
	"strings"

	"github.com/davidbalbert/chatter/api"
	"github.com/davidbalbert/chatter/rpc"
)

type interfaceSlice []*rpc.Interface

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

		table, err := tabulate(interfaces, []string{"Name", "State", "MTU", "Addresses"}, false, func(iface *rpc.Interface) ([]string, error) {
			state := "Down"
			if net.Flags(iface.Flags)&net.FlagUp != 0 {
				state = "Up"
			}

			addrs := make([]string, len(iface.Addrs))
			for i, p := range iface.Addrs {
				addr, ok := netip.AddrFromSlice(p.Addr)
				if !ok {
					return nil, fmt.Errorf("invalid IP address: %v", p.Addr)
				}

				prefix := netip.PrefixFrom(addr, int(p.PrefixLen))
				addrs[i] = prefix.String()
			}

			return []string{
				iface.Name,
				state,
				fmt.Sprintf("%d", iface.Mtu),
				strings.Join(addrs, "\n"),
			}, nil
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
