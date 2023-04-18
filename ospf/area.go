package ospf

import (
	"context"
	"net/netip"

	"github.com/davidbalbert/chatter/chatterd/common"
	"golang.org/x/sync/errgroup"
)

type AddressRange struct {
	Prefix    netip.Prefix
	Advertise bool
}

type Area struct {
	ID            common.AreaID
	AddressRanges []AddressRange
	Interfaces    map[string]*Interface
	// TODO: LSDB
	// TODO: ShortestPathTree
	TransitCapability         bool // calculated when ShortestPathTree is calculated
	ExternalRoutingCapability bool
	// TODO: StubDefaultCost
}

func newArea(id common.AreaID) *Area {
	return &Area{
		ID:         id,
		Interfaces: make(map[string]*Interface),
	}
}

func (a *Area) run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, iface := range a.Interfaces {
		iface := iface
		g.Go(func() error {
			return iface.run(ctx)
		})
	}

	return g.Wait()
}
