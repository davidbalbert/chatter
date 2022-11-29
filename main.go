package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/ipv4"
)

// https://www.rfc-editor.org/rfc/rfc2328.html

var allSPFRouters = net.IPAddr{IP: net.ParseIP("224.0.0.5")}
var allDRouters = net.IPAddr{IP: net.ParseIP("224.0.0.6")}
var helloInterval = 10 * time.Second

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
	RouterID net.IP
	AreaID   net.IP
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
	DRouter         net.IP
	BDRouter        net.IP
	Neighbors       []net.IP
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
	h.RouterID = data[4:8]
	h.AreaID = data[8:12]
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
		hello.DRouter = data[36:40]
		hello.BDRouter = data[40:44]

		for i := 44; i < int(h.Length); i += 4 {
			hello.Neighbors = append(hello.Neighbors, data[i:i+4])
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
	if p.data != nil {
		return p.data, nil
	}

	if err := p.updateLength(); err != nil {
		return nil, err
	}

	p.Checksum = checksum(p.data[0:16], p.data[24:])

	b := make([]byte, p.Length)

	b[0] = 2
	b[1] = uint8(p.Type)
	binary.BigEndian.PutUint16(b[2:4], p.Length)
	copy(b[4:8], p.RouterID)
	copy(b[8:12], p.AreaID)
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
		copy(b[36:40], hello.DRouter)
		copy(b[40:44], hello.BDRouter)
		for i, neighbor := range hello.Neighbors {
			copy(b[44+i*4:48+i*4], neighbor)
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

func listen(conn net.PacketConn, c chan *Packet) {
	for {
		buf := make([]byte, 1500)
		n, _, err := conn.ReadFrom(buf)
		if err != nil {
			panic(err)
		}

		p, err := decodePacket(buf[:n])
		if err != nil {
			fmt.Printf("Error decoding header: %v\n", err)
			continue
		}

		c <- p
	}
}

func send(conn net.PacketConn, c chan *Packet) {
	for {
		p := <-c
		fmt.Printf("Sending %s\n", p)
	}
}

func main() {
	fmt.Printf("Starting ospfd with uid %d\n", os.Getuid())

	conn, err := net.ListenPacket("ip4:ospf", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	raw := ipv4.NewPacketConn(conn)

	iface, err := net.InterfaceByName("bridge100")
	if err != nil {
		panic(err)
	}

	if err := raw.JoinGroup(iface, &allSPFRouters); err != nil {
		panic(err)
	}
	defer raw.LeaveGroup(iface, &allSPFRouters)

	if err := raw.SetMulticastInterface(iface); err != nil {
		panic(err)
	}

	if err := raw.SetMulticastLoopback(false); err != nil {
		panic(err)
	}

	// 0xc0 == DSCP CS6, No ECN
	if err := raw.SetTOS(0xc0); err != nil {
		panic(err)
	}

	// TODO: try SetTTL instead of SetMulticastTTL and see what happens
	if err := raw.SetMulticastTTL(1); err != nil {
		panic(err)
	}

	r := make(chan *Packet)
	w := make(chan *Packet)
	hello := time.Tick(helloInterval)

	go listen(conn, r)
	go send(conn, w)

	for {
		select {
		case p := <-r:
			fmt.Println(p)
		case <-hello:
			var p Packet
			p.Type = TypeHello
			p.Length = 44
			p.RouterID = []byte{192, 168, 1, 1}
			p.AreaID = []byte{0, 0, 0, 0}
			p.AuthType = 0
			p.AuthData = 0

			var hello Hello
			hello.NetworkMask = []byte{255, 255, 255, 0}
			hello.HelloInterval = 10
			hello.Options = 0x2
			hello.RtrPriority = 1
			hello.RtrDeadInterval = 40
			hello.DRouter = []byte{0, 0, 0, 0}
			hello.BDRouter = []byte{0, 0, 0, 0}
			p.Content = hello

			w <- &p
		}
	}

	// allSPFRouters := net.IPAddr{IP: net.IPv4(224, 0, 0, 5)}

	// fmt.Println(iface.MulticastAddrs())

	// conn, err := net.Dial("ip4:ospf", "127.0.0.1")
	// if err != nil {
	// 	panic(err)
	// }
	// defer conn.Close()

	// fmt.Println(conn.LocalAddr(), conn.RemoteAddr())

	// n, err := conn.Write([]byte{1, 2, 3, 4})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println(n)
}
