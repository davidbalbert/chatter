package main

import (
	"fmt"
	"net"
	"net/netip"
	"time"

	"golang.org/x/net/ipv4"
)

type networkType int

const (
	networkBroadcast networkType = iota
	networkPointToPoint
	networkPointToMultipoint
	networkNonBroadcastMultipleAccess
	networkVirtualLink
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
	areaID             netip.Addr
	HelloInteral       uint16
	RouterDeadInterval uint32
	RxmtInterval       uint16
	neighbors          map[netip.Addr]*Neighbor

	instance *Instance
	conn     *ipv4.RawConn

	rm chan *Neighbor
}

func NewInterface(inst *Instance, addr netip.Prefix, netif *net.Interface, ifconfig *InterfaceConfig, netconfig *NetworkConfig) *Interface {
	iface := &Interface{
		networkType:        ifconfig.NetworkType,
		netif:              netif,
		Address:            addr.Addr(),
		Prefix:             netconfig.Network,
		areaID:             netconfig.AreaID,
		HelloInteral:       ifconfig.HelloInterval,
		RouterDeadInterval: ifconfig.DeadInterval,
		RxmtInterval:       ifconfig.RxmtInterval,
		neighbors:          make(map[netip.Addr]*Neighbor),
		instance:           inst,
	}

	return iface
}

func (iface *Interface) HelloDuration() time.Duration {
	return time.Duration(iface.HelloInteral) * time.Second
}

func (iface *Interface) RxmtDuration() time.Duration {
	return time.Duration(iface.RxmtInterval) * time.Second
}

func (iface *Interface) send(dst netip.Addr, p Packet) {
	fmt.Printf("Sending %s\n", p)

	b := p.encode()

	ip := &ipv4.Header{
		Version:  ipv4.Version,
		Len:      ipv4.HeaderLen,
		TOS:      0xc0,
		TotalLen: ipv4.HeaderLen + len(b),
		TTL:      1,
		Protocol: 89,
		Dst:      to4(dst),
	}

	iface.conn.WriteTo(ip, b, nil)
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

		if p.AreaID() != iface.areaID {
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

// TODO: can we come up with a good name for an interface with the following signature, and then replace
// the two arguments to neighborKey with a single argument of this type?
// type Hmm interface {
// 	NeighborAddr() netip.Addr
// 	NeighborID() netip.Addr
// }

func (iface *Interface) neighborKey(src, routerID netip.Addr) netip.Addr {
	t := iface.networkType
	if t == networkBroadcast || t == networkPointToMultipoint || t == networkNonBroadcastMultipleAccess {
		return src
	} else {
		return routerID
	}
}

func (iface *Interface) handleHello(h *helloPacket) {
	if iface.networkType != networkPointToPoint && h.netmaskBits() != iface.Prefix.Bits() {
		return
	}

	if h.helloInterval != iface.HelloInteral || h.routerDeadInterval != iface.RouterDeadInterval {
		return
	}

	// TODO: E-bit of interface should match E-bit of the hello packet.

	key := iface.neighborKey(h.src, h.routerID)
	neighbor, ok := iface.neighbors[key]
	if !ok {
		neighbor = newNeighbor(iface, h)
		go neighbor.run()
		iface.neighbors[key] = neighbor
	}

	// var routerPriorityChanged bool
	// var dRouterChanged bool
	// var bdRouterChanged bool
	if iface.networkType == networkBroadcast || iface.networkType == networkPointToMultipoint || iface.networkType == networkNonBroadcastMultipleAccess {
		if neighbor.routerPriority != h.routerPriority {
			neighbor.routerPriority = h.routerPriority
			// routerPriorityChanged = true
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

	neighbor.sendEvent(neHelloReceived)

	var found bool
	for _, routerID := range h.neighbors {
		if routerID == iface.instance.routerID {
			found = true
			break
		}
	}

	if !found {
		neighbor.sendEvent(ne1WayReceived)
		return
	} else {
		neighbor.sendEvent(ne2WayReceived)
	}

	// if routerPriorityChanged {
	// 	// TODO: interface state machine
	// 	iface.scheduleEvent(ieNeighborChange)
	// }

	// TODO: "If the neighbor is both declaring itself to be Designated"
	// in RFC 2328

	// TODO: "If the neighbor is declaring itself to be Backup Designated"
	// in RFC 2328
}

func (iface *Interface) handleDatabaseDescription(dd *databaseDescriptionPacket) {
	if int(dd.interfaceMTU) > iface.netif.MTU {
		fmt.Printf("Received database description packet with MTU %v, but interface MTU is %v\n", dd.interfaceMTU, iface.netif.MTU)
		return
	}

	key := iface.neighborKey(dd.src, dd.routerID)
	neighbor, ok := iface.neighbors[key]
	if !ok {
		fmt.Printf("Received database description packet from unknown neighbor addr=%v router_id=%v\n", dd.src, dd.routerID)
		return
	}

	neighbor.sendPacket(dd)
}

func (iface *Interface) handleLinkStateRequest(lsr *linkStateRequestPacket) {
	key := iface.neighborKey(lsr.src, lsr.routerID)
	neighbor, ok := iface.neighbors[key]
	if !ok {
		fmt.Printf("Received link state request packet from unknown neighbor addr=%v router_id=%v\n", lsr.src, lsr.routerID)
		return
	}

	neighbor.sendPacket(lsr)
}

func (iface *Interface) handleLinkStateUpdate(lsu *linkStateUpdatePacket) {
	// noop
}

func (iface *Interface) handleLinkStateAcknowledgment(lsack *linkStateAcknowledgmentPacket) {
	// noop
}

func (iface *Interface) routerID() netip.Addr {
	return iface.instance.routerID
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
	helloTick := tickImmediately(iface.HelloDuration())

	go iface.receive(r)

	for {
		select {
		case p := <-r:
			p.handleOn(iface)
		case <-helloTick:
			iface.send(allSPFRouters, newHello(iface))
		case neighbor := <-iface.rm:
			neighbor.shutdown()
			delete(iface.neighbors, iface.neighborKey(neighbor.addr, neighbor.neighborID))
		}
	}
}
