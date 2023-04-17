package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"golang.org/x/sync/errgroup"
)

type serviceMessageType int

const (
	smStart serviceMessageType = iota
	smStop
)

type ServiceMessage struct {
	Type        serviceMessageType
	ServiceType string
	Name        string
}

type ServiceManager struct {
	services map[string]context.CancelFunc
	ctl      chan ServiceMessage
}

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		services: make(map[string]context.CancelFunc),
		ctl:      make(chan ServiceMessage),
	}
}

func NewInterfaceMonitor(m *ServiceManager) Runner {
	return nil
}

func NewOSPF(m *ServiceManager) Runner {
	return nil
}

type Runner interface {
	Run(ctx context.Context) error
}

type BuilderFunc func(*ServiceManager) Runner

var builders = make(map[string]BuilderFunc)

func RegisterServiceType(t string, fn BuilderFunc) error {
	_, ok := builders[t]
	if ok {
		return fmt.Errorf("service type already registered: %v", t)
	}

	builders[t] = fn

	return nil
}

func MustRegisterServiceType(t string, constructor func(*ServiceManager) Runner) {
	err := RegisterServiceType(t, constructor)
	if err != nil {
		panic(err)
	}
}

func (m *ServiceManager) buildService(t string) (Runner, error) {
	constructor := builders[t]

	if constructor == nil {
		return nil, fmt.Errorf("unknown service type: %v", t)
	}

	return constructor(m), nil
}

func (m *ServiceManager) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case msg := <-m.ctl:
				switch msg.Type {
				case smStart:
					_, ok := m.services[msg.Name]
					if ok {
						return fmt.Errorf("service already running: %q", msg.Name)
					}

					service, err := m.buildService(msg.ServiceType)
					if err != nil {
						return err
					}

					ctx, cancel := context.WithCancel(ctx)
					m.services[msg.Name] = cancel

					// TODO: is it kosher to call g.Go() from within a g.Go()?
					g.Go(func() error {
						return service.Run(ctx)
					})
				case smStop:
					stopService, ok := m.services[msg.Name]
					if !ok {
						return fmt.Errorf("unknown service: %q", msg.Name)
					}

					stopService()
				default:
					return fmt.Errorf("unknown service message type: %v", msg.Type)
				}
			}
		}
	})

	return g.Wait()
}

func sketch() {
	MustRegisterServiceType("InterfaceMonitor", NewInterfaceMonitor)
	MustRegisterServiceType("OSPF", NewOSPF)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	serviceManager := NewServiceManager()

	err := serviceManager.Run(ctx)

	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
