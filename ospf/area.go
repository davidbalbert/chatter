package ospf

import (
	"net/netip"

	"github.com/davidbalbert/chatter/chatterd/common"
	"github.com/davidbalbert/chatter/config"
)

type AddressRange struct {
	Prefix    netip.Prefix
	Advertise bool
}

type Area struct {
	ID            common.AreaID
	AddressRanges []AddressRange
	// Interfaces are stored in Instance.Interfaces

	// TODO: LSDB
	// TODO: ShortestPathTree
	TransitCapability         bool // calculated when ShortestPathTree is calculated
	ExternalRoutingCapability bool
	// TODO: StubDefaultCost
}

func newArea(areaID common.AreaID, conf config.OSPFAreaConfig) *Area {
	return &Area{
		ID: areaID,
	}
}
