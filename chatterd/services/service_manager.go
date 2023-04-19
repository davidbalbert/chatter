package services

import (
	"context"
	"fmt"

	"github.com/davidbalbert/chatter/config"
	"golang.org/x/sync/errgroup"
)

type Runner interface {
	Run(ctx context.Context) error
}

type BuilderFunc func(*ServiceManager) (Runner, error)

var builders = make(map[config.ServiceType]BuilderFunc)

func registerServiceType(t config.ServiceType, fn BuilderFunc) error {
	_, ok := builders[t]
	if ok {
		return fmt.Errorf("service type already registered: %v", t)
	}

	builders[t] = fn

	return nil
}

func MustRegisterServiceType(t config.ServiceType, fn BuilderFunc) {
	err := registerServiceType(t, fn)
	if err != nil {
		panic(err)
	}
}

type ServiceController struct {
	service any
	cancel  context.CancelFunc
	done    chan struct{}
}

func (c *ServiceController) Stop() {
	c.cancel()
}

func (c *ServiceController) Wait() error {
	<-c.done
	return nil
}

type ServiceManager struct {
	services map[string]ServiceController
}

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		services: make(map[string]ServiceController),
	}
}

func (s *ServiceManager) Run(ctx context.Context, configManager *config.ConfigManager) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case event := <-configManager.Events():
				switch event.Type {
				case config.ConfigUpdated:
					for _, service := range s.services {
						service.Stop()
					}

					for name, service := range s.services {
						service.Wait()
						delete(s.services, name)
					}

					for _, service := range configManager.GetConfig().ServicesInBootOrder() {
						err := s.start(ctx, g, service)
						if err != nil {
							return err
						}
					}
				default:
					return fmt.Errorf("unknown config event type: %v", event.Type)
				}
			}
		}
	})

	return g.Wait()
}

func (s *ServiceManager) start(ctx context.Context, g *errgroup.Group, service config.Service) error {
	_, ok := s.services[service.Name]
	if ok {
		return fmt.Errorf("service already running: %s", service.Name)
	}

	builder, ok := builders[service.Type]
	if !ok {
		return fmt.Errorf("unknown service type: %v", service.Type)
	}

	runner, err := builder(s)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})

	s.services[service.Name] = ServiceController{
		service: runner,
		cancel:  cancel,
		done:    done,
	}

	// TODO: is it kosher to call g.Go() from within g.Go()?
	g.Go(func() error {
		err := runner.Run(ctx)
		close(done)
		return err
	})

	return nil
}

func (s *ServiceManager) Get(service config.Service) (any, error) {
	controller, ok := s.services[service.Name]
	if !ok {
		return nil, fmt.Errorf("service not running: %s", service.Name)
	}

	return controller.service, nil
}
