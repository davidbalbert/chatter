package config

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServiceType int

const (
	ServiceTypeInterfaceMonitor ServiceType = iota
	ServiceTypeOSPF
)

type Service struct {
	Type ServiceType
	Name string
}

var (
	ServiceInterfaceMonitor = Service{Type: ServiceTypeInterfaceMonitor, Name: "InterfaceMonitor"}
	ServiceOSPF             = Service{Type: ServiceTypeOSPF, Name: "OSPF"}
)

type protocolConfig interface {
	ShouldRun() bool
	Dependencies() []Service
	Copy() protocolConfig
}

type Config struct {
	protocols map[Service]protocolConfig
}

func loadConfig(path string) (*Config, error) {
	s, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config, err := parseConfig(string(s))
	if err != nil {
		return nil, err
	}

	err = config.validate()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func parseConfig(s string) (*Config, error) {
	var data map[string]interface{}

	if err := yaml.Unmarshal([]byte(s), &data); err != nil {
		return nil, err
	}

	c := Config{
		protocols: make(map[Service]protocolConfig),
	}

	for k, v := range data {
		switch k {
		case "ospf":
			v, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("ospf must be a map")
			}

			ospfConfig, err := parseOSPFConfig(v)
			if err != nil {
				return nil, err
			}

			c.protocols[ServiceOSPF] = ospfConfig
		default:
			return nil, fmt.Errorf("unknown top level key: %s", k)
		}
	}

	return &c, nil
}

func (c *Config) Copy() *Config {
	newConfig := Config{
		protocols: make(map[Service]protocolConfig),
	}

	for k, v := range c.protocols {
		newConfig.protocols[k] = v.Copy()
	}

	return &newConfig
}

func (c *Config) validate() error {
	return nil
}

type EventType int

const (
	ConfigUpdated EventType = iota
)

type Event struct {
	Type EventType
}

type managerState struct {
	runningConfig *Config
}

type ConfigManager struct {
	s       chan managerState
	configs chan Config
	events  chan Event
	path    string
}

func NewConfigManager(path string) (*ConfigManager, error) {
	config, err := loadConfig(path)
	if err != nil {
		return nil, err
	}

	s := make(chan managerState, 1)
	s <- managerState{
		runningConfig: config,
	}

	return &ConfigManager{
		s:       s,
		configs: make(chan Config),
		events:  make(chan Event),
		path:    path,
	}, nil
}

func (c *ConfigManager) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case config := <-c.configs:
			state := <-c.s
			state.runningConfig = &config
			c.s <- state

			c.events <- Event{
				Type: ConfigUpdated,
			}
		}
	}
}

func (c *ConfigManager) Events() <-chan Event {
	return c.events
}

func (c *ConfigManager) UpdateConfig(config *Config) error {
	err := config.validate()
	if err != nil {
		return err
	}

	c.configs <- *config

	return nil
}

func (c *ConfigManager) GetConfig() *Config {
	state := <-c.s
	config := state.runningConfig.Copy()
	c.s <- state

	return config
}

type Runner interface {
	Run(ctx context.Context) error
}

type BuilderFunc func(*ServiceManager) (Runner, error)

var builders = make(map[ServiceType]BuilderFunc)

func registerServiceType(t ServiceType, fn BuilderFunc) error {
	_, ok := builders[t]
	if ok {
		return fmt.Errorf("service type already registered: %v", t)
	}

	builders[t] = fn

	return nil
}

func MustRegisterServiceType(t ServiceType, fn BuilderFunc) {
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
	configManager *ConfigManager
	services      map[string]ServiceController
}

func NewServiceManager(configManager *ConfigManager) *ServiceManager {
	return &ServiceManager{
		configManager: configManager,
		services:      make(map[string]ServiceController),
	}
}

func (s *ServiceManager) Run(ctx context.Context) error {
	return nil
}
