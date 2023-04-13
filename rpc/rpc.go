//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative rpc.proto

package rpc

import (
	context "context"
)

type APIService interface {
	GetVersion(ctx context.Context) (string, error)
}

type Server struct {
	UnimplementedAPIServer
	apiService APIService
}

func NewAPIServer(apiService APIService) *Server {
	return &Server{
		apiService: apiService,
	}
}

func (s *Server) GetVersion(ctx context.Context, req *GetVersionRequest) (*GetVersionResponse, error) {
	version, err := s.apiService.GetVersion(ctx)
	if err != nil {
		return nil, err
	}

	return &GetVersionResponse{
		Version: version,
	}, nil
}
