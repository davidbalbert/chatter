package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"

	"golang.org/x/net/ipv4"
)

func ipChecksum(data ...[]byte) uint16 {
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

type packetType uint8

const (
	pHello packetType = iota + 1
	pDatabaseDescription
	pLinkStateRequest
	pLinkStateUpdate
	pLinkStateAcknowledgement
)

func (t packetType) String() string {
	switch t {
	case pHello:
		return "Hello"
	case pDatabaseDescription:
		return "Database Description"
	case pLinkStateRequest:
		return "Link State Request"
	case pLinkStateUpdate:
		return "Link State Update"
	case pLinkStateAcknowledgement:
		return "Link State Acknowledgement"
	default:
		return "Unknown"
	}
}

type PacketHandler interface {
	handleHello(*helloPacket)
	handleDatabaseDescription(*databaseDescriptionPacket)
}

type Packet interface {
	encode() []byte
	AreaID() netip.Addr
	handleOn(PacketHandler)
}

type header struct {
	packetType
	length   uint16
	routerID netip.Addr
	areaID   netip.Addr
	checksum uint16
	authType uint16
	authData uint64

	src netip.Addr
}

func (h *header) encodeTo(data []byte) {
	if len(data) < 24 {
		panic("header.encodeTo: data is too short")
	}

	data[0] = 2
	data[1] = uint8(h.packetType)
	binary.BigEndian.PutUint16(data[2:4], h.length)
	copy(data[4:8], to4(h.routerID))
	copy(data[8:12], to4(h.areaID))

	// Skip Checksum - data[12:14]. Packet.encode must fill it in at the end.

	binary.BigEndian.PutUint16(data[14:16], h.authType)
	binary.BigEndian.PutUint64(data[16:24], h.authData)
}

func (h *header) String() string {
	return fmt.Sprintf("OSPFv2 %s router=%s area=%s", h.packetType, h.routerID, h.areaID)
}

func (h *header) AreaID() netip.Addr {
	return h.areaID
}

type helloPacket struct {
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

var minHelloSize = 44

func newHello(iface *Interface) *helloPacket {
	hello := &helloPacket{
		header: header{
			packetType: pHello,
			length:     uint16(minHelloSize + len(iface.neighbors)*4),
			routerID:   iface.instance.routerID,
			areaID:     iface.areaID,
			authType:   0,
			authData:   0,
		},
		networkMask:        net.CIDRMask(iface.Prefix.Bits(), 32),
		helloInterval:      iface.HelloInteral,
		options:            capE, // TODO: don't set capE if we're in a stub area
		routerPriority:     1,    // TODO
		routerDeadInterval: iface.RouterDeadInterval,
		dRouter:            netip.IPv4Unspecified(),
		bdRouter:           netip.IPv4Unspecified(),
		neighbors:          make([]netip.Addr, 0, len(iface.neighbors)),
	}

	for _, neighbor := range iface.neighbors {
		hello.neighbors = append(hello.neighbors, neighbor.neighborID)
	}

	return hello
}

func (hello *helloPacket) String() string {
	var b bytes.Buffer

	fmt.Fprint(&b, hello.header.String())
	fmt.Fprintf(&b, " mask=%s interval=%d options=0x%x priority=%d dead=%d dr=%s bdr=%s", net.IP(hello.networkMask), hello.helloInterval, hello.options, hello.routerPriority, hello.routerDeadInterval, hello.dRouter, hello.bdRouter)

	for _, n := range hello.neighbors {
		fmt.Fprintf(&b, "\n  neighbor=%s", n)
	}

	return b.String()
}

func (hello *helloPacket) netmaskBits() int {
	ones, _ := hello.networkMask.Size()

	return ones
}

func (hello *helloPacket) encode() []byte {
	hello.length = uint16(minHelloSize + len(hello.neighbors)*4)

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

	hello.checksum = ipChecksum(data[0:16], data[24:])
	binary.BigEndian.PutUint16(data[12:14], hello.checksum)

	return data
}

func (hello *helloPacket) handleOn(handler PacketHandler) {
	handler.handleHello(hello)
}

func (hello *helloPacket) Header() *header {
	return &hello.header
}

type databaseDescriptionPacket struct {
	header
	interfaceMTU   uint16
	options        uint8
	init           bool
	more           bool
	master         bool
	sequenceNumber uint32
	lsaHeaders     []lsaHeader
}

var minDDSize = 32

const (
	ddFlagMasterSlave byte = 1 << iota
	ddFlagMore
	ddFlagInit
)

func newDatabaseDescription(iface *Interface, sequenceNumber uint32, init, more, master bool, lsas []lsaHeader) *databaseDescriptionPacket {
	dd := databaseDescriptionPacket{
		header: header{
			packetType: pDatabaseDescription,
			length:     uint16(minDDSize + len(lsas)*lsaHeaderSize),
			routerID:   iface.instance.routerID,
			areaID:     iface.areaID,
			authType:   0,
			authData:   0,
		},
		interfaceMTU:   uint16(iface.netif.MTU),
		options:        0x2, // TODO
		init:           init,
		more:           more,
		master:         master,
		sequenceNumber: sequenceNumber,
		lsaHeaders:     lsas,
	}

	return &dd
}

func (dd *databaseDescriptionPacket) encode() []byte {
	dd.length = uint16(minDDSize + len(dd.lsaHeaders)*lsaHeaderSize)

	data := make([]byte, dd.length)
	dd.header.encodeTo(data)

	binary.BigEndian.PutUint16(data[24:26], dd.interfaceMTU)
	data[26] = dd.options
	data[27] = 0
	if dd.master {
		data[27] |= ddFlagMasterSlave
	}
	if dd.more {
		data[27] |= ddFlagMore
	}
	if dd.init {
		data[27] |= ddFlagInit
	}
	binary.BigEndian.PutUint32(data[28:32], dd.sequenceNumber)

	for i, lsa := range dd.lsaHeaders {
		lsa.encodeTo(data[32+i*lsaHeaderSize:])
	}

	dd.checksum = ipChecksum(data[0:16], data[24:])
	binary.BigEndian.PutUint16(data[12:14], dd.checksum)

	return data
}

func (dd *databaseDescriptionPacket) handleOn(handler PacketHandler) {
	handler.handleDatabaseDescription(dd)
}

func (dd *databaseDescriptionPacket) Header() *header {
	return &dd.header
}

func decodePacket(ip *ipv4.Header, data []byte) (Packet, error) {
	if len(data) < 24 {
		return nil, fmt.Errorf("packet too short")
	}

	version := data[0]
	if version != 2 {
		return nil, fmt.Errorf("unsupported OSPF version %d", version)
	}

	h := header{
		packetType: packetType(data[1]),
		length:     binary.BigEndian.Uint16(data[2:4]),
		routerID:   mustAddrFromSlice(data[4:8]),
		areaID:     mustAddrFromSlice(data[8:12]),
		checksum:   binary.BigEndian.Uint16(data[12:14]),
		authType:   binary.BigEndian.Uint16(data[14:16]),
		authData:   binary.BigEndian.Uint64(data[16:24]),

		src: mustAddrFromSlice(ip.Src),
	}

	if len(data) != int(h.length) {
		return nil, fmt.Errorf("packet length mismatch")
	}

	if ipChecksum(data) != 0 {
		return nil, fmt.Errorf("packet checksum mismatch")
	}

	switch h.packetType {
	case pHello:
		if int(h.length) < minHelloSize {
			return nil, fmt.Errorf("hello packet too short")
		}

		hello := helloPacket{
			header:             h,
			networkMask:        data[24:28],
			helloInterval:      binary.BigEndian.Uint16(data[28:30]),
			options:            data[30],
			routerPriority:     data[31],
			routerDeadInterval: binary.BigEndian.Uint32(data[32:36]),
			dRouter:            mustAddrFromSlice(data[36:40]),
			bdRouter:           mustAddrFromSlice(data[40:44]),
		}

		for i := 44; i < int(h.length); i += 4 {
			hello.neighbors = append(hello.neighbors, mustAddrFromSlice(data[i:i+4]))
		}

		return &hello, nil
	case pDatabaseDescription:
		if int(h.length) < minDDSize {
			return nil, fmt.Errorf("database description packet too short")
		}

		dd := databaseDescriptionPacket{
			header:         h,
			interfaceMTU:   binary.BigEndian.Uint16(data[24:26]),
			options:        data[26],
			init:           data[27]&ddFlagInit != 0,
			more:           data[27]&ddFlagMore != 0,
			master:         data[27]&ddFlagMasterSlave != 0,
			sequenceNumber: binary.BigEndian.Uint32(data[28:32]),
		}

		for i := 32; i < int(h.length); i += lsaHeaderSize {
			header, err := decodeLSAHeader(data[i : i+lsaHeaderSize])
			if err != nil {
				return nil, fmt.Errorf("failed to decode LSA header: %v", err)
			}

			dd.lsaHeaders = append(dd.lsaHeaders, *header)
		}

		return &dd, nil
	case pLinkStateRequest:
		fallthrough
	case pLinkStateUpdate:
		fallthrough
	case pLinkStateAcknowledgement:
		return nil, fmt.Errorf("unsupported OSPF packet type %s", h.packetType)
	default:
		return nil, fmt.Errorf("unknown OSPF packet type %d", h.packetType)
	}
}
