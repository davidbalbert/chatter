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

func NewInstance(serviceManager *services.ServiceManager, conf any) (services.Service, error) {
	if conf == nil {
		return nil, fmt.Errorf("no ospf config provided")
	}

	ospfConf, ok := conf.(*config.OSPFConfig)
	if !ok {
		return nil, fmt.Errorf("expected *config.OSPFConfig, but got %T", conf)
	}

	areas := make(map[common.AreaID]*Area)

	for id, areaConf := range ospfConf.Areas {
		area, err := newArea(id, areaConf)
		if err != nil {
			return nil, err
		}
		areas[id] = area
	}

	return &Instance{
		RouterID: ospfConf.RouterID,
		Areas:    areas,

		serviceManager: serviceManager,
		events:         make(chan config.Event),
		config:         ospfConf,
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
