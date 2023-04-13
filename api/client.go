package api

import (
	"context"
	"fmt"
	"net"

	"github.com/davidbalbert/chatter/rpc"
	"github.com/davidbalbert/chatter/system"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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

func (c *Client) Shutdown(ctx context.Context) error {
	_, err := c.rpcClient.Shutdown(ctx, &rpc.ShutdownRequest{})
	return err
}

func (c *Client) GetInterfaces(ctx context.Context) ([]system.Interface, error) {
	resp, err := c.rpcClient.GetInterfaces(ctx, &rpc.GetInterfacesRequest{})
	if err != nil {
		return nil, err
	}

	interfaces := make([]system.Interface, len(resp.Interfaces))
	for i, iface := range resp.Interfaces {
		interfaces[i] = system.Interface{
			Interface: net.Interface{
				Index:        int(iface.Index),
				MTU:          int(iface.Mtu),
				Name:         iface.Name,
				HardwareAddr: net.HardwareAddr(iface.HardwareAddr),
				Flags:        net.Flags(iface.Flags),
			},
		}
	}

	return interfaces, nil
}
