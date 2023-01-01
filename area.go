package main

import (
	"fmt"
	"net/netip"
)

var (
	area0 = netip.IPv4Unspecified()
)

type addressRange struct {
	prefix    netip.Prefix
	advertise bool
}

type area struct {
	inst *Instance

	id netip.Addr
	// addressRanges []addressRange
	// routerLSAs    []*routerLSA
	// networkLSAs   []*networkLSA
	// summaryLSAs   []*summaryLSA

	// TODO: shortestPathTree
	// TODO: transitCapability

	externalRoutingCapability bool

	// TODO: stubDefaultCost
}

func newArea(inst *Instance, areaID netip.Addr, stub bool) (*area, error) {
	if areaID == area0 && stub {
		return nil, fmt.Errorf("area 0 cannot be a stub area")
	}

	return &area{
		inst:                      inst,
		id:                        areaID,
		externalRoutingCapability: !stub,
	}, nil
}

func (area *area) isStub() bool {
	return !area.externalRoutingCapability
}

func (area *area) neighbors() []*Neighbor {
	neighbors := make([]*Neighbor, 0)
	for _, iface := range area.inst.interfaces {
		if iface.areaID == area.id {
			for _, neighbor := range iface.neighbors {
				neighbors = append(neighbors, neighbor)
			}
		}
	}

	return neighbors
}
