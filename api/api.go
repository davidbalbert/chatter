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

func (s *Server) GetVersion(ctx context.Context, req *rpc.GetVersionRequest) (*rpc.GetVersionResponse, error) {
	return &rpc.GetVersionResponse{
		Version: "0.0.1",
	}, nil
}

func (s *Server) GetInterfaces(ctx context.Context, req *rpc.GetInterfacesRequest) (*rpc.GetInterfacesResponse, error) {
	return &rpc.GetInterfacesResponse{
		Interfaces: []*rpc.Interface{
			{
				Name: "bridge100",
			},
		},
	}, nil
}

type Client struct {
	*grpc.ClientConn
	rpc.APIClient
}

func NewClient() (*Client, error) {
	target := fmt.Sprintf("unix://%s", socketPath)
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		ClientConn: conn,
		APIClient:  rpc.NewAPIClient(conn),
	}, nil
}
