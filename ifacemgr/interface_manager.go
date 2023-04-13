package ifacemgr

import (
	"context"
	"net"
)

type Interface struct {
	net.Interface
}

type InterfaceManager struct {
}

func (m *InterfaceManager) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (m *InterfaceManager) GetInterfaces() ([]Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	interfaces := make([]Interface, len(ifaces))
	for i, iface := range ifaces {
		interfaces[i] = Interface{iface}
	}

	return interfaces, nil
}
