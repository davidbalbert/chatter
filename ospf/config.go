package ospf

import (
	"fmt"
	"math"
	"strings"
)

type Config struct {
	RouterID      RouterID
	Cost          uint16
	HelloInterval uint16
	DeadInterval  uint32
	Areas         map[AreaID]AreaConfig
}

type AreaConfig struct {
	Cost          uint16
	HelloInterval uint16
	DeadInterval  uint32
	Interfaces    map[string]InterfaceConfig
}

type InterfaceConfig struct {
	Cost uint16
}

func ParseConfig(data map[string]interface{}) (*Config, error) {
	c := &Config{
		RouterID:      0,
		Cost:          1,
		HelloInterval: 10,
		DeadInterval:  40,
		Areas:         make(map[AreaID]AreaConfig),
	}

	for k, v := range data {
		if k == "router-id" {
			switch v := v.(type) {
			case string:
				id, err := parseID(v)
				if err != nil {
					return nil, fmt.Errorf("ospf: invalid router-id: %s", err)
				}

				c.RouterID = RouterID(id)
			case int:
				if v < 0 {
					return nil, fmt.Errorf("ospf: router-id must be positive: %d", v)
				} else if v > math.MaxUint32 {
					return nil, fmt.Errorf("ospf: router-id too big: %d", v)
				}

				c.RouterID = RouterID(v)
			default:
				return nil, fmt.Errorf("ospf: router-id must be an IPv4 address or an unsigned 32 bit integer")
			}
		} else if k == "cost" {
			v, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("ospf: cost must be an integer")
			}

			if v < 1 {
				return nil, fmt.Errorf("ospf: cost too small: %d", v)
			} else if v > math.MaxUint16 {
				return nil, fmt.Errorf("ospf: cost too big: %d", v)
			}

			c.Cost = uint16(v)
		} else if k == "hello-interval" {
			v, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("ospf: hello-interval must be an integer")
			}

			if v < 1 {
				return nil, fmt.Errorf("ospf: hello-interval too small: %d", v)
			} else if v > math.MaxUint16 {
				return nil, fmt.Errorf("ospf: hello-interval too big: %d", v)
			}

			c.HelloInterval = uint16(v)
		} else if k == "dead-interval" {
			v, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("ospf: dead-interval must be an integer")
			}

			if v < 1 {
				return nil, fmt.Errorf("ospf: dead-interval too small: %d", v)
			} else if v > math.MaxUint32 {
				return nil, fmt.Errorf("ospf: dead-interval too big: %d", v)
			}

			c.DeadInterval = uint32(v)
		} else if strings.HasPrefix(k, "area ") {
			name := strings.TrimPrefix(k, "area ")

			id, err := parseID(name)
			if err != nil {
				return nil, fmt.Errorf("ospf: invalid area id: %s", err)
			}

			area, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("ospf: area must be a map")
			}

			ac, err := parseAreaConfig(name, area)
			if err != nil {
				return nil, err
			}

			c.Areas[AreaID(id)] = *ac
		} else {
			return nil, fmt.Errorf("ospf: unknown key: %s", k)
		}
	}

	for k, ac := range c.Areas {
		ac.setDefaults(c)
		c.Areas[k] = ac
	}

	_, ok := c.Areas[AreaID(0)]
	if !ok {
		return nil, fmt.Errorf("ospf: backbone area must be configured")
	}

	return c, nil
}

func (ac *AreaConfig) setDefaults(c *Config) {
	if ac.HelloInterval == 0 {
		ac.HelloInterval = c.HelloInterval
	}

	if ac.DeadInterval == 0 {
		ac.DeadInterval = c.DeadInterval
	}

	if ac.Cost == 0 {
		ac.Cost = c.Cost
	}

	for k, ic := range ac.Interfaces {
		ic.setDefaults(ac)
		ac.Interfaces[k] = ic
	}
}

func (ic *InterfaceConfig) setDefaults(ac *AreaConfig) {
	if ic.Cost == 0 {
		ic.Cost = ac.Cost
	}
}

func parseAreaConfig(areaID string, data map[string]interface{}) (*AreaConfig, error) {
	ac := AreaConfig{
		Cost:          0,
		HelloInterval: 0,
		DeadInterval:  0,
		Interfaces:    make(map[string]InterfaceConfig),
	}

	for k, v := range data {
		if k == "cost" {
			v, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("ospf area %s: cost must be an integer", areaID)
			}

			if v < 1 {
				return nil, fmt.Errorf("ospf area %s: cost too small: %d", areaID, v)
			} else if v > math.MaxUint16 {
				return nil, fmt.Errorf("ospf area %s: cost too big: %d", areaID, v)
			}

			ac.Cost = uint16(v)
		} else if k == "hello-interval" {
			v, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("ospf area %s: hello-interval must be an integer", areaID)
			}

			if v < 1 {
				return nil, fmt.Errorf("ospf area %s: hello-interval too small: %d", areaID, v)
			} else if v > math.MaxUint16 {
				return nil, fmt.Errorf("ospf area %s: hello-interval too big: %d", areaID, v)
			}

			ac.HelloInterval = uint16(v)
		} else if k == "dead-interval" {
			v, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("ospf area %s: dead-interval must be an integer", areaID)
			}

			if v < 1 {
				return nil, fmt.Errorf("ospf area %s: dead-interval too small: %d", areaID, v)
			} else if v > math.MaxUint32 {
				return nil, fmt.Errorf("ospf area %s: dead-interval too big: %d", areaID, v)
			}

			ac.DeadInterval = uint32(v)
		} else if strings.HasPrefix(k, "interface ") {
			interfaceName := strings.TrimPrefix(k, "interface ")

			i, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("ospf area %s: interface must be a map", interfaceName)
			}

			ic, err := parseInterfaceConfig(areaID, interfaceName, i)
			if err != nil {
				return nil, err
			}

			ac.Interfaces[interfaceName] = *ic
		} else {
			return nil, fmt.Errorf("ospf area %s: unknown key: %s", areaID, k)
		}
	}

	return &ac, nil
}

func parseInterfaceConfig(areaName, name string, data map[string]interface{}) (*InterfaceConfig, error) {
	ic := InterfaceConfig{
		Cost: 0,
	}

	for k, v := range data {
		if k == "cost" {
			v, ok := v.(int)
			if !ok {
				return nil, fmt.Errorf("ospf area %s interface %s: cost must be an integer", areaName, name)
			}

			if v < 1 {
				return nil, fmt.Errorf("ospf area %s interface %s: cost too small: %d", areaName, name, v)
			} else if v > math.MaxUint16 {
				return nil, fmt.Errorf("ospf area %s interface %s: cost too big: %d", areaName, name, v)
			}

			ic.Cost = uint16(v)
		} else {
			return nil, fmt.Errorf("ospf area %s interface %s: unknown key: %s", areaName, name, k)
		}
	}

	return &ic, nil
}
