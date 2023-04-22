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

type BuilderFunc func(m *ServiceManager, conf any) (Runner, error)

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
	st            chan state
	configManager *config.ConfigManager
}

func NewServiceManager(configManager *config.ConfigManager) *ServiceManager {
	st := state{
		controllers: make(map[string]ServiceController),
	}

	c := make(chan state, 1)
	c <- st

	return &ServiceManager{
		st:            c,
		configManager: configManager,
	}
}

func (s *ServiceManager) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	confCh := make(chan *config.Config, 1)

	g.Go(func() error {
		conf, seq := s.configManager.LastChange()
		for {
			select {
			case <-ctx.Done():
				return nil
			case confCh <- conf:
			}

			conf, seq = s.configManager.AwaitChange(ctx, seq)
		}
	})

	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case conf := <-confCh:
				st := <-s.st

				for _, controller := range st.controllers {
					controller.Stop()
				}

				for name, controller := range st.controllers {
					controller.Wait()
					delete(st.controllers, name)
				}

				for _, b := range conf.Bootstraps() {
					err := s.start(ctx, g, st, b)
					if err != nil {
						return err
					}
				}

				s.st <- st
			}
		}
	})

	return g.Wait()
}

func (s *ServiceManager) start(ctx context.Context, g *errgroup.Group, st state, b config.Bootstrap) error {
	_, ok := st.controllers[b.ID.Name]
	if ok {
		return fmt.Errorf("service already running: %s", b.ID.Name)
	}

	builder, ok := builders[b.ID.Type]
	if !ok {
		return fmt.Errorf("unknown service type: %v", b.ID.Type)
	}

	service, err := builder(s, b.Config)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})

	st.controllers[b.ID.Name] = ServiceController{
		service: service,
		id:      b.ID,
		cancel:  cancel,
		done:    done,
	}

	fmt.Printf("starting service: %s\n", b.ID.Name)

	// TODO: is it kosher to call g.Go() from within g.Go()?
	g.Go(func() error {
		err := service.Run(ctx)
		close(done)
		return err
	})

	return nil
}

func (s *ServiceManager) Get(id config.ServiceID) (any, error) {
	st := <-s.st
	defer func() {
		s.st <- st
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
	st := <-s.st
	defer func() {
		s.st <- st
	}()

	var ids []config.ServiceID

	for _, controller := range st.controllers {
		ids = append(ids, controller.id)
	}

	return ids
}
