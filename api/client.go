package api

import (
	"context"
	"fmt"

	"github.com/davidbalbert/chatter/config"
	"github.com/davidbalbert/chatter/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	*grpc.ClientConn
	rpcClient rpc.APIClient
}

func NewClient(socket string) (*Client, error) {
	target := fmt.Sprintf("unix://%s", socket)
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

func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.rpcClient.Shutdown(ctx, &rpc.ShutdownRequest{})
	return err
}

func (c *Client) GetInterfaces(ctx context.Context) ([]*rpc.Interface, error) {
	resp, err := c.rpcClient.GetInterfaces(ctx, &rpc.GetInterfacesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.Interfaces, nil
}

func (c *Client) GetServices(ctx context.Context) ([]config.ServiceID, error) {
	resp, err := c.rpcClient.GetServices(ctx, &rpc.GetServicesRequest{})
	if err != nil {
		return nil, err
	}

	services := make([]config.ServiceID, len(resp.Services))
	for i, service := range resp.Services {
		services[i] = config.ServiceID{
			Type: config.ServiceType(service.Type),
			Name: service.Name,
		}
	}

	return services, nil
}
