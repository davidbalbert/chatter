package config

import (
	"context"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServiceType int

const (
	ServiceTypeAPIServer ServiceType = iota
	ServiceTypeInterfaceMonitor
	ServiceTypeOSPF
)

func (t ServiceType) String() string {
	switch t {
	case ServiceTypeAPIServer:
		return "APIServer"
	case ServiceTypeInterfaceMonitor:
		return "InterfaceMonitor"
	case ServiceTypeOSPF:
		return "OSPF"
	default:
		return fmt.Sprintf("unknown service type: %d", t)
	}
}

type ServiceID struct {
	Type ServiceType
	Name string
}

var (
	ServiceAPIServer        = ServiceID{Type: ServiceTypeAPIServer, Name: "APIServer"}
	ServiceInterfaceMonitor = ServiceID{Type: ServiceTypeInterfaceMonitor, Name: "InterfaceMonitor"}
	ServiceOSPF             = ServiceID{Type: ServiceTypeOSPF, Name: "OSPF"}
)

type graph struct {
	nodes map[ServiceID][]ServiceID
}

func newGraph() *graph {
	return &graph{nodes: make(map[ServiceID][]ServiceID)}
}

func (g *graph) addNode(service ServiceID, deps ...ServiceID) {
	g.nodes[service] = deps
}

func (g *graph) topologicalSort() []ServiceID {
	visited := make(map[ServiceID]bool)
	stack := []ServiceID{}

	var visit func(ServiceID)

	visit = func(service ServiceID) {
		if _, ok := visited[service]; !ok {
			visited[service] = true

			for _, dep := range g.nodes[service] {
				visit(dep)
			}

			stack = append(stack, service)
		}
	}

	for service := range g.nodes {
		visit(service)
	}

	return stack
}

type protocolConfig interface {
	shouldRun() bool
	dependencies() []ServiceID
	copy() protocolConfig
}

type Config struct {
	protocols map[ServiceID]protocolConfig
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
		protocols: make(map[ServiceID]protocolConfig),
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

func (c *Config) ServicesInBootOrder() []ServiceID {
	g := newGraph()

	g.addNode(ServiceAPIServer)

	for s, p := range c.protocols {
		if p.shouldRun() {
			g.addNode(s, p.dependencies()...)
		}
	}

	return g.topologicalSort()
}

func (c *Config) copy() *Config {
	newConfig := Config{
		protocols: make(map[ServiceID]protocolConfig),
	}

	for k, v := range c.protocols {
		newConfig.protocols[k] = v.copy()
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
	data any
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

	events := make(chan Event, 1)
	events <- Event{
		Type: ConfigUpdated,
	}

	return &ConfigManager{
		s:       s,
		configs: make(chan Config),
		events:  events,
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
	config := state.runningConfig.copy()
	c.s <- state

	return config
}
