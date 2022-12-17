package main

import (
	"fmt"
	"net/netip"
	"os"
)

// https://www.rfc-editor.org/rfc/rfc2328.html

// TODO: where to put these? Maybe eventually ospf.go?

var allSPFRouters = netip.MustParseAddr("224.0.0.5")
var allDRouters = netip.MustParseAddr("224.0.0.6")

// capabilities
const (
	capE  = 1 << 1
	capMC = 1 << 2
	capNP = 1 << 3
	capEA = 1 << 4
	capDC = 1 << 5
)

func main() {
	fmt.Printf("Starting ospfd with uid %d\n", os.Getuid())

	config, err := NewConfig("192.168.200.1")
	if err != nil {
		panic(err)
	}

	if err := config.AddNetwork("192.168.105.0/24", "0.0.0.0"); err != nil {
		panic(err)
	}

	if err := config.AddInterface("bridge100", "0.0.0.0", networkPointToMultipoint, 10, 40, 5); err != nil {
		panic(err)
	}

	instance, err := NewInstance(config)
	if err != nil {
		panic(err)
	}

	instance.Run()

	select {}
}
