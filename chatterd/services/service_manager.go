package services

import (
	"context"
	"fmt"

	"github.com/davidbalbert/chatter/config"
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
	cancel context.CancelFunc
	done   chan struct{}
}

func (c *ServiceController) Stop() {
	c.cancel()
}

func (c *ServiceController) Wait() error {
	<-c.done
	return nil
}

type ServiceManager struct {
	configManager *config.ConfigManager
	services      map[string]ServiceController
}

func NewServiceManager(configManager *config.ConfigManager) *ServiceManager {
	return &ServiceManager{
		configManager: configManager,
		services:      make(map[string]ServiceController),
	}
}

func (s *ServiceManager) Run(ctx context.Context) error {
	return nil
}
