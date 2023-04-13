package api

import (
	"context"
	"fmt"

	"github.com/davidbalbert/chatter/rpc"
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
