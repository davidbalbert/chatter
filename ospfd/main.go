package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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

func main() {
	fmt.Printf("Starting ospfd with uid %d\n", os.Getuid())

	conf, err := config.ParseConfig(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(conf)

	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	handle := func(w io.Writer, cmd string) {
		if strings.TrimSpace(cmd) == "shutdown" {
			cancel()
			return
		}

		w.Write([]byte("-> "))
		w.Write([]byte(cmd))
	}

	server := vty.NewServer()
	server.HandleFunc(handle)

	g.Go(func() error {
		return server.ListenAndServe(ctx)
	})

	g.Go(func() error {
		<-ctx.Done()
		return server.Shutdown()
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
