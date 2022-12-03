package main

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"time"

	"golang.org/x/net/ipv4"
)

// https://www.rfc-editor.org/rfc/rfc2328.html

var allSPFRouters = netip.MustParseAddr("224.0.0.5")
var allDRouters = netip.MustParseAddr("224.0.0.6")

func toNetAddr(addr netip.Addr) net.Addr {
	return &net.IPAddr{IP: addr.AsSlice()}
}

func to4(addr netip.Addr) []byte {
	b := addr.As4()
	return b[:]
}

func mustAddrFromSlice(b []byte) netip.Addr {
	addr, ok := netip.AddrFromSlice(b)
	if !ok {
		panic("mustAddrFromSlice: slice should be either 4 or 16 bytes, but got " + fmt.Sprint(len(b)))
	}
	return addr
}

func tickImmediately(d time.Duration) <-chan time.Time {
	c := make(chan time.Time)

	go func() {
		c <- time.Now()
		for t := range time.Tick(d) {
			c <- t
		}
	}()

	return c
}

type networkType int

const (
	networkBroadcast networkType = iota
	networkPointToPoint
	networkPointToMultipoint
	networkNonBroadcastMultipleAccess
)

func (t networkType) String() string {
	switch t {
	case networkBroadcast:
		return "broadcast"
	case networkPointToPoint:
		return "point-to-point"
	case networkPointToMultipoint:
		return "point-to-multipoint"
	case networkNonBroadcastMultipleAccess:
		return "NBMA"
	default:
		return "unknown"
	}
}

type Interface struct {
	networkType
	netif              *net.Interface
	Address            netip.Addr
	Prefix             netip.Prefix
	AreaID             netip.Addr
	HelloInteral       uint16
	RouterDeadInterval uint32
	neighbors          map[netip.Addr]*Neighbor

	instance *Instance
	conn     *ipv4.RawConn

	downNeighbors chan *Neighbor
}

func (iface *Interface) HelloDuration() time.Duration {
	return time.Duration(iface.HelloInteral) * time.Second
}

func (iface *Interface) send(c chan Packet) {
	for {
		p := <-c
		fmt.Printf("Sending %s\n", p)

		b := p.encode()

		ip := &ipv4.Header{
			Version:  ipv4.Version,
			Len:      ipv4.HeaderLen,
			TOS:      0xc0,
			TotalLen: ipv4.HeaderLen + len(b),
			TTL:      1,
			Protocol: 89,
			Dst:      to4(allSPFRouters),
		}

		iface.conn.WriteTo(ip, b, nil)
	}
}

func (iface *Interface) receive(c chan Packet) {
	for {
		buf := make([]byte, 1500)
		ip, payload, cm, err := iface.conn.ReadFrom(buf)
		if err != nil {
			fmt.Printf("Error reading from %s: %s\n", iface.netif.Name, err)
			continue
		}

		// Ignore packets not on our interface. Go makes us listen on 0.0.0.0 if we want
		// to receive any multicast packets, so we need to filter out packets for
		// other interfaces.
		if cm.IfIndex != iface.netif.Index {
			continue
		}

		// Ignore packets not on our subnet. We've eliminated other interfaces, but not
		// other subnets.
		if iface.networkType != networkPointToPoint && !iface.Prefix.Contains(mustAddrFromSlice(ip.Src).Unmap()) {
			continue
		}

		p, err := decodePacket(ip, payload)
		if err != nil {
			fmt.Printf("Error decoding header: %v\n", err)
			continue
		}

		if p.AreaID() != iface.AreaID {
			continue
		}

		// RFC2328 section 8.2 says:
		//   o   Packets whose IP destination is AllDRouters should only be
		//       accepted if the state of the receiving interface is DR or
		//       Backup (see Section 9.1).
		//
		// This should always be true because we only listen on AllDRouters when
		// we're the DR or Backup.

		// TODO: check authentication

		c <- p
	}
}

func (iface *Interface) neighborId(h *Hello) netip.Addr {
	t := iface.networkType
	if t == networkBroadcast || t == networkPointToMultipoint || t == networkNonBroadcastMultipleAccess {
		return h.src
	} else {
		return h.routerID
	}
}

func (iface *Interface) handleHello(h *Hello) {
	if iface.networkType != networkPointToPoint && h.netmaskBits() != iface.Prefix.Bits() {
		return
	}

	if h.helloInterval != iface.HelloInteral || h.routerDeadInterval != iface.RouterDeadInterval {
		return
	}

	// TODO: E-bit of interface should match E-bit of the hello packet.

	id := iface.neighborId(h)
	neighbor, ok := iface.neighbors[id]
	if !ok {
		neighbor = NewNeighbor(id, iface, h)
		go neighbor.run()
		iface.neighbors[id] = neighbor
	}

	var routerPriorityChanged bool
	// var dRouterChanged bool
	// var bdRouterChanged bool
	if iface.networkType == networkBroadcast || iface.networkType == networkPointToMultipoint || iface.networkType == networkNonBroadcastMultipleAccess {
		if neighbor.routerPriority != h.routerPriority {
			neighbor.routerPriority = h.routerPriority
			routerPriorityChanged = true
		}

		if neighbor.dRouter != h.dRouter {
			neighbor.dRouter = h.dRouter
			// dRouterChanged = true
		}

		if neighbor.bdRouter != h.bdRouter {
			neighbor.bdRouter = h.bdRouter
			// bdRouterChanged = true
		}
	}

	neighbor.executeEvent(neHelloReceived)

	var found bool
	for _, routerID := range h.neighbors {
		if routerID == iface.instance.RouterID {
			found = true
			break
		}
	}

	if !found {
		neighbor.executeEvent(ne1WayReceived)
		return
	} else {
		neighbor.executeEvent(ne2WayReceived)
	}

	if routerPriorityChanged {
		// TODO: interface state machine
		// iface.scheduleEvent(ieNeighborChange)
	}

	// TODO: "If the neighbor is both declaring itself to be Designated"
	// in RFC 2328

	// TODO: "If the neighbor is declaring itself to be Backup Designated"
	// in RFC 2328
}

func (iface *Interface) removeNeighbor(n *Neighbor) {
	iface.downNeighbors <- n
}

func (iface *Interface) run() {
	conn, err := net.ListenPacket("ip4:ospf", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	raw, err := ipv4.NewRawConn(conn)
	if err != nil {
		panic(err)
	}

	if err := raw.JoinGroup(iface.netif, toNetAddr(allSPFRouters)); err != nil {
		panic(err)
	}
	defer raw.LeaveGroup(iface.netif, toNetAddr(allSPFRouters))

	if err := raw.SetMulticastInterface(iface.netif); err != nil {
		panic(err)
	}

	if err := raw.SetMulticastLoopback(false); err != nil {
		panic(err)
	}

	if err := raw.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		panic(err)
	}

	iface.conn = raw

	r := make(chan Packet)
	w := make(chan Packet)
	hello := tickImmediately(iface.HelloDuration())

	go iface.receive(r)
	go iface.send(w)

	for {
		select {
		case p := <-r:
			p.handleOn(iface)
		case neighbor := <-iface.downNeighbors:
			neighbor.stop()
			delete(iface.neighbors, neighbor.source)

		case <-hello:
			// w <- &Packet{
			// 	Header: Header{
			// 		Type:     TypeHello,
			// 		Length:   44,
			// 		RouterID: iface.instance.RouterID,
			// 		AreaID:   iface.AreaID,
			// 		AuthType: 0,
			// 		AuthData: 0,
			// 	},
			// 	Content: Hello{
			// 		NetworkMask:     net.CIDRMask(iface.Prefix.Bits(), 32),
			// 		HelloInterval:   10,
			// 		Options:         0x2,
			// 		RtrPriority:     1,
			// 		RtrDeadInterval: 40,
			// 		DRouter:         netip.IPv4Unspecified(),
			// 		BDRouter:        netip.IPv4Unspecified(),
			// 	},
			// }
		}
	}
}

type Instance struct {
	RouterID   netip.Addr
	Interfaces []Interface
}

func NewInstance(c *Config) (*Instance, error) {
	inst := &Instance{
		RouterID: c.RouterID,
	}

	for _, ifconfig := range c.Interfaces {
		netif, err := net.InterfaceByName(ifconfig.Name)
		if err != nil {
			return nil, err
		}

		addrs, err := netif.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			interfacePrefix, err := netip.ParsePrefix(addr.String())
			if err != nil {
				return nil, fmt.Errorf("error parsing prefix for %s: %w", netif.Name, err)
			}

			for _, network := range c.Networks {
				if interfacePrefix.Masked() == network.Network {
					inst.Interfaces = append(inst.Interfaces, Interface{
						networkType:        ifconfig.NetworkType,
						netif:              netif,
						Address:            interfacePrefix.Addr(),
						Prefix:             network.Network,
						AreaID:             network.AreaID,
						HelloInteral:       ifconfig.HelloInterval,
						RouterDeadInterval: ifconfig.DeadInterval,
						neighbors:          make(map[netip.Addr]*Neighbor),
						instance:           inst,
						downNeighbors:      make(chan *Neighbor),
					})
				}
			}
		}
	}

	return inst, nil
}

func (inst *Instance) Run() {
	for _, iface := range inst.Interfaces {
		go iface.run()
	}

	select {}
}

func main() {
	fmt.Printf("Starting ospfd with uid %d\n", os.Getuid())

	config, err := NewConfig("192.168.200.1")
	if err != nil {
		panic(err)
	}

	if err := config.AddNetwork("192.168.105.0/24", "0.0.0.0"); err != nil {
		panic(err)
	}

	config.AddInterface("bridge100", networkPointToMultipoint, 10, 40)

	instance, err := NewInstance(config)
	if err != nil {
		panic(err)
	}

	instance.Run()
}
