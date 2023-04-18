package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/davidbalbert/chatter/config"
	"github.com/davidbalbert/chatter/ospf"
	"github.com/davidbalbert/chatter/system"
	"golang.org/x/sync/errgroup"
)

const s = `
ospf:
  router-id: 192.168.200.1

  area 0:
    interface bridge100: {}
`

func main() {
	fmt.Printf("Starting chatterd v0.0.1 with uid %d\n", os.Getuid())

	config.MustRegisterServiceType(config.ServiceTypeInterfaceMonitor, system.NewInterfaceMonitor)
	config.MustRegisterServiceType(config.ServiceTypeOSPF, ospf.NewInstance)

	var configPath string

	flag.StringVar(&configPath, "config", "/etc/chatterd/chatterd.conf", "path to chatterd.conf")
	flag.StringVar(&configPath, "c", "/etc/chatterd/chatterd.conf", "path to chatterd.conf (shorthand)")
	flag.Parse()

	configManager, err := config.NewConfigManager(configPath)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	g, ctx := errgroup.WithContext(ctx)

	serviceManager := config.NewServiceManager(configManager)

	g.Go(func() error {
		return configManager.Run(ctx)
	})

	g.Go(func() error {
		return serviceManager.Run(ctx)
	})

	err = g.Wait()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
