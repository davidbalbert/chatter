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

	areaID        netip.Addr
	addressRanges []addressRange
	routerLSAs    []*routerLSA
	networkLSAs   []*networkLSA
	summaryLSAs   []*summaryLSA

	// TODO: shortestPathTree
	// TODO: transitCapability

	externalRoutingCapability bool

	// TODO: stubDefaultCost

	interfaces []*Interface
}

func newArea(inst *Instance, areaID netip.Addr, stub bool) (*area, error) {
	if areaID == area0 && stub {
		return nil, fmt.Errorf("area 0 cannot be a stub area")
	}

	return &area{
		inst:                      inst,
		areaID:                    areaID,
		externalRoutingCapability: !stub,
	}, nil
}

func (area *area) isStub() bool {
	return !area.externalRoutingCapability
}

func (area *area) addInterface(iface *Interface) {
	area.interfaces = append(area.interfaces, iface)
}
