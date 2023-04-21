package api

import (
	"context"
	"fmt"
	"net"

	"github.com/davidbalbert/chatter/chatterd/services"
	"github.com/davidbalbert/chatter/config"
	"github.com/davidbalbert/chatter/rpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Server struct {
	serviceManager *services.ServiceManager
	shutdown       context.CancelFunc
	socket         string
	version        string
}

func NewServer(serviceManager *services.ServiceManager, socket string, shutdown context.CancelFunc, version string) *Server {
	return &Server{
		serviceManager: serviceManager,
		shutdown:       shutdown,
		socket:         socket,
		version:        version,
	}
}

func (s *Server) Run(ctx context.Context) error {
	listener, err := net.Listen("unix", s.socket)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	rpcServer := rpc.NewAPIServer(s)

	rpc.RegisterAPIServer(grpcServer, rpcServer)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return grpcServer.Serve(listener)
	})

	g.Go(func() error {
		<-ctx.Done()
		grpcServer.GracefulStop()
		return nil
	})

	return g.Wait()
}

func (s *Server) SendEvent(event config.Event) error {
	return fmt.Errorf("api server does not support sending events")
}

func (s *Server) GetVersion(ctx context.Context) (string, error) {
	return s.version, nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.shutdown()
	return nil
}

func (s *Server) GetServices(ctx context.Context) ([]config.ServiceID, error) {
	return s.serviceManager.RunningServices(), nil
}

func (s *Server) GetInterfaces(ctx context.Context) ([]*rpc.Interface, error) {
	netifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	ifaces := make([]*rpc.Interface, len(netifs))

	for i, netif := range netifs {
		addrs, err := netif.Addrs()
		if err != nil {
			return nil, err
		}

		prefixes := make([]*rpc.Prefix, len(addrs))
		for j, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				return nil, fmt.Errorf("failed to convert address to IPNet")
			}

			ones, _ := ipnet.Mask.Size()

			ip := ipnet.IP.To4()
			if ip == nil {
				ip = ipnet.IP
			}

			prefixes[j] = &rpc.Prefix{
				Addr:      ip,
				PrefixLen: int32(ones),
			}
		}

		ifaces[i] = &rpc.Interface{
			Index:        int32(netif.Index),
			Mtu:          int32(netif.MTU),
			Name:         netif.Name,
			HardwareAddr: netif.HardwareAddr,
			Flags:        uint32(netif.Flags),
			Addrs:        prefixes,
		}
	}

	return ifaces, nil
}
