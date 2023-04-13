package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/davidbalbert/chatter/ospf"
)

type Config struct {
	OSPF ospf.Config
}

func ParseConfig(s string) (*Config, error) {
	var data map[string]interface{}

	if err := yaml.Unmarshal([]byte(s), &data); err != nil {
		return nil, err
	}

	c := Config{}

	for k, v := range data {
		switch k {
		case "ospf":
			v, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("ospf must be a map")
			}

			ospfConfig, err := ospf.ParseConfig(v)
			if err != nil {
				return nil, err
			}

			c.OSPF = *ospfConfig
		default:
			return nil, fmt.Errorf("unknown top level key: %s", k)
		}
	}

	return &c, nil
}
