//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative rpc.proto

package rpc

import (
	context "context"

	"github.com/davidbalbert/chatter/config"
)

type APIService interface {
	GetVersion(ctx context.Context) (string, error)
	Shutdown(ctx context.Context) error
	GetServices(ctx context.Context) ([]config.ServiceID, error)

	GetInterfaces(ctx context.Context) ([]*Interface, error)
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

func (s *Server) GetVersion(ctx context.Context, req *GetVersionRequest) (*GetVersionReply, error) {
	version, err := s.apiService.GetVersion(ctx)
	if err != nil {
		return nil, err
	}

	return &GetVersionReply{
		Version: version,
	}, nil
}

func (s *Server) Shutdown(ctx context.Context, req *ShutdownRequest) (*ShutdownReply, error) {
	err := s.apiService.Shutdown(ctx)
	if err != nil {
		return nil, err
	}

	return &ShutdownReply{}, nil
}

func (s *Server) GetServices(ctx context.Context, req *GetServicesRequest) (*GetServicesReply, error) {
	services, err := s.apiService.GetServices(ctx)
	if err != nil {
		return nil, err
	}

	srvcs := make([]*Service, len(services))

	for i, service := range services {
		srvcs[i] = &Service{
			Type: int32(service.Type),
			Name: service.Name,
		}
	}

	return &GetServicesReply{
		Services: srvcs,
	}, nil
}

func (s *Server) GetInterfaces(ctx context.Context, req *GetInterfacesRequest) (*GetInterfacesReply, error) {
	ifaces, err := s.apiService.GetInterfaces(ctx)
	if err != nil {
		return nil, err
	}

	return &GetInterfacesReply{
		Interfaces: ifaces,
	}, nil
}
