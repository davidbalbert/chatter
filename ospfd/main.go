package main

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"

	"github.com/davidbalbert/ospfd/api"
	"github.com/davidbalbert/ospfd/config"
	"github.com/davidbalbert/ospfd/vty"
	"golang.org/x/sync/errgroup"
)

const s = `
ospf:
  router-id: 192.168.200.1

  area 0:
    interface bridge100: {}
`

type invocation struct {
	f func(*RandProtocol) string
	c chan string
}

type RandProtocol struct {
	invocations chan invocation
}

func NewRandProtocol() *RandProtocol {
	return &RandProtocol{
		invocations: make(chan invocation),
	}
}

func (p *RandProtocol) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case inv := <-p.invocations:
			inv.c <- inv.f(p)
		}
	}
}

func (p *RandProtocol) randInt() string {
	return fmt.Sprintf("%d", rand.Intn(100))
}

func (p *RandProtocol) randString() string {
	return "hello world"
}

func (p *RandProtocol) executeCommand(f func(*RandProtocol) string) string {
	c := make(chan string)
	p.invocations <- invocation{f, c}
	return <-c
}

func (p *RandProtocol) RegisterCommands(a *api.API) error {
	var err error
	a.RegisterCommand("show rand", "show a random number", &err, func() string {
		return p.executeCommand((*RandProtocol).randInt)
	})
	a.RegisterCommand("show rand int", "show a random number", &err, func() string {
		return p.executeCommand((*RandProtocol).randInt)
	})
	a.RegisterCommand("show rand string", "show a random string", &err, func() string {
		return p.executeCommand((*RandProtocol).randString)
	})
	return err
}

func main() {
	fmt.Printf("Starting ospfd with uid %d\n", os.Getuid())

	_, err := config.ParseConfig(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx, _ := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// What we need in an API server?
	// - A way to syncronously send and receive messages - DONE (see Command struct)
	// - A way to specify command hierarchy - e.g. we have to be able to specify "show" (and a help text for show)
	//   and then a way to specify that "rand" is an object that you can show

	rand := NewRandProtocol()

	api := api.NewAPI()
	api.RegisterNamespace("show", "show running system information", &err)
	if err != nil {
		fmt.Printf("error registering namespace: %v\n", err)
		os.Exit(1)
	}

	err = api.RegisterService(rand)
	if err != nil {
		fmt.Printf("error registering service: %v\n", err)
		os.Exit(1)
	}

	g.Go(func() error {
		return rand.Run(ctx)
	})

	handle := func(w io.Writer, cmd string) {
		resp := api.ExecuteCommand(cmd)
		w.Write([]byte(resp))

		// switch cmd {
		// case "shutdown":
		// 	cancel()
		// case "show rand":
		// 	resp := rand.ExecuteCommand(cmd)
		// 	w.Write([]byte(resp))
		// default:
		// 	w.Write([]byte("unknown command"))
		// }
	}

	server := &vty.Server{}
	server.HandleFunc(handle)

	g.Go(func() error {
		return server.ListenAndServe(ctx)
	})

	err = g.Wait()
	if err != nil {
		fmt.Printf("x: %v\n", err)
		os.Exit(1)
	}

	// Create a context
	// Start VTY.Run() in a goroutine (pass it a context)
	// Create something else in a goroutine (something to stand in for the OSPF daemon)
	// Create a console client
	// Find a way forward requests from the console server to the daemon, and then for the daemon to pass a response back
	// Select?
}
