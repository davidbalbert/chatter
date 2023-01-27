//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api.proto

package api

import (
	context "context"
	"math/rand"
	"net"

	grpc "google.golang.org/grpc"
)

const socketPath = "/tmp/ospfd.sock"

type Server struct {
	UnimplementedAPIServer
	g *grpc.Server
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	s.g = grpc.NewServer()
	RegisterAPIServer(s.g, s)
	return s.g.Serve(listener)
}

func (s *Server) GracefulStop() {
	s.g.GracefulStop()
}

func (s *Server) GetRandInt(ctx context.Context, in *Empty) (*RandInt, error) {
	return &RandInt{Value: rand.Uint32()}, nil
}

func (s *Server) GetRandString(ctx context.Context, in *Empty) (*RandString, error) {
	return &RandString{Value: "Hello, world!"}, nil
}

// import (
// 	"fmt"
// 	"strings"
// )

// type ServiceProvider interface {
// 	RegisterCommands(*API) error
// }

// type node struct {
// 	id       string
// 	help     string
// 	execute  func() string
// 	children map[string]*node
// }

// func incompleteCommand() string {
// 	return "Incomplete command"
// }

// func namespace(id, help string) *node {
// 	return &node{
// 		id:       id,
// 		help:     help,
// 		execute:  incompleteCommand,
// 		children: make(map[string]*node),
// 	}
// }

// func command(id, help string, execute func() string) *node {
// 	return &node{
// 		id:       id,
// 		help:     help,
// 		execute:  execute,
// 		children: make(map[string]*node),
// 	}
// }

// type API struct {
// 	root *node
// }

// func NewAPI() *API {
// 	return &API{
// 		root: namespace("", ""),
// 	}
// }

// func (a *API) RegisterService(s ServiceProvider) error {
// 	return s.RegisterCommands(a)
// }

// func (a *API) RegisterNamespace(path, help string, errp *error) {
// 	ids := strings.Fields(path)

// 	a.registerNode(path, namespace(ids[len(ids)-1], help), errp)
// }

// func (a *API) RegisterCommand(path, help string, errp *error, execute func() string) {
// 	ids := strings.Fields(path)

// 	a.registerNode(path, command(ids[len(ids)-1], help, execute), errp)
// }

// func (a *API) registerNode(path string, node *node, errp *error) {
// 	if *errp != nil {
// 		return
// 	}

// 	ids := strings.Fields(path)
// 	nids := len(ids)

// 	n := a.root
// 	for i, id := range ids[:nids-1] {
// 		n = n.children[id]
// 		if n == nil {
// 			*errp = fmt.Errorf("cannot register \"%s\", missing node at \"%s\"", path, strings.Join(ids[:i+1], " "))
// 			return
// 		}
// 	}

// 	id := ids[nids-1]
// 	if n.children[id] != nil {
// 		*errp = fmt.Errorf("cannot register \"%s\", node already exists", path)
// 		return
// 	}

// 	n.children[id] = node
// }

// func (a *API) ExecuteCommand(cmd string) string {
// 	ids := strings.Fields(cmd)

// 	n := a.root
// 	for _, id := range ids {
// 		n = n.children[id]
// 		if n == nil {
// 			return "Unknown command"
// 		}
// 	}

// 	return n.execute()
// }
