package system

import (
	"context"
	"fmt"

	"github.com/davidbalbert/chatter/chatterd/services"
	"github.com/davidbalbert/chatter/events"
	"github.com/davidbalbert/chatter/sync"
)

type platformMonitor interface {
	run(context.Context, *InterfaceMonitor) error
}

type InterfaceMonitor struct {
	*sync.SimpleNotifier
	p platformMonitor
}

func NewInterfaceMonitor(serviceManager *services.ServiceManager, conf any) (services.Service, error) {
	return &InterfaceMonitor{
		SimpleNotifier: sync.NewSimpleNotifier(),
		p:              newPlatformMonitor(),
	}, nil
}

func (m *InterfaceMonitor) Run(ctx context.Context) error {
	return m.p.run(ctx, m)
}

func (m *InterfaceMonitor) SendEvent(e events.Event) error {
	return fmt.Errorf("interface monitor does not receive events")
}
