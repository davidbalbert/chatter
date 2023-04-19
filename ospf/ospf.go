package ospf

import (
	"context"
	"fmt"
	"net/netip"

	"github.com/davidbalbert/chatter/chatterd/common"
	"github.com/davidbalbert/chatter/chatterd/services"
	"github.com/davidbalbert/chatter/config"
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
	events         chan config.Event
	config         *config.OSPFConfig
}

func NewInstance(serviceManager *services.ServiceManager, data any) (services.Service, error) {
	if data == nil {
		return nil, fmt.Errorf("no ospf config provided")
	}

	conf, ok := data.(*config.OSPFConfig)
	if !ok {
		return nil, fmt.Errorf("expected OSPFConfig, but got %T", data)
	}

	backboneID := common.AreaID(0)
	backbone := newArea(backboneID)

	events := make(chan config.Event)

	return &Instance{
		Areas: map[common.AreaID]*Area{
			backboneID: backbone,
		},

		serviceManager: serviceManager,
		events:         events,
		config:         conf,
	}, nil
}

func (i *Instance) SendEvent(e config.Event) error {
	i.events <- e

	return nil
}

func (i *Instance) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case e := <-i.events:
			fmt.Printf("got event: %s\n", e.Type)
		}
	}
}
