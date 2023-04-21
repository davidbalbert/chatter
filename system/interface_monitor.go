package system

import (
	"context"
	"fmt"

	"github.com/davidbalbert/chatter/events"
)

// TODO: it's possible to miss events if it takes you too long to call wait again.
// we should use the Notifier from the "Rethinking Classical Concurrency Patterns"
// slides.
type InterfaceMonitor interface {
	Run(context.Context) error
	Wait(context.Context)
}

type baseInterfaceMonitor struct {
	events chan chan struct{}
}

func newBaseInterfaceMonitor() *baseInterfaceMonitor {
	events := make(chan chan struct{}, 1)
	events <- make(chan struct{})

	return &baseInterfaceMonitor{
		events: events,
	}
}

func (m *baseInterfaceMonitor) notify() error {
	e := <-m.events
	close(e)
	m.events <- make(chan struct{})

	return nil
}

func (m *baseInterfaceMonitor) Wait(ctx context.Context) {
	c := <-m.events
	m.events <- c

	select {
	case <-ctx.Done():
		return
	case <-c:
	}
}

func (m *baseInterfaceMonitor) SendEvent(e events.Event) error {
	return fmt.Errorf("interface monitor does not receive events")
}
