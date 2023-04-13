package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"reflect"

	"github.com/davidbalbert/chatter/api"
	"github.com/davidbalbert/chatter/system"
	"golang.org/x/sync/errgroup"
)

// const s = `
// ospf:
//   router-id: 192.168.200.1

//   area 0:
//     interface bridge100: {}
// `

type Invocation struct {
	Func reflect.Value
	Args []reflect.Value
	C    chan reflect.Value
}

type RandProtocol struct {
	invocations chan Invocation
}

func NewRandProtocol() *RandProtocol {
	return &RandProtocol{
		invocations: make(chan Invocation),
	}
}

func (p *RandProtocol) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case inv := <-p.invocations:
			inv.C <- inv.Func.Call(inv.Args)[0]
		}
	}
}

func (p *RandProtocol) Invoke(f any, argv ...any) any {
	args := make([]reflect.Value, len(argv))
	for i, arg := range argv {
		args[i] = reflect.ValueOf(arg)
	}

	c := make(chan reflect.Value)
	p.invocations <- Invocation{
		Func: reflect.ValueOf(f),
		Args: args,
		C:    c,
	}

	ret := <-c
	return ret.Interface()
}

func (p *RandProtocol) DoGetRandInt(n int) int {
	return rand.Intn(n)
}

func (p *RandProtocol) GetRandInt(n int) int {
	return p.Invoke(p.DoGetRandInt, n).(int)
}

func (p *RandProtocol) DoGetRandString() string {
	return "hello world"
}

func (p *RandProtocol) GetRandString() string {
	return p.Invoke(p.DoGetRandString).(string)
}

// func (p *RandProtocol) RegisterCommands(a *api.API) error {
// 	var err error
// 	a.RegisterCommand("show rand", "show a random number", &err, func() string {
// 		return fmt.Sprintf("%d", p.GetRandInt(100))
// 	})
// 	a.RegisterCommand("show rand int", "show a random number", &err, func() string {
// 		return fmt.Sprintf("%d", p.GetRandInt(100))
// 	})
// 	a.RegisterCommand("show rand string", "show a random string", &err, func() string {
// 		return p.GetRandString()
// 	})
// 	return err
// }

func main() {
	fmt.Printf("Starting chatterd v0.0.1 with uid %d\n", os.Getuid())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	g, ctx := errgroup.WithContext(ctx)

	// What we need in an API server?
	// - A way to syncronously send and receive messages - DONE (see Command struct)
	// - A way to specify command hierarchy - e.g. we have to be able to specify "show" (and a help text for show)
	//   and then a way to specify that "rand" is an object that you can show

	interfaceManager := &system.InterfaceManager{}

	g.Go(func() error {
		return interfaceManager.Run(ctx)
	})

	g.Go(func() error {
		apiServer := &api.Server{InterfaceManager: interfaceManager, ShutdownFunc: cancel}
		return apiServer.ListenAndServe(ctx)
	})

	err := g.Wait()
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
