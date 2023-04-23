package netmon

import (
	"context"

	"github.com/davidbalbert/chatter/chatterd/services"
	"github.com/davidbalbert/chatter/sync"
)

type platformMonitor interface {
	run(context.Context, *Monitor) error
}

type Monitor struct {
	*sync.SimpleNotifier
	p platformMonitor
}

func New(serviceManager *services.ServiceManager, conf any) (services.Runner, error) {
	return &Monitor{
		SimpleNotifier: sync.NewSimpleNotifier(),
		p:              newPlatformMonitor(),
	}, nil
}

func (m *Monitor) Run(ctx context.Context) error {
	return m.p.run(ctx, m)
}
