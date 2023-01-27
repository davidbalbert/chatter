//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api.proto

package api

import (
	context "context"
	"math/rand"
)

const SocketPath = "/tmp/ospfd.sock"

type apiServer struct {
	UnimplementedAPIServer
}

func NewAPIServer() APIServer {
	return &apiServer{}
}

func (s *apiServer) GetRandInt(ctx context.Context, in *Empty) (*RandInt, error) {
	return &RandInt{Value: rand.Uint32()}, nil
}

func (s *apiServer) GetRandString(ctx context.Context, in *Empty) (*RandString, error) {
	return &RandString{Value: "Hello, world!"}, nil
}
