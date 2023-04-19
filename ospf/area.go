package ospf

import (
	"net"
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
	Interfaces    []*Interface
	// TODO: LSDB
	// TODO: ShortestPathTree
	TransitCapability         bool // calculated when ShortestPathTree is calculated
	ExternalRoutingCapability bool
	// TODO: StubDefaultCost
}

func newArea(id common.AreaID, conf config.OSPFAreaConfig) (*Area, error) {
	netifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// net.InterfaceByName would be better but there's no way to tell whether
	// it returned an error because the interface doesn't exist or for some
	// other reason.
	netifsByName := make(map[string]*net.Interface)
	for _, netif := range netifs {
		netifsByName[netif.Name] = &netif
	}

	var interfaces []*Interface

	for name, ifaceConf := range conf.Interfaces {
		netif := netifsByName[name]
		if netif == nil {
			continue
		}

		addrs, err := netif.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			prefix, err := netip.ParsePrefix(addr.String())
			if err != nil {
				return nil, err
			}

			iface, err := newInterface(ifaceConf, id, prefix)
			if err != nil {
				return nil, err
			}

			interfaces = append(interfaces, iface)
		}
	}

	return &Area{
		ID:         id,
		Interfaces: interfaces,
	}, nil
}
