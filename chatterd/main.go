package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/davidbalbert/chatter/api"
	"github.com/davidbalbert/chatter/chatterd/services"
	"github.com/davidbalbert/chatter/config"
	"github.com/davidbalbert/chatter/net/netmon"
	"github.com/davidbalbert/chatter/ospf"
	"golang.org/x/sync/errgroup"
)

var (
	version    string
	configPath string
	socketPath string
)

func main() {
	fmt.Printf("Starting chatterd %s with uid %d\n", version, os.Getuid())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	g, ctx := errgroup.WithContext(ctx)

	flag.StringVar(&configPath, "config", "/etc/chatterd/chatterd.yaml", "path to chatterd.yaml")
	flag.StringVar(&socketPath, "socket", "/var/run/chatterd.sock", "path to chatterd socket")

	flag.Parse()

	services.MustRegisterServiceType(config.ServiceTypeAPIServer, func(serviceManager *services.ServiceManager, conf any) (services.Runner, error) {
		return api.NewServer(serviceManager, socketPath, cancel, version), nil
	})
	services.MustRegisterServiceType(config.ServiceTypeInterfaceMonitor, netmon.New)
	services.MustRegisterServiceType(config.ServiceTypeOSPF, ospf.NewInstance)

	configManager, err := config.NewConfigManager(configPath)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}

	serviceManager := services.NewServiceManager(configManager)

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
