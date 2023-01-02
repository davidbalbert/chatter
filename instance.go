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

func (inst *Instance) nPartialAdjacent() int {
	n := 0
	for _, iface := range inst.interfaces {
		for _, neigh := range iface.neighbors {
			if neigh.state == nExchange || neigh.state == nLoading {
				n++
			}
		}
	}

	return n
}

func (inst *Instance) eligibleInterfacesFor(lsa lsa, areaID netip.Addr) []*Interface {
	interfaces := make([]*Interface, 0)
	for _, iface := range inst.interfaces {
		if lsa.Type() == lsTypeASExternal {
			area := inst.areas[iface.areaID]

			// TODO: store a pointer to the area in the interface, so this becomes impossible
			if area == nil {
				panic("interface has no area")
			}

			if !area.isStub() && iface.networkType != networkVirtualLink {
				interfaces = append(interfaces, iface)
			}
		} else if iface.areaID == areaID {
			if areaID == area0 || iface.networkType != networkVirtualLink {
				interfaces = append(interfaces, iface)
			}
		}
	}

	return interfaces
}

func (inst *Instance) queueFlood(lsa lsa, from *Neighbor) bool {
	ifaces := inst.eligibleInterfacesFor(lsa, from.iface.areaID)
	floodedOutReceivingInterface := false

	// Section 13.3
	for _, iface := range ifaces {
		retransmissionScheduled := false

		// 1) Examine each neighbor on the interface.
		for _, n := range iface.neighbors {
			// 1a) Don't flood to neighbors in state less than Exchange
			if n.state < nExchange {
				continue
			}

			existingIndex := n.requestListIndexOf(lsa)

			// 1b) Handle neighbors that are doing database exchange
			if existingIndex != -1 && (n.state == nExchange || n.state == nLoading) {
				existing := &n.linkStateRequestList[existingIndex]

				cmp := lsa.Compare(existing)

				if cmp == -1 {
					continue
				} else if cmp == 0 {
					n.removeFromRequestListAtIndex(existingIndex)
					continue
				} else {
					n.removeFromRequestListAtIndex(existingIndex)
				}
			}

			// 1c) Skip the neighbor that we received the LSA from
			if n == from {
				continue
			}

			// 1d) Add this LSA to the neighbor's retransmission list
			n.linkStateRetransmissionList = append(n.linkStateRetransmissionList, lsa.copyHeader())
			retransmissionScheduled = true
		}

		// 2) If we didn't add the LSA to any neighbor's retransmission list, skip this interface
		if !retransmissionScheduled {
			continue
		}

		if from.iface == iface {
			// 3) If we received the LSA on this interface from a DR or BDR, skip this interface
			if from.isDR() || from.isBDR() {
				continue
			}

			// TODO: uncomment this when we implement the interface state machine
			// 4) If we're the BDR on this interface, skip this interface
			// if iface.state == iBackup {
			// 	continue
			// }

			floodedOutReceivingInterface = true
		}

		iface.floodList = append(iface.floodList, lsa)
	}

	return floodedOutReceivingInterface
}

func (inst *Instance) flood() {
	for _, iface := range inst.interfaces {
		if len(iface.floodList) == 0 {
			continue
		}

		iface.floodLinkStateUpdates()
	}
}
