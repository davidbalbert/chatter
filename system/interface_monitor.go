package system

import (
	"context"
	"fmt"
	"net"
)

type Interface struct {
	net.Interface
}

type InterfaceEvent struct {
	detail string
}

type EventListener interface {
	OnEvents(event []InterfaceEvent)
}

type InterfaceMonitor interface {
	Run(ctx context.Context) error
	Subscribe(listener EventListener)
	Unsubscribe(listener EventListener)
	Interfaces() ([]Interface, error)
}

type baseInterfaceMonitor struct {
	interfaces []Interface
	listeners  []EventListener
}

func (m *baseInterfaceMonitor) Subscribe(listener EventListener) {
	m.listeners = append(m.listeners, listener)
}

func (m *baseInterfaceMonitor) Unsubscribe(listener EventListener) {
	for i, l := range m.listeners {
		if l == listener {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			return
		}
	}
}

func (m *baseInterfaceMonitor) notify(events []InterfaceEvent) {
	fmt.Printf("notify: %v\n", events)

	for _, listener := range m.listeners {
		listener.OnEvents(events)
	}
}

func (m *baseInterfaceMonitor) getInterfaces() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		m.interfaces = nil
		return err
	}

	interfaces := make([]Interface, len(ifaces))
	for i, iface := range ifaces {
		interfaces[i] = Interface{iface}
	}

	m.interfaces = interfaces

	return nil
}

func (m *baseInterfaceMonitor) Interfaces() ([]Interface, error) {
	if m.interfaces == nil {
		err := m.getInterfaces()
		if err != nil {
			return nil, err
		}
	}

	return m.interfaces, nil
}
