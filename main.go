package main

import (
	"bytes"
	"encoding/binary"
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

type messageType uint8

const (
	TypeHello messageType = iota + 1
	TypeDatabaseDescription
	TypeLinkStateRequest
	TypeLinkStateUpdate
	TypeLinkStateAcknowledgement
)

func (t messageType) String() string {
	switch t {
	case TypeHello:
		return "Hello"
	case TypeDatabaseDescription:
		return "Database Description"
	case TypeLinkStateRequest:
		return "Link State Request"
	case TypeLinkStateUpdate:
		return "Link State Update"
	case TypeLinkStateAcknowledgement:
		return "Link State Acknowledgement"
	default:
		return "Unknown"
	}
}

type Header struct {
	Type     messageType
	Length   uint16
	RouterID netip.Addr
	AreaID   netip.Addr
	Checksum uint16
	AuthType uint16
	AuthData uint64
}

func (header *Header) encodeTo(data []byte) ([]byte, error) {
	data[0] = 2
	data[1] = uint8(header.Type)
	binary.BigEndian.PutUint16(data[2:4], header.Length)
	copy(data[4:8], to4(header.RouterID))
	copy(data[8:12], to4(header.AreaID))

	// Skip CHecksum - data[12:14]

	binary.BigEndian.PutUint16(data[14:16], header.AuthType)
	binary.BigEndian.PutUint64(data[16:24], header.AuthData)

	return data, nil
}

func (h *Header) String() string {
	return fmt.Sprintf("OSPFv2 %s router=%s area=%s", h.Type, h.RouterID, h.AreaID)
}

type Packet interface {
	encode() []byte
	handle(*Interface) error
}

type Hello struct {
	Header

	NetworkMask     net.IPMask
	HelloInterval   uint16
	Options         uint8
	RtrPriority     uint8
	RtrDeadInterval uint32
	DRouter         netip.Addr
	BDRouter        netip.Addr
	Neighbors       []netip.Addr
}

func (hello *Hello) String() string {
	var b bytes.Buffer

	fmt.Fprint(&b, hello.Header.String())
	fmt.Fprintf(&b, " mask=%s interval=%d options=0x%x priority=%d dead=%d dr=%s bdr=%s", net.IP(hello.NetworkMask), hello.HelloInterval, hello.Options, hello.RtrPriority, hello.RtrDeadInterval, hello.DRouter, hello.BDRouter)

	for _, n := range hello.Neighbors {
		fmt.Fprintf(&b, "\n  neighbor=%s", n)
	}

	return b.String()
}

func (hello *Hello) encode() []byte {
	hello.Length = 44 + uint16(len(hello.Neighbors)*4)

	data := make([]byte, hello.Length)
	hello.Header.encodeTo(data)

	copy(data[24:28], hello.NetworkMask)
	binary.BigEndian.PutUint16(data[28:30], hello.HelloInterval)
	data[30] = hello.Options
	data[31] = hello.RtrPriority
	binary.BigEndian.PutUint32(data[32:36], hello.RtrDeadInterval)
	copy(data[36:40], to4(hello.DRouter))
	copy(data[40:44], to4(hello.BDRouter))
	for i, neighbor := range hello.Neighbors {
		copy(data[44+i*4:48+i*4], to4(neighbor))
	}

	hello.Checksum = checksum(data[0:16], data[24:])
	binary.BigEndian.PutUint16(data[12:14], hello.Checksum)

	return data
}

func (hello *Hello) handle(iface *Interface) error {
	fmt.Println(hello)

	return nil
}

func decodePacket(data []byte) (Packet, error) {
	if len(data) < 24 {
		return nil, fmt.Errorf("packet too short")
	}

	version := data[0]
	if version != 2 {
		return nil, fmt.Errorf("unsupported OSPF version %d", version)
	}

	var h Header

	h.Type = messageType(data[1])
	h.Length = binary.BigEndian.Uint16(data[2:4])
	h.RouterID = mustAddrFromSlice(data[4:8])
	h.AreaID = mustAddrFromSlice(data[8:12])
	h.Checksum = binary.BigEndian.Uint16(data[12:14])
	h.AuthType = binary.BigEndian.Uint16(data[14:16])
	h.AuthData = binary.BigEndian.Uint64(data[16:24])

	if len(data) != int(h.Length) {
		return nil, fmt.Errorf("packet length mismatch")
	}

	if checksum(data) != 0 {
		return nil, fmt.Errorf("packet checksum mismatch")
	}

	switch h.Type {
	case TypeHello:
		if h.Length < 44 {
			return nil, fmt.Errorf("hello packet too short")
		}

		var hello Hello
		hello.Header = h
		hello.NetworkMask = data[24:28]
		hello.HelloInterval = binary.BigEndian.Uint16(data[28:30])
		hello.Options = data[30]
		hello.RtrPriority = data[31]
		hello.RtrDeadInterval = binary.BigEndian.Uint32(data[32:36])
		hello.DRouter = mustAddrFromSlice(data[36:40])
		hello.BDRouter = mustAddrFromSlice(data[40:44])

		for i := 44; i < int(h.Length); i += 4 {
			hello.Neighbors = append(hello.Neighbors, mustAddrFromSlice(data[i:i+4]))
		}

		return &hello, nil
	case TypeDatabaseDescription:
		fallthrough
	case TypeLinkStateRequest:
		fallthrough
	case TypeLinkStateUpdate:
		fallthrough
	case TypeLinkStateAcknowledgement:
		return nil, fmt.Errorf("unsupported OSPF packet type %s", h.Type)
	default:
		return nil, fmt.Errorf("unknown OSPF packet type %d", h.Type)
	}
}

func checksum(data ...[]byte) uint16 {
	var sum uint32
	for _, d := range data {
		l := len(d)
		for i := 0; i < l; i += 2 {
			if i+1 < l {
				sum += uint32(d[i])<<8 | uint32(d[i+1])
			} else {
				sum += uint32(d[i]) << 8
			}
		}
	}

	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16

	return ^uint16(sum)
}

type Neighbor struct {
	RouterID netip.Addr
	IP       netip.Addr
}

type Interface struct {
	netif        *net.Interface
	Address      netip.Addr
	Prefix       netip.Prefix
	AreaID       netip.Addr
	HelloInteral uint16
	DeadInterval uint32
	neighbors    map[netip.Addr]Neighbor

	instance *Instance
	conn     *ipv4.RawConn
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
		_, payload, cm, err := iface.conn.ReadFrom(buf)
		if err != nil {
			panic(err)
		}

		// Ignore packets not on our interface
		if cm.IfIndex != iface.netif.Index {
			continue
		}

		p, err := decodePacket(payload)
		if err != nil {
			fmt.Printf("Error decoding header: %v\n", err)
			continue
		}

		c <- p
	}
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
			p.handle(iface)

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

	for _, iface := range c.Interfaces {
		netif, err := net.InterfaceByName(iface.Name)
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
						netif:        netif,
						Address:      interfacePrefix.Addr(),
						Prefix:       network.Network,
						AreaID:       network.AreaID,
						HelloInteral: iface.HelloInterval,
						DeadInterval: iface.DeadInterval,
						neighbors:    make(map[netip.Addr]Neighbor),
						instance:     inst,
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

	config.AddInterface("bridge100", 10, 40)

	instance, err := NewInstance(config)
	if err != nil {
		panic(err)
	}

	instance.Run()
}
