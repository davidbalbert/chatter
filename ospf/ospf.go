package ospf

import (
	"context"
	"fmt"
	"net"
	"net/netip"

	"github.com/davidbalbert/chatter/chatterd/common"
	"github.com/davidbalbert/chatter/chatterd/services"
	"github.com/davidbalbert/chatter/config"
	"github.com/davidbalbert/chatter/events"
	"github.com/davidbalbert/chatter/system"
	"go4.org/netipx"
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

	Interfaces map[interfaceID]*Interface

	serviceManager *services.ServiceManager
	events         chan events.Event
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
		areas[id] = newArea(id, areaConf)
	}

	return &Instance{
		RouterID: ospfConf.RouterID,
		Areas:    areas,

		Interfaces: make(map[interfaceID]*Interface),

		serviceManager: serviceManager,
		events:         make(chan events.Event),
		config:         ospfConf,
	}, nil
}

func (i *Instance) SendEvent(e events.Event) error {
	i.events <- e

	return nil
}

func (i *Instance) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	s, err := i.serviceManager.Get(config.ServiceInterfaceMonitor)
	if err != nil {
		return fmt.Errorf("failed to get interface monitor service: %w", err)
	}

	interfaceMonitor, ok := s.(system.InterfaceMonitor)
	if !ok {
		return fmt.Errorf("expected system.InterfaceMonitor but got %v", s)
	}

	intCh := make(chan struct{}, 1)
	intCh <- struct{}{}

	g.Go(func() error {
		for {
			// TODO: I do need to make sure I don't miss any events. If two events come in quick succession
			// and I miss the second one, I'll never get it, so I might have out of date interface info for
			// a while.
			//
			// Two options:
			// 1. interfaceMonitor.Subscribe(ctx, intCh)
			// 2. seq := interfaceMonitor.Seq()
			//    for {
			//    	seq := interfaceMonitor.Wait(ctx, seq)
			//      // send on intCh
			//    }
			interfaceMonitor.Wait(ctx)
			select {
			case <-ctx.Done():
				return nil
			case intCh <- struct{}{}:
			}
		}
	})

	for {
		select {
		case <-ctx.Done():
			return g.Wait()
		case e := <-i.events:
			fmt.Printf("got event: %s\n", e.Type)
		case <-intCh:
			err := i.updateInterfaces()
			if err != nil {
				return err
			}
		}
	}
}

type netifAndPrefixes struct {
	netif    net.Interface
	prefixes []netip.Prefix
}

func (i *Instance) updateInterfaces() error {
	netifs, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get interfaces: %w", err)
	}

	nameToNetif := make(map[string]netifAndPrefixes)

	for _, netif := range netifs {

		prefixes, err := netifPrefixesV4(netif)
		if err != nil {
			return err
		}

		nameToNetif[netif.Name] = netifAndPrefixes{netif, prefixes}
	}

	// update existing interfaces and remove any that are gone
	for id, iface := range i.Interfaces {
		nap, ok := nameToNetif[id.name]
		if !ok { // interface was removed
			iface.handleEvent(ieInterfaceDown)
			delete(i.Interfaces, id)
			continue
		}

		netif, prefixes := nap.netif, nap.prefixes

		found := false
		for _, prefix := range prefixes {
			if prefixEquals(prefix, iface.Prefix) {
				found = true
				break
			}
		}

		// address was removed
		if !found {
			iface.handleEvent(ieInterfaceDown)
			delete(i.Interfaces, id)
			continue
		}

		// detect changes to interface state
		if isUp(netif) && !iface.isUp() {
			iface.handleEvent(ieInterfaceUp)
		} else if !isUp(netif) && iface.isUp() {
			iface.handleEvent(ieInterfaceDown)
		}

		if isLoopback(netif) && !iface.isLoopback() {
			iface.handleEvent(ieLoopInd)
		} else if !isLoopback(netif) && iface.isLoopback() {
			iface.handleEvent(ieUnloopInd)
		}
	}

	// add new interfaces
	configs := i.config.InterfaceConfigs()
	for name, conf := range configs {
		nap, ok := nameToNetif[name]
		if !ok {
			continue
		}

		netif, prefixes := nap.netif, nap.prefixes

		for _, prefix := range prefixes {
			id := interfaceID{name: name, prefix: prefix}

			if _, ok := i.Interfaces[id]; ok {
				continue
			}

			i.Interfaces[id] = newInterface(conf, conf.AreaID, name, prefix)

			if isUp(netif) {
				i.Interfaces[id].handleEvent(ieInterfaceUp)
			}

			if isLoopback(netif) {
				i.Interfaces[id].handleEvent(ieLoopInd)
			}
		}
	}

	return nil
}

func netifPrefixesV4(netif net.Interface) ([]netip.Prefix, error) {
	addrs, err := netif.Addrs()
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses for interface %s: %w", netif.Name, err)
	}

	var prefixes []netip.Prefix
	for _, addr := range addrs {
		prefix, ok := prefixFromSTDNetAddr(addr)
		if ok && prefix.Addr().Is4() {
			prefixes = append(prefixes, prefix)
		}
	}

	return prefixes, nil
}

// net.Interface.Addrs() returns []net.Addr which is really
// []*net.IPNet.
func prefixFromSTDNetAddr(addr net.Addr) (netip.Prefix, bool) {
	ipnet, ok := addr.(*net.IPNet)
	if !ok {
		return netip.Prefix{}, false
	}

	prefix, ok := netipx.FromStdIPNet(ipnet)
	if !ok {
		return netip.Prefix{}, false
	}

	return prefix, true
}

func isUp(netif net.Interface) bool {
	return netif.Flags&net.FlagUp != 0
}

func isLoopback(netif net.Interface) bool {
	return netif.Flags&net.FlagLoopback != 0
}

func prefixEquals(a, b netip.Prefix) bool {
	return a.Addr().Compare(b.Addr()) == 0 && a.Bits() == b.Bits()
}
