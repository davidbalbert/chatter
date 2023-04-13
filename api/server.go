package api

import (
	"context"
	"net"

	"github.com/davidbalbert/chatter/rpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type Server struct {
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
