package system

import (
	"context"
	"net"
)

type Interface struct {
	net.Interface
}

func getInterfaces() ([]Interface, error) {
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

type InterfaceMonitor interface {
	Run(context.Context) error
	Interfaces() []Interface
	WaitInterfaces(context.Context) []Interface
}

type baseInterfaceMonitor struct {
	events     chan chan struct{}
	interfaces chan []Interface
}

func newBaseInterfaceMonitor() (*baseInterfaceMonitor, error) {
	events := make(chan chan struct{}, 1)
	events <- make(chan struct{})

	interfaces := make(chan []Interface, 1)

	ifaces, err := getInterfaces()
	if err != nil {
		return nil, err
	}

	interfaces <- ifaces

	return &baseInterfaceMonitor{
		interfaces: interfaces,
		events:     events,
	}, nil
}

func (m *baseInterfaceMonitor) notify() error {
	<-m.interfaces

	interfaces, err := getInterfaces()
	if err != nil {
		return err
	}

	m.interfaces <- interfaces

	e := <-m.events
	close(e)
	m.events <- make(chan struct{})

	return nil
}

func (m *baseInterfaceMonitor) WaitInterfaces(ctx context.Context) []Interface {
	c := <-m.events
	m.events <- c

	select {
	case <-ctx.Done():
		return nil
	case <-c:
	}

	return m.Interfaces()
}

func (m *baseInterfaceMonitor) Interfaces() []Interface {
	i1 := <-m.interfaces

	i2 := make([]Interface, len(i1))
	copy(i2, i1)

	m.interfaces <- i1

	return i2
}
