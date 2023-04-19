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
	id      config.ServiceID
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

type state struct {
	controllers map[string]ServiceController
}

type ServiceManager struct {
	c             chan state
	configManager *config.ConfigManager
}

func NewServiceManager(configManager *config.ConfigManager) *ServiceManager {
	st := state{
		controllers: make(map[string]ServiceController),
	}

	c := make(chan state, 1)
	c <- st

	return &ServiceManager{
		c:             c,
		configManager: configManager,
	}
}

func (s *ServiceManager) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case event := <-s.configManager.Events():
				switch event.Type {
				case config.ConfigUpdated:
					st := <-s.c

					for _, controller := range st.controllers {
						controller.Stop()
					}

					for name, controller := range st.controllers {
						controller.Wait()
						delete(st.controllers, name)
					}

					for _, service := range s.configManager.GetConfig().ServicesInBootOrder() {
						err := s.start(ctx, g, st, service)
						if err != nil {
							return err
						}
					}

					s.c <- st
				default:
					return fmt.Errorf("unknown config event type: %v", event.Type)
				}
			}
		}
	})

	return g.Wait()
}

func (s *ServiceManager) start(ctx context.Context, g *errgroup.Group, st state, id config.ServiceID) error {
	_, ok := st.controllers[id.Name]
	if ok {
		return fmt.Errorf("service already running: %s", id.Name)
	}

	builder, ok := builders[id.Type]
	if !ok {
		return fmt.Errorf("unknown service type: %v", id.Type)
	}

	runner, err := builder(s)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})

	st.controllers[id.Name] = ServiceController{
		service: runner,
		id:      id,
		cancel:  cancel,
		done:    done,
	}

	fmt.Printf("starting service: %s\n", id.Name)

	// TODO: is it kosher to call g.Go() from within g.Go()?
	g.Go(func() error {
		err := runner.Run(ctx)
		close(done)
		return err
	})

	return nil
}

func (s *ServiceManager) Get(id config.ServiceID) (any, error) {
	st := <-s.c
	defer func() {
		s.c <- st
	}()

	controller, ok := st.controllers[id.Name]
	if !ok {
		return nil, fmt.Errorf("service not running: %s", id.Name)
	}

	return controller.service, nil
}

func (s *ServiceManager) ConfigManager() *config.ConfigManager {
	return s.configManager
}

func (s *ServiceManager) RunningServices() []config.ServiceID {
	st := <-s.c
	defer func() {
		s.c <- st
	}()

	var ids []config.ServiceID

	for _, controller := range st.controllers {
		ids = append(ids, controller.id)
	}

	return ids
}
