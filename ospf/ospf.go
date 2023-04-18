package ospf

import (
	"context"
	"net/netip"

	"github.com/davidbalbert/chatter/chatterd/common"
	"github.com/davidbalbert/chatter/chatterd/services"
	"golang.org/x/sync/errgroup"
)

var (
	AllSPFRouters = netip.MustParseAddr("224.0.0.5")
	AllDRouters   = netip.MustParseAddr("224.0.0.6")
)

type Instance struct {
	RouterID common.RouterID
	Areas    map[common.AreaID]*Area
	// TODO: VirtualLinks
	// TODO: ExternalRoutes
	// TODO: LSDB (or maybe just AS external?)
	// TODO: RIB

	serviceManager *services.ServiceManager
}

func NewInstance(serviceManager *services.ServiceManager) (services.Runner, error) {
	backboneID := common.AreaID(0)
	backbone := newArea(backboneID)

	return &Instance{
		Areas: map[common.AreaID]*Area{
			backboneID: backbone,
		},

		serviceManager: serviceManager,
	}, nil
}

func (p *Instance) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	for _, area := range p.Areas {
		area := area
		g.Go(func() error {
			return area.run(ctx)
		})
	}

	return g.Wait()
}
