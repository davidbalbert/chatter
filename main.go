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

func (h *Header) String() string {
	return fmt.Sprintf("OSPFv2 %s len=%d router=%s area=%s checksum=0x%x authType=%d authData=%d", h.Type, h.Length, h.RouterID, h.AreaID, h.Checksum, h.AuthType, h.AuthData)
}

type Hello struct {
	NetworkMask     net.IPMask
	HelloInterval   uint16
	Options         uint8
	RtrPriority     uint8
	RtrDeadInterval uint32
	DRouter         netip.Addr
	BDRouter        netip.Addr
	Neighbors       []netip.Addr
}

type Packet struct {
	Header
	Content any
	data    []byte
}

func decodePacket(data []byte) (*Packet, error) {
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

	var p Packet
	p.Header = h
	p.data = data

	switch h.Type {
	case TypeHello:
		if h.Length < 44 {
			return nil, fmt.Errorf("hello packet too short")
		}

		var hello Hello
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

		p.Content = hello
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

	return &p, nil
}

func (p *Packet) updateLength() error {
	switch p.Type {
	case TypeHello:
		hello := p.Content.(Hello)
		p.Length = 44 + uint16(len(hello.Neighbors)*4)
	case TypeDatabaseDescription:
		fallthrough
	case TypeLinkStateRequest:
		fallthrough
	case TypeLinkStateUpdate:
		fallthrough
	case TypeLinkStateAcknowledgement:
		return fmt.Errorf("unsupported OSPF packet type %s", p.Type)
	default:
		return fmt.Errorf("unknown OSPF packet type %d", p.Type)
	}

	return nil
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

func (p *Packet) encode() ([]byte, error) {
	if err := p.updateLength(); err != nil {
		return nil, err
	}

	b := make([]byte, p.Length)

	b[0] = 2
	b[1] = uint8(p.Type)
	binary.BigEndian.PutUint16(b[2:4], p.Length)
	copy(b[4:8], to4(p.RouterID))
	copy(b[8:12], to4(p.AreaID))
	binary.BigEndian.PutUint16(b[12:14], p.Checksum) // TODO!
	binary.BigEndian.PutUint16(b[14:16], p.AuthType)
	binary.BigEndian.PutUint64(b[16:24], p.AuthData)

	switch p.Type {
	case TypeHello:
		hello := p.Content.(Hello)
		copy(b[24:28], hello.NetworkMask)
		binary.BigEndian.PutUint16(b[28:30], hello.HelloInterval)
		b[30] = hello.Options
		b[31] = hello.RtrPriority
		binary.BigEndian.PutUint32(b[32:36], hello.RtrDeadInterval)
		copy(b[36:40], to4(hello.DRouter))
		copy(b[40:44], to4(hello.BDRouter))
		for i, neighbor := range hello.Neighbors {
			copy(b[44+i*4:48+i*4], to4(neighbor))
		}
	case TypeDatabaseDescription:
		fallthrough
	case TypeLinkStateRequest:
		fallthrough
	case TypeLinkStateUpdate:
		fallthrough
	case TypeLinkStateAcknowledgement:
		return nil, fmt.Errorf("unsupported OSPF packet type %s", p.Type)
	default:
		return nil, fmt.Errorf("unknown OSPF packet type %d", p.Type)
	}

	p.data = b
	p.Checksum = checksum(p.data[0:16], p.data[24:])
	binary.BigEndian.PutUint16(b[12:14], p.Checksum)

	return b, nil
}

func (p *Packet) String() string {
	var b bytes.Buffer

	fmt.Fprintf(&b, "OSPFv2 %s router=%s area=%s", p.Type, p.RouterID, p.AreaID)

	switch p.Type {
	case TypeHello:
		hello := p.Content.(Hello)
		fmt.Fprintf(&b, " mask=%s interval=%d options=0x%x priority=%d dead=%d dr=%s bdr=%s", net.IP(hello.NetworkMask), hello.HelloInterval, hello.Options, hello.RtrPriority, hello.RtrDeadInterval, hello.DRouter, hello.BDRouter)
		for _, n := range hello.Neighbors {
			fmt.Fprintf(&b, "\n  neighbor=%s", n)
		}
	default:

	}

	return b.String()
}

func listen(conn *ipv4.RawConn, c chan *Packet) {
	for {
		buf := make([]byte, 1500)
		_, payload, _, err := conn.ReadFrom(buf)
		if err != nil {
			panic(err)
		}

		p, err := decodePacket(payload)
		if err != nil {
			fmt.Printf("Error decoding header: %v\n", err)
			continue
		}

		c <- p
	}
}

func send(conn *ipv4.RawConn, c chan *Packet) {
	for {
		p := <-c
		fmt.Printf("Sending %s\n", p)

		b, err := p.encode()
		if err != nil {
			fmt.Printf("Error encoding packet: %v\n", err)
			continue
		}

		ip := &ipv4.Header{
			Version:  ipv4.Version,
			Len:      ipv4.HeaderLen,
			TOS:      0xc0,
			TotalLen: ipv4.HeaderLen + len(b),
			TTL:      1,
			Protocol: 89,
			Dst:      to4(allSPFRouters),
		}

		conn.WriteTo(ip, b, nil)
	}
}

type NetworkConfig struct {
	Network netip.Prefix
	AreaID  netip.Addr
}

type InterfaceConfig struct {
	Name          string
	HelloInterval uint16
	DeadInterval  uint32
}

type Config struct {
	RouterID   netip.Addr
	Networks   []NetworkConfig
	Interfaces []InterfaceConfig
}

func NewConfig(routerID string) (*Config, error) {
	addr, err := netip.ParseAddr(routerID)
	if err != nil {
		return nil, fmt.Errorf("error parsing router id: %w", err)
	}

	return &Config{
		RouterID: addr,
	}, nil
}

func (c *Config) AddNetwork(network, areaID string) error {
	net, err := netip.ParsePrefix(network)
	if err != nil {
		return fmt.Errorf("error parsing network prefix: %w", err)
	}

	if net.Addr().Is6() {
		return fmt.Errorf("ipv6 networks are not supported")
	}

	addr, err := netip.ParseAddr(areaID)
	if err != nil {
		return fmt.Errorf("error parsing area id: %w", err)
	}

	if addr.Is6() {
		return fmt.Errorf("ipv6 area ids are not supported")
	}

	c.Networks = append(c.Networks, NetworkConfig{
		Network: net,
		AreaID:  addr,
	})

	return nil
}

func (c *Config) AddInterface(name string, helloInterval uint16, deadInterval uint32) {
	c.Interfaces = append(c.Interfaces, InterfaceConfig{
		Name:          name,
		HelloInterval: helloInterval,
		DeadInterval:  deadInterval,
	})
}

type Interface struct {
	Interface    *net.Interface
	Address      netip.Addr
	Prefix       netip.Prefix
	AreaID       netip.Addr
	HelloInteral uint16
	DeadInterval uint32
}

func (iface *Interface) HelloDuration() time.Duration {
	return time.Duration(iface.HelloInteral) * time.Second
}

type Neighbor struct {
	RouterID  netip.Addr
	IP        netip.Addr
	Interface *Interface
}

type Instance struct {
	RouterID   netip.Addr
	Interfaces []Interface
	Neighbors  map[string]Neighbor
}

func NewInstance(c *Config) (*Instance, error) {
	i := &Instance{
		RouterID:  c.RouterID,
		Neighbors: make(map[string]Neighbor),
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
			prefix, err := netip.ParsePrefix(addr.String())
			if err != nil {
				return nil, fmt.Errorf("error parsing prefix for %s: %w", netif.Name, err)
			}

			for _, net := range c.Networks {
				if net.Network == prefix.Masked() {
					i.Interfaces = append(i.Interfaces, Interface{
						Interface:    netif,
						Address:      prefix.Addr(),
						Prefix:       net.Network,
						AreaID:       net.AreaID,
						HelloInteral: iface.HelloInterval,
						DeadInterval: iface.DeadInterval,
					})
				}
			}
		}
	}

	return i, nil
}

func (inst *Instance) Run() {
	if len(inst.Interfaces) == 0 {
		panic("no interfaces configured")
	}

	// TODO: listen on all interfaces
	iface := inst.Interfaces[0]

	conn, err := net.ListenPacket("ip4:ospf", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	raw, err := ipv4.NewRawConn(conn)
	if err != nil {
		panic(err)
	}

	if err := raw.JoinGroup(iface.Interface, toNetAddr(allSPFRouters)); err != nil {
		panic(err)
	}
	defer raw.LeaveGroup(iface.Interface, toNetAddr(allSPFRouters))

	if err := raw.SetMulticastInterface(iface.Interface); err != nil {
		panic(err)
	}

	if err := raw.SetMulticastLoopback(false); err != nil {
		panic(err)
	}

	r := make(chan *Packet)
	w := make(chan *Packet)
	hello := time.Tick(iface.HelloDuration())

	go listen(raw, r)
	go send(raw, w)

	for {
		select {
		case p := <-r:
			fmt.Println(p)
		case <-hello:
			w <- &Packet{
				Header: Header{
					Type:     TypeHello,
					Length:   44,
					RouterID: inst.RouterID,
					AreaID:   iface.AreaID,
					AuthType: 0,
					AuthData: 0,
				},
				Content: Hello{
					NetworkMask:     net.CIDRMask(iface.Prefix.Bits(), 32),
					HelloInterval:   10,
					Options:         0x2,
					RtrPriority:     1,
					RtrDeadInterval: 40,
					DRouter:         netip.IPv4Unspecified(),
					BDRouter:        netip.IPv4Unspecified(),
				},
			}
		}
	}

}

func main() {
	fmt.Printf("Starting ospfd with uid %d\n", os.Getuid())

	config, err := NewConfig("192.168.105.1")
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
