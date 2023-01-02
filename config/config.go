package config

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/davidbalbert/ospfd/ospf"
)

type Config struct {
	OSPF ospf.Config `yaml:"ospf"`
}

func ParseConfig(s string) (*Config, error) {
	var c Config
	d := yaml.NewDecoder(strings.NewReader(s))
	d.KnownFields(true)

	if err := d.Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}
