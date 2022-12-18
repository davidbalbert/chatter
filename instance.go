package main

import (
	"fmt"
	"net"
	"net/netip"
)

type Instance struct {
	routerID   netip.Addr
	interfaces []*Interface
	areas      map[netip.Addr]*area
	db         *lsdb
}

func NewInstance(c *Config) (*Instance, error) {
	inst := &Instance{
		routerID: c.RouterID,
		areas:    make(map[netip.Addr]*area),
		db:       newLSDB(),
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
					_, ok := inst.areas[netconfig.AreaID]
					if !ok {
						area, err := newArea(inst, netconfig.AreaID, false)
						if err != nil {
							return nil, err
						}

						inst.areas[netconfig.AreaID] = area
					}

					iface := NewInterface(inst, addr, netif, &ifconfig, &netconfig)
					inst.interfaces = append(inst.interfaces, iface)
				}
			}
		}
	}

	for _, area := range inst.areas {
		lsa, err := newRouterLSA(inst, area)
		fmt.Printf("new router lsa: %v\n", lsa)
		if err != nil {
			return nil, err
		}

		inst.db.set(area.id, lsa)
	}

	return inst, nil
}

func (inst *Instance) Run() {
	for _, iface := range inst.interfaces {
		// TODO, maybe make run spawn a goroutine and return?
		go iface.run()
	}
}
