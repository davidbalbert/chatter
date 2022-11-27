package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"

	"golang.org/x/net/ipv4"
)

// https://www.rfc-editor.org/rfc/rfc2328.html

var allSPFRouters = net.IPAddr{IP: net.ParseIP("224.0.0.5")}
var allDRouters = net.IPAddr{IP: net.ParseIP("224.0.0.6")}

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

func (h Header) String() string {
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

	// TODO: calculate checksum

	var p Packet
	p.Header = h

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

func (p Packet) String() string {
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

	for {
		buf := make([]byte, 1500)
		n, cm, src, err := raw.ReadFrom(buf)
		if err != nil {
			panic(err)
		}

		p, err := decodePacket(buf[:n])
		if err != nil {
			fmt.Printf("Error decoding header: %v\n", err)
			continue
		}

		fmt.Println(n, cm, src)
		fmt.Println(p)
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
