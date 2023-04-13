package api

import (
	"context"
	"fmt"
	"net"

	"github.com/davidbalbert/chatter/rpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const socketPath = "/tmp/ospfd.sock"

type Server struct {
	rpc.UnimplementedAPIServer
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	rpc.RegisterAPIServer(grpcServer, s)

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

func (s *Server) GetRandInt(ctx context.Context, in *rpc.Empty) (*rpc.RandInt, error) {
	return &rpc.RandInt{Value: 0}, nil
}

func (s *Server) GetRandString(ctx context.Context, in *rpc.Empty) (*rpc.RandString, error) {
	return &rpc.RandString{Value: "Hello, world"}, nil
}

type Client interface {
	rpc.APIClient
}

type client struct {
	*grpc.ClientConn
	rpc.APIClient
}

func NewClient() (Client, error) {
	target := fmt.Sprintf("unix://%s", socketPath)
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &client{
		ClientConn: conn,
		APIClient:  rpc.NewAPIClient(conn),
	}, nil
}
