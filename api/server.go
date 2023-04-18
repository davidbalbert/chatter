package api

import (
	"context"
	"net"

	"github.com/davidbalbert/chatter/chatterd/services"
	"github.com/davidbalbert/chatter/rpc"
	"github.com/davidbalbert/chatter/system"
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

func (s *Server) GetVersion(ctx context.Context) (string, error) {
	return s.version, nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.shutdown()
	return nil
}

func (s *Server) GetInterfaces(ctx context.Context) ([]system.Interface, error) {
	// return s.InterfaceMonitor.Interfaces(), nil
	return nil, nil
}
