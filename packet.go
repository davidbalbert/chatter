package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"

	"golang.org/x/net/ipv4"
)

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

type Packet interface {
	encode() []byte
	AreaID() netip.Addr
	handleOn(*Interface)
}

type header struct {
	messageType
	length   uint16
	routerID netip.Addr
	areaID   netip.Addr
	checksum uint16
	authType uint16
	authData uint64

	src netip.Addr
}

func (h *header) encodeTo(data []byte) {
	data[0] = 2
	data[1] = uint8(h.messageType)
	binary.BigEndian.PutUint16(data[2:4], h.length)
	copy(data[4:8], to4(h.routerID))
	copy(data[8:12], to4(h.areaID))

	// Skip Checksum - data[12:14]. Packet.encode must fill it in at the end.

	binary.BigEndian.PutUint16(data[14:16], h.authType)
	binary.BigEndian.PutUint64(data[16:24], h.authData)
}

func (h *header) String() string {
	return fmt.Sprintf("OSPFv2 %s router=%s area=%s", h.messageType, h.routerID, h.areaID)
}

func (h *header) AreaID() netip.Addr {
	return h.areaID
}

type Hello struct {
	header

	networkMask        net.IPMask
	helloInterval      uint16
	options            uint8
	routerPriority     uint8
	routerDeadInterval uint32
	dRouter            netip.Addr
	bdRouter           netip.Addr
	neighbors          []netip.Addr
}

func (hello *Hello) String() string {
	var b bytes.Buffer

	fmt.Fprint(&b, hello.header.String())
	fmt.Fprintf(&b, " mask=%s interval=%d options=0x%x priority=%d dead=%d dr=%s bdr=%s", net.IP(hello.networkMask), hello.helloInterval, hello.options, hello.routerPriority, hello.routerDeadInterval, hello.dRouter, hello.bdRouter)

	for _, n := range hello.neighbors {
		fmt.Fprintf(&b, "\n  neighbor=%s", n)
	}

	return b.String()
}

func (hello *Hello) netmaskBits() int {
	ones, _ := hello.networkMask.Size()

	return ones
}

func (hello *Hello) encode() []byte {
	hello.length = 44 + uint16(len(hello.neighbors)*4)

	data := make([]byte, hello.length)
	hello.header.encodeTo(data)

	copy(data[24:28], hello.networkMask)
	binary.BigEndian.PutUint16(data[28:30], hello.helloInterval)
	data[30] = hello.options
	data[31] = hello.routerPriority
	binary.BigEndian.PutUint32(data[32:36], hello.routerDeadInterval)
	copy(data[36:40], to4(hello.dRouter))
	copy(data[40:44], to4(hello.bdRouter))
	for i, neighbor := range hello.neighbors {
		copy(data[44+i*4:48+i*4], to4(neighbor))
	}

	hello.checksum = checksum(data[0:16], data[24:])
	binary.BigEndian.PutUint16(data[12:14], hello.checksum)

	return data
}

func (hello *Hello) handleOn(iface *Interface) {
	iface.handleHello(hello)
}

func decodePacket(ip *ipv4.Header, data []byte) (Packet, error) {
	if len(data) < 24 {
		return nil, fmt.Errorf("packet too short")
	}

	version := data[0]
	if version != 2 {
		return nil, fmt.Errorf("unsupported OSPF version %d", version)
	}

	var h header

	h.messageType = messageType(data[1])
	h.length = binary.BigEndian.Uint16(data[2:4])
	h.routerID = mustAddrFromSlice(data[4:8])
	h.areaID = mustAddrFromSlice(data[8:12])
	h.checksum = binary.BigEndian.Uint16(data[12:14])
	h.authType = binary.BigEndian.Uint16(data[14:16])
	h.authData = binary.BigEndian.Uint64(data[16:24])

	h.src = mustAddrFromSlice(ip.Src)

	if len(data) != int(h.length) {
		return nil, fmt.Errorf("packet length mismatch")
	}

	if checksum(data) != 0 {
		return nil, fmt.Errorf("packet checksum mismatch")
	}

	switch h.messageType {
	case TypeHello:
		if h.length < 44 {
			return nil, fmt.Errorf("hello packet too short")
		}

		var hello Hello
		hello.header = h
		hello.networkMask = data[24:28]
		hello.helloInterval = binary.BigEndian.Uint16(data[28:30])
		hello.options = data[30]
		hello.routerPriority = data[31]
		hello.routerDeadInterval = binary.BigEndian.Uint32(data[32:36])
		hello.dRouter = mustAddrFromSlice(data[36:40])
		hello.bdRouter = mustAddrFromSlice(data[40:44])

		for i := 44; i < int(h.length); i += 4 {
			hello.neighbors = append(hello.neighbors, mustAddrFromSlice(data[i:i+4]))
		}

		return &hello, nil
	case TypeDatabaseDescription:
		fallthrough
	case TypeLinkStateRequest:
		fallthrough
	case TypeLinkStateUpdate:
		fallthrough
	case TypeLinkStateAcknowledgement:
		return nil, fmt.Errorf("unsupported OSPF packet type %s", h.messageType)
	default:
		return nil, fmt.Errorf("unknown OSPF packet type %d", h.messageType)
	}
}
