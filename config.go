package main

import (
	"fmt"
	"net/netip"
)

type NetworkConfig struct {
	Network netip.Prefix
	AreaID  netip.Addr
}

type InterfaceConfig struct {
	Name          string
	HelloInterval uint16
	DeadInterval  uint32
	NetworkType   networkType
}

type Config struct {
	RouterID   netip.Addr
	Networks   []NetworkConfig
	Interfaces []InterfaceConfig
}

func NewConfig(routerID string) (*Config, error) {
	addr, err := netip.ParseAddr(routerID)
	if err != nil {
		return nil, fmt.Errorf("error parsing router id: %w", err)
	}

	return &Config{
		RouterID: addr,
	}, nil
}

func (c *Config) AddNetwork(network, areaID string) error {
	net, err := netip.ParsePrefix(network)
	if err != nil {
		return fmt.Errorf("error parsing network prefix: %w", err)
	}

	if net.Addr().Is6() {
		return fmt.Errorf("ipv6 networks are not supported")
	}

	addr, err := netip.ParseAddr(areaID)
	if err != nil {
		return fmt.Errorf("error parsing area id: %w", err)
	}

	if addr.Is6() {
		return fmt.Errorf("ipv6 area ids are not supported")
	}

	c.Networks = append(c.Networks, NetworkConfig{
		Network: net,
		AreaID:  addr,
	})

	return nil
}

// TODO: Go question – is networkType networkType kosher?
func (c *Config) AddInterface(name string, networkType networkType, helloInterval uint16, deadInterval uint32) {
	c.Interfaces = append(c.Interfaces, InterfaceConfig{
		Name:          name,
		HelloInterval: helloInterval,
		DeadInterval:  deadInterval,
		NetworkType:   networkType,
	})
}
