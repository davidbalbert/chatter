package main

import (
	"fmt"
	"os"

	"github.com/davidbalbert/ospfd/config"
	"gopkg.in/yaml.v3"
)

const s = `
ospf:
  router-id: 192.168.200.1
  cost: 10
  hello-interval: 5
  dead-interval: 15

  area 0:
    cost: 20

    interface eth0:
      cost: 30
    
    interface eth1: {}

  area 1:
    hello-interval: 20
    interface eth2:
      cost: 40

  area 1.1.1.1:
    dead-interval: 30
    interface eth3:
      cost: 50
`

func main() {
	fmt.Printf("Starting ospfd with uid %d\n", os.Getuid())

	c, err := config.ParseConfig(s)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Config: %+v\n", c)

	s2, err := yaml.Marshal(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("%s", s2)
}
