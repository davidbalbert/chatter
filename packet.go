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
	pLinkStateAcknowledgment
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
	case pLinkStateAcknowledgment:
		return "Link State Acknowledgement"
	default:
		return "Unknown"
	}
}

type PacketHandler interface {
	handleHello(*helloPacket)
	handleDatabaseDescription(*databaseDescriptionPacket)
	handleLinkStateRequest(*linkStateRequestPacket)
	handleLinkStateUpdate(*linkStateUpdatePacket)
	handleLinkStateAcknowledgment(*linkStateAcknowledgmentPacket)
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

func decodeHello(h *header, data []byte) (*helloPacket, error) {
	if int(h.length) < minHelloSize {
		return nil, fmt.Errorf("hello packet too short")
	}

	hello := helloPacket{
		header:             *h,
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

func decodeDatabaseDescription(h *header, data []byte) (*databaseDescriptionPacket, error) {
	if int(h.length) < minDDSize {
		return nil, fmt.Errorf("database description packet too short")
	}

	dd := databaseDescriptionPacket{
		header:         *h,
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

	for i, lsaHeader := range dd.lsaHeaders {
		copy(data[32+i*lsaHeaderSize:], lsaHeader.Bytes())
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

type req struct {
	lsType            lsType
	lsID              netip.Addr
	advertisingRouter netip.Addr
}

const reqSize = 12

func decodeReq(data []byte) *req {
	return &req{
		lsType:            lsType(binary.BigEndian.Uint32(data[0:4])),
		lsID:              mustAddrFromSlice(data[4:8]),
		advertisingRouter: mustAddrFromSlice(data[8:12]),
	}
}

func (r *req) encodeTo(data []byte) {
	if len(data) < reqSize {
		panic("req.encodeTo: data too short")
	}

	binary.BigEndian.PutUint32(data[0:4], uint32(r.lsType))
	copy(data[4:8], to4(r.lsID))
	copy(data[8:12], to4(r.advertisingRouter))
}

type linkStateRequestPacket struct {
	header
	reqs []req
}

var minLsrSize = 24

func newLinkStateRequest(iface *Interface, reqs []req) *linkStateRequestPacket {
	lsr := linkStateRequestPacket{
		header: header{
			packetType: pLinkStateRequest,
			length:     uint16(minLsrSize),
			routerID:   iface.instance.routerID,
			areaID:     iface.areaID,
			authType:   0,
			authData:   0,
		},
		reqs: reqs,
	}

	return &lsr
}

func decodeLinkStateRequest(h *header, data []byte) (*linkStateRequestPacket, error) {
	if int(h.length) < minLsrSize {
		return nil, fmt.Errorf("link state request packet too short")
	}

	lsr := linkStateRequestPacket{
		header: *h,
	}

	for i := minLsrSize; i < int(h.length); i += reqSize {
		lsr.reqs = append(lsr.reqs, *decodeReq(data[i:]))
	}

	return &lsr, nil
}

func (lsr *linkStateRequestPacket) encode() []byte {
	lsr.length = uint16(minLsrSize + len(lsr.reqs)*12)

	data := make([]byte, lsr.length)
	lsr.header.encodeTo(data)

	for i, r := range lsr.reqs {
		r.encodeTo(data[24+i*12:])
	}

	lsr.checksum = ipChecksum(data[0:16], data[24:])
	binary.BigEndian.PutUint16(data[12:14], lsr.checksum)

	return data
}

func (lsr *linkStateRequestPacket) AreaID() netip.Addr {
	return lsr.areaID
}

func (lsr *linkStateRequestPacket) handleOn(handler PacketHandler) {
	handler.handleLinkStateRequest(lsr)
}

type linkStateUpdatePacket struct {
	header
	lsas []lsa
}

var minLSUSize = 28

func newLinkStateUpdate(iface *Interface, lsas []lsa) *linkStateUpdatePacket {
	size := minLSUSize
	for _, lsa := range lsas {
		size += lsaHeaderSize + lsa.Length()
	}

	lsu := linkStateUpdatePacket{
		header: header{
			packetType: pLinkStateUpdate,
			length:     uint16(size),
			routerID:   iface.instance.routerID,
			areaID:     iface.areaID,
			authType:   0,
			authData:   0,
		},
		lsas: lsas,
	}

	return &lsu
}

func decodeLinkStateUpdate(h *header, data []byte) (*linkStateUpdatePacket, error) {
	if int(h.length) < minLSUSize {
		return nil, fmt.Errorf("link state update packet too short")
	}

	lsu := linkStateUpdatePacket{
		header: *h,
	}

	nLSAs := int(binary.BigEndian.Uint32(data[24:28]))

	offset := 28
	for i := 0; i < nLSAs; i++ {
		lsa, err := decodeLSA(data[offset:])
		if err != nil {
			return nil, fmt.Errorf("failed to decode LSA: %v", err)
		}

		lsu.lsas = append(lsu.lsas, lsa)
		offset += lsa.Length()
	}

	return &lsu, nil
}

func (lsu *linkStateUpdatePacket) encode() []byte {
	size := minLSUSize
	for _, lsa := range lsu.lsas {
		size += lsa.Length()
	}
	lsu.length = uint16(size)

	data := make([]byte, lsu.length)
	lsu.header.encodeTo(data)

	binary.BigEndian.PutUint32(data[24:28], uint32(len(lsu.lsas)))
	offset := 28
	for _, lsa := range lsu.lsas {
		offset += copy(data[offset:], lsa.Bytes())
	}

	lsu.checksum = ipChecksum(data[0:16], data[24:])
	binary.BigEndian.PutUint16(data[12:14], lsu.checksum)

	return data
}

func (lsu *linkStateUpdatePacket) handleOn(handler PacketHandler) {
	handler.handleLinkStateUpdate(lsu)
}

type linkStateAcknowledgmentPacket struct {
	header
	lsaHeaders []lsaHeader
}

var minLSAckSize = 24

func newLinkStateAcknowledgment(iface *Interface, lsaHeaders []lsaHeader) *linkStateAcknowledgmentPacket {
	lsack := linkStateAcknowledgmentPacket{
		header: header{
			packetType: pLinkStateAcknowledgment,
			length:     uint16(minLSAckSize + len(lsaHeaders)*lsaHeaderSize),
			routerID:   iface.instance.routerID,
			areaID:     iface.areaID,
			authType:   0,
			authData:   0,
		},
		lsaHeaders: lsaHeaders,
	}

	return &lsack
}

func decodeLinkStateAcknowledgment(h *header, data []byte) (*linkStateAcknowledgmentPacket, error) {
	if int(h.length) < minLSAckSize {
		return nil, fmt.Errorf("link state acknowledgement packet too short")
	}

	lsack := linkStateAcknowledgmentPacket{
		header: *h,
	}

	for i := 24; i < int(h.length); i += lsaHeaderSize {
		header, err := decodeLSAHeader(data[i:])
		if err != nil {
			return nil, fmt.Errorf("failed to decode LSA header: %v", err)
		}

		lsack.lsaHeaders = append(lsack.lsaHeaders, *header)
	}

	return &lsack, nil
}

func (lsack *linkStateAcknowledgmentPacket) encode() []byte {
	lsack.length = uint16(minLSAckSize + len(lsack.lsaHeaders)*lsaHeaderSize)

	data := make([]byte, lsack.length)
	lsack.header.encodeTo(data)

	offset := 24
	for _, lsaHeader := range lsack.lsaHeaders {
		offset += copy(data[offset:], lsaHeader.Bytes())
	}

	lsack.checksum = ipChecksum(data[0:16], data[24:])
	binary.BigEndian.PutUint16(data[12:14], lsack.checksum)

	return data
}

func (lsack *linkStateAcknowledgmentPacket) handleOn(handler PacketHandler) {
	handler.handleLinkStateAcknowledgment(lsack)
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
		return decodeHello(&h, data)
	case pDatabaseDescription:
		return decodeDatabaseDescription(&h, data)
	case pLinkStateRequest:
		return decodeLinkStateRequest(&h, data)
	case pLinkStateUpdate:
		return decodeLinkStateUpdate(&h, data)
	case pLinkStateAcknowledgment:
		return decodeLinkStateAcknowledgment(&h, data)
	default:
		return nil, fmt.Errorf("unknown OSPF packet type %d", h.packetType)
	}
}
