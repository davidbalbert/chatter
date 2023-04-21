package config

import (
	"encoding/binary"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"strings"

	"github.com/davidbalbert/chatter/chatterd/common"
)

func parseID(s string) (uint32, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err == nil {
		return uint32(n), nil
	}

	addr, err := netip.ParseAddr(s)
	if err != nil || !addr.Is4() {
		return 0, fmt.Errorf("must be an IPv4 address or an unsigned 32 bit integer")
	}

	return binary.BigEndian.Uint32(addr.AsSlice()), nil
}

type OSPFConfig struct {
	RouterID           common.RouterID
	Cost               uint16
	HelloInterval      uint16
	RouterDeadInterval uint32
	Areas              map[common.AreaID]OSPFAreaConfig
}

func (c *OSPFConfig) shouldRun() bool {
	for _, area := range c.Areas {
		if len(area.Interfaces) > 0 {
			return true
		}
	}

	return false
}

func (c *OSPFConfig) dependencies() []ServiceID {
	return []ServiceID{ServiceInterfaceMonitor}
}

func (c *OSPFConfig) copy() protocolConfig {
	newConfig := OSPFConfig{
		RouterID:           c.RouterID,
		Cost:               c.Cost,
		HelloInterval:      c.HelloInterval,
		RouterDeadInterval: c.RouterDeadInterval,
		Areas:              make(map[common.AreaID]OSPFAreaConfig),
	}

	for k, v := range c.Areas {
		newConfig.Areas[k] = v.copy()
	}

	return &newConfig
}

func (c *OSPFConfig) InterfaceConfigs() map[string]OSPFInterfaceConfig {
	configs := make(map[string]OSPFInterfaceConfig)

	for _, area := range c.Areas {
		for name, conf := range area.Interfaces {
			configs[name] = conf
		}
	}

	return configs
}

type OSPFAreaConfig struct {
	Cost               uint16
	HelloInterval      uint16
	RouterDeadInterval uint32
	Interfaces         map[string]OSPFInterfaceConfig
}

func (c *OSPFAreaConfig) copy() OSPFAreaConfig {
	newConfig := OSPFAreaConfig{
		Cost:               c.Cost,
		HelloInterval:      c.HelloInterval,
		RouterDeadInterval: c.RouterDeadInterval,
		Interfaces:         make(map[string]OSPFInterfaceConfig),
	}

	for k, v := range c.Interfaces {
		newConfig.Interfaces[k] = v
	}

	return newConfig
}

type OSPFInterfaceConfig struct {
	AreaID             common.AreaID
	Cost               uint16
	HelloInterval      uint16
	RouterDeadInterval uint32
}

func parseOSPFConfig(data map[string]interface{}) (*OSPFConfig, error) {
	c := &OSPFConfig{
		RouterID:           0,
		Cost:               1,
		HelloInterval:      10,
		RouterDeadInterval: 40,
		Areas:              make(map[common.AreaID]OSPFAreaConfig),
	}

	for k, v := range data {
		if k == "router-id" {
			switch v := v.(type) {
			case string:
				id, err := parseID(v)
				if err != nil {
					return nil, fmt.Errorf("ospf: invalid router-id: %s", err)
				}

				c.RouterID = common.RouterID(id)
			case int:
				if v < 0 {
					return nil, fmt.Errorf("ospf: router-id must be positive: %d", v)
				} else if v > math.MaxUint32 {
					return nil, fmt.Errorf("ospf: router-id too big: %d", v)
				}

				c.RouterID = common.RouterID(v)
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

			c.RouterDeadInterval = uint32(v)
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

			c.Areas[common.AreaID(id)] = *ac
		} else {
			return nil, fmt.Errorf("ospf: unknown key: %s", k)
		}
	}

	for k, ac := range c.Areas {
		ac.setDefaults(c)
		c.Areas[k] = ac
	}

	_, ok := c.Areas[common.AreaID(0)]
	if !ok {
		return nil, fmt.Errorf("ospf: backbone area must be configured")
	}

	return c, nil
}

func (ac *OSPFAreaConfig) setDefaults(c *OSPFConfig) {
	if ac.HelloInterval == 0 {
		ac.HelloInterval = c.HelloInterval
	}

	if ac.RouterDeadInterval == 0 {
		ac.RouterDeadInterval = c.RouterDeadInterval
	}

	if ac.Cost == 0 {
		ac.Cost = c.Cost
	}

	for k, ic := range ac.Interfaces {
		ic.setDefaults(ac)
		ac.Interfaces[k] = ic
	}
}

func (ic *OSPFInterfaceConfig) setDefaults(ac *OSPFAreaConfig) {
	if ic.Cost == 0 {
		ic.Cost = ac.Cost
	}

	if ic.HelloInterval == 0 {
		ic.HelloInterval = ac.HelloInterval
	}

	if ic.RouterDeadInterval == 0 {
		ic.RouterDeadInterval = ac.RouterDeadInterval
	}
}

func parseAreaConfig(areaID string, data map[string]interface{}) (*OSPFAreaConfig, error) {
	ac := OSPFAreaConfig{
		Cost:               0,
		HelloInterval:      0,
		RouterDeadInterval: 0,
		Interfaces:         make(map[string]OSPFInterfaceConfig),
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

			ac.RouterDeadInterval = uint32(v)
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

func parseInterfaceConfig(areaName, name string, data map[string]interface{}) (*OSPFInterfaceConfig, error) {
	id, err := parseID(areaName)
	if err != nil {
		return nil, fmt.Errorf("ospf: invalid area id: %s", err)
	}

	ic := OSPFInterfaceConfig{
		AreaID: common.AreaID(id),
		Cost:   0,
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

			ic.HelloInterval = uint16(v)
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

			ic.RouterDeadInterval = uint32(v)
		} else {
			return nil, fmt.Errorf("ospf area %s interface %s: unknown key: %s", areaName, name, k)
		}
	}

	return &ic, nil
}
