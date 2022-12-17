package main

import (
	"fmt"
	"net"
	"net/netip"
)

type Instance struct {
	RouterID netip.Addr
	Areas    map[netip.Addr]*area
}

func NewInstance(c *Config) (*Instance, error) {
	inst := &Instance{
		RouterID: c.RouterID,
		Areas:    make(map[netip.Addr]*area),
	}

	for _, ifconfig := range c.Interfaces {
		netif, err := net.InterfaceByName(ifconfig.Name)
		if err != nil {
			return nil, err
		}

		addrs, err := netif.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			addr, err := netip.ParsePrefix(addr.String())
			if err != nil {
				return nil, fmt.Errorf("error parsing prefix for %s: %w", netif.Name, err)
			}

			for _, netconfig := range c.Networks {
				if addr.Masked() == netconfig.Network {
					area, ok := inst.Areas[netconfig.AreaID]
					if !ok {
						area, err = newArea(inst, netconfig.AreaID, false)
						if err != nil {
							return nil, err
						}

						inst.Areas[netconfig.AreaID] = area
					}

					iface := NewInterface(inst, addr, netif, &ifconfig, &netconfig)

					area.interfaces = append(area.interfaces, iface)
				}
			}
		}
	}

	return inst, nil
}

func (inst *Instance) Run() {
	for _, area := range inst.Areas {
		for _, iface := range area.interfaces {
			// TODO, maybe make run spawn a goroutine and return?
			go iface.run()
		}
	}
}
