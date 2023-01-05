package main

import (
	"fmt"
	"os"

	"github.com/davidbalbert/ospfd/config"
)

const s = `
ospf:
  router-id: 192.168.200.1

  area 0:
    interface bridge100: {}
`

func main() {
	fmt.Printf("Starting ospfd with uid %d\n", os.Getuid())

	c, err := config.ParseConfig(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Config: %+v\n", c)
}
