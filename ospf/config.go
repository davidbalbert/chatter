package ospf

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type rawConfig struct {
	RouterID      RouterID                 `yaml:"router-id"`
	Cost          *uint16                  `yaml:"cost,omitempty"`
	HelloInterval *uint16                  `yaml:"hello-interval,omitempty"`
	DeadInterval  *uint32                  `yaml:"dead-interval,omitempty"`
	Areas         map[string]rawAreaConfig `yaml:",inline,omitempty"`
}

type Config struct {
	routerID      RouterID
	cost          *uint16
	helloInterval *uint16
	deadInterval  *uint32
	areas         map[AreaID]AreaConfig
}

func (r *rawConfig) config() (*Config, error) {
	c := &Config{
		routerID:      r.RouterID,
		cost:          r.Cost,
		helloInterval: r.HelloInterval,
		deadInterval:  r.DeadInterval,
	}

	areas := make(map[AreaID]AreaConfig, len(r.Areas))

	for k, v := range r.Areas {
		if !strings.HasPrefix(k, "area ") {
			return nil, fmt.Errorf("invalid area key '%s'", k)
		}

		areaID, err := parseID(strings.TrimPrefix(k, "area "))
		if err != nil {
			return nil, fmt.Errorf("area-id %w", err)
		}

		a, err := v.areaConfig()
		if err != nil {
			return nil, fmt.Errorf("area %s: %w", areaID, err)
		}

		a.config = c
		areas[AreaID(areaID)] = *a
	}

	c.areas = areas

	return c, nil
}

func (c *Config) rawConfig() *rawConfig {
	areas := make(map[string]rawAreaConfig, len(c.areas))

	for k, v := range c.areas {
		name := fmt.Sprintf("area %d", uint32(k))
		areas[name] = *v.rawAreaConfig()
	}

	return &rawConfig{
		RouterID:      c.routerID,
		Cost:          c.cost,
		HelloInterval: c.helloInterval,
		DeadInterval:  c.deadInterval,
		Areas:         areas,
	}
}

func (c Config) MarshalYAML() (interface{}, error) {
	return c.rawConfig(), nil
}

func (c *Config) UnmarshalYAML(value *yaml.Node) error {
	var raw rawConfig
	if err := value.Decode(&raw); err != nil {
		return err
	}

	config, err := raw.config()
	if err != nil {
		return err
	}

	*c = *config

	return nil
}

func (c *Config) RouterID() RouterID {
	return c.routerID
}

func (c *Config) Cost() uint16 {
	if c.cost != nil {
		return *c.cost
	} else {
		return 1
	}
}

func (c *Config) HelloInterval() uint16 {
	if c.helloInterval != nil {
		return *c.helloInterval
	} else {
		return 10
	}
}

func (c *Config) DeadInterval() uint32 {
	if c.deadInterval != nil {
		return *c.deadInterval
	} else {
		return 40
	}
}

func (c *Config) Areas() map[AreaID]AreaConfig {
	return c.areas
}

func (c Config) String() string {
	return fmt.Sprintf("{RouterID: %s, Cost: %d, HelloInterval: %d, DeadInterval: %d, Areas: %v}", c.RouterID(), c.Cost(), c.HelloInterval(), c.DeadInterval(), c.Areas())
}

type rawAreaConfig struct {
	Cost          *uint16                       `yaml:"cost,omitempty"`
	HelloInterval *uint16                       `yaml:"hello-interval,omitempty"`
	DeadInterval  *uint32                       `yaml:"dead-interval,omitempty"`
	Interfaces    map[string]rawInterfaceConfig `yaml:",inline,omitempty"`
}

type AreaConfig struct {
	cost          *uint16
	helloInterval *uint16
	deadInterval  *uint32
	interfaces    map[string]InterfaceConfig

	config *Config
}

func (r *rawAreaConfig) areaConfig() (*AreaConfig, error) {
	a := &AreaConfig{
		cost:          r.Cost,
		helloInterval: r.HelloInterval,
		deadInterval:  r.DeadInterval,
	}

	interfaces := make(map[string]InterfaceConfig, len(r.Interfaces))

	for k, v := range r.Interfaces {
		if !strings.HasPrefix(k, "interface ") {
			return nil, fmt.Errorf("invalid interface key '%s'", k)
		}

		i := v.interfaceConfig()
		i.areaConfig = a
		interfaces[strings.TrimPrefix(k, "interface ")] = *i
	}

	a.interfaces = interfaces

	return a, nil
}

func (a *AreaConfig) rawAreaConfig() *rawAreaConfig {
	r := &rawAreaConfig{
		Cost:          a.cost,
		HelloInterval: a.helloInterval,
		DeadInterval:  a.deadInterval,
	}

	interfaces := make(map[string]rawInterfaceConfig, len(a.interfaces))

	for k, v := range a.interfaces {
		name := fmt.Sprintf("interface %s", k)
		interfaces[name] = *v.rawInterfaceConfig()
	}

	r.Interfaces = interfaces

	return r
}

func (a *AreaConfig) Cost() uint16 {
	if a.cost != nil {
		return *a.cost
	} else {
		return a.config.Cost()
	}
}

func (a *AreaConfig) HelloInterval() uint16 {
	if a.helloInterval != nil {
		return *a.helloInterval
	} else {
		return a.config.HelloInterval()
	}
}

func (a *AreaConfig) DeadInterval() uint32 {
	if a.deadInterval != nil {
		return *a.deadInterval
	} else {
		return a.config.DeadInterval()
	}
}

func (a *AreaConfig) Interfaces() map[string]InterfaceConfig {
	return a.interfaces
}

func (a AreaConfig) String() string {
	return fmt.Sprintf("{Cost: %d, HelloInterval: %d, DeadInterval: %d, Interfaces: %v}", a.Cost(), a.HelloInterval(), a.DeadInterval(), a.Interfaces())
}

type rawInterfaceConfig struct {
	Cost *uint16 `yaml:"cost,omitempty"`
}

type InterfaceConfig struct {
	cost *uint16

	areaConfig *AreaConfig
}

func (r *rawInterfaceConfig) interfaceConfig() *InterfaceConfig {
	return &InterfaceConfig{
		cost: r.Cost,
	}
}

func (i *InterfaceConfig) rawInterfaceConfig() *rawInterfaceConfig {
	return &rawInterfaceConfig{
		Cost: i.cost,
	}
}

func (i *InterfaceConfig) Cost() uint16 {
	if i.cost != nil {
		return *i.cost
	} else {
		return i.areaConfig.Cost()
	}
}

func (i InterfaceConfig) String() string {
	return fmt.Sprintf("{Cost: %d}", i.Cost())
}

type id uint32

type RouterID id
type AreaID id

func parseID(s string) (id, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err == nil {
		return id(n), nil
	}

	addr, err := netip.ParseAddr(s)
	if err != nil || !addr.Is4() {
		return 0, fmt.Errorf("must be an IPv4 address or an unsigned 32 bit integer")
	}

	return id(binary.BigEndian.Uint32(addr.AsSlice())), nil
}

func (i id) String() string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(i))
	addr := netip.AddrFrom4(b)

	return addr.String()
}

func (r *RouterID) UnmarshalText(text []byte) error {
	id, err := parseID(string(text))
	if err != nil {
		return fmt.Errorf("router-id %w", err)
	}

	*r = RouterID(id)
	return nil
}

func (r RouterID) MarshalText() ([]byte, error) {
	return []byte(r.String()), nil
}

func (r RouterID) String() string {
	return id(r).String()
}

func (a *AreaID) UnmarshalText(text []byte) error {
	id, err := parseID(string(text))
	if err != nil {
		return fmt.Errorf("area-id %w", err)
	}

	*a = AreaID(id)
	return nil
}

func (a AreaID) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a AreaID) String() string {
	return id(a).String()
}
