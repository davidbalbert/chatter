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

// func (s *Server) GetInterfaces(ctx context.Context, req *rpc.GetInterfacesRequest) (*rpc.GetInterfacesResponse, error) {
// 	return &rpc.GetInterfacesResponse{
// 		Interfaces: []*rpc.Interface{
// 			{
// 				Name: "bridge100",
// 			},
// 		},
// 	}, nil
// }

type Client struct {
	*grpc.ClientConn
	rpcClient rpc.APIClient
}

func NewClient() (*Client, error) {
	target := fmt.Sprintf("unix://%s", socketPath)
	conn, err := grpc.Dial(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	rpcClient := rpc.NewAPIClient(conn)

	return &Client{
		ClientConn: conn,
		rpcClient:  rpcClient,
	}, nil
}

func (c *Client) GetVersion(ctx context.Context) (string, error) {
	resp, err := c.rpcClient.GetVersion(ctx, &rpc.GetVersionRequest{})
	if err != nil {
		return "", err
	}

	return resp.Version, nil
}
