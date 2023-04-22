package config

import (
	"context"

	"github.com/davidbalbert/chatter/sync"
)

type ConfigManager struct {
	*sync.Notifier[*Config]
}

func NewConfigManager(path string) (*ConfigManager, error) {
	conf, err := loadConfig(path)
	if err != nil {
		return nil, err
	}

	return &ConfigManager{sync.NewNotifier(conf)}, nil
}

func (c *ConfigManager) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (c *ConfigManager) UpdateConfig(conf *Config) error {
	err := conf.validate()
	if err != nil {
		return err
	}

	c.NotifyChange(conf)

	return nil
}
