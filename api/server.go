package api

import (
	"context"
	"net"

	"github.com/davidbalbert/chatter/rpc"
	"github.com/davidbalbert/chatter/system"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Server struct {
	InterfaceMonitor system.InterfaceMonitor
	ShutdownFunc     context.CancelFunc
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("unix", socketPath)
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
	return "0.0.1", nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.ShutdownFunc()
	return nil
}

func (s *Server) GetInterfaces(ctx context.Context) ([]system.Interface, error) {
	return s.InterfaceMonitor.Interfaces(), nil
}
