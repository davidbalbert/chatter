package config

import (
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

type protocolConfig interface {
	shouldRun() bool
	dependencies() []ServiceID
	copy() protocolConfig
}

type Config struct {
	protocolConfigs map[ServiceID]protocolConfig
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
		protocolConfigs: make(map[ServiceID]protocolConfig),
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

			c.protocolConfigs[ServiceOSPF] = ospfConfig
		default:
			return nil, fmt.Errorf("unknown top level key: %s", k)
		}
	}

	return &c, nil
}

type Bootstrap struct {
	ID     ServiceID
	Config any
}

func (c *Config) Bootstraps() []Bootstrap {
	g := newGraph()

	g.addNode(ServiceAPIServer)

	for s, conf := range c.protocolConfigs {
		if conf.shouldRun() {
			g.addNode(s, conf.dependencies()...)
		}
	}

	ids := g.topologicalSort()

	bootstraps := make([]Bootstrap, 0, len(ids))

	for _, id := range ids {
		conf := c.protocolConfigs[id]

		bootstraps = append(bootstraps, Bootstrap{
			ID:     id,
			Config: conf,
		})
	}

	return bootstraps
}

func (c *Config) Copy() *Config {
	newConfig := Config{
		protocolConfigs: make(map[ServiceID]protocolConfig),
	}

	for k, v := range c.protocolConfigs {
		newConfig.protocolConfigs[k] = v.copy()
	}

	return &newConfig
}

func (c *Config) validate() error {
	return nil
}
