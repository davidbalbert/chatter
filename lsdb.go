package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net/netip"
	"time"
)

const (
	initialSequenceNumber = math.MinInt32 + 1
	maxSequenceNumber     = math.MaxInt32
	maxAge                = 3600 // 1 hour
	maxAgeDiff            = 900  // 15 minutes
	minLSArrival          = 1 * time.Second
)

type lsdbKey struct {
	lsType            lsType
	lsID              netip.Addr
	advertisingRouter netip.Addr
}

// TODO: we need to make the LSDB safe for use from multiple goroutines at the same time
// Requirements:
// - Many readers, one writer
// - Do it in a go-like way: share memory by communicating
// - Interface should be simple, blocking. Hide goroutines inside the implementation.
// - Has to work with copyHeaders, which gets many goroutines
type lsdb struct {
	areas      map[netip.Addr]map[lsdbKey]*installedLSA
	externalDB map[lsdbKey]*installedLSA
}

func newLSDB() *lsdb {
	return &lsdb{
		areas:      make(map[netip.Addr]map[lsdbKey]*installedLSA),
		externalDB: make(map[lsdbKey]*installedLSA),
	}
}

func shouldRecalculateRoutingTable(old, new lsa) bool {
	if old == nil {
		return true
	}

	optionsChanged := old.Options() != new.Options()
	maxAgeChanged := old.Age() == maxAge && new.Age() != maxAge || old.Age() != maxAge && new.Age() == maxAge
	lengthChanged := old.Length() != new.Length()
	bodyChanged := !bytes.Equal(old.Bytes()[lsaHeaderSize:], new.Bytes()[lsaHeaderSize:])

	return optionsChanged || maxAgeChanged || lengthChanged || bodyChanged
}

func (db *lsdb) set(areaID netip.Addr, new lsa) {
	key := new.Key()

	if new.Type() == lsTypeASExternal {
		// old := db.externalDB[key]
		db.externalDB[key] = &installedLSA{new, time.Now()}

		// TODO: nil pointer derererence here
		// if shouldRecalculateRoutingTable(old, new) {
		// 	fmt.Println("TODO: recalculate routing table")
		// }

		return
	}

	if _, ok := db.areas[areaID]; !ok {
		db.areas[areaID] = make(map[lsdbKey]*installedLSA)
	}

	// old := db.areas[areaID][key]
	db.areas[areaID][key] = &installedLSA{new, time.Now()}

	// TODO: nil pointer dereference here
	// if shouldRecalculateRoutingTable(old, new) {
	// 	fmt.Println("TODO: recalculate routing table")
	// }
}

func (db *lsdb) get(areaID netip.Addr, lsType lsType, lsID netip.Addr, advRtr netip.Addr) *installedLSA {
	key := lsdbKey{
		lsType:            lsType,
		lsID:              lsID,
		advertisingRouter: advRtr,
	}

	if lsType == lsTypeASExternal {
		return db.externalDB[key]
	}

	areaDB, ok := db.areas[areaID]
	if !ok {
		return nil
	}

	return areaDB[key]
}

func (db *lsdb) delete(areaID netip.Addr, lsType lsType, lsID netip.Addr, advRtr netip.Addr) {
	key := lsdbKey{
		lsType:            lsType,
		lsID:              lsID,
		advertisingRouter: advRtr,
	}

	if lsType == lsTypeASExternal {
		delete(db.externalDB, key)
		return
	}

	areaDB, ok := db.areas[areaID]
	if !ok {
		return
	}

	delete(areaDB, key)
}

func (db *lsdb) copyHeaders(areaID netip.Addr) []lsaHeader {
	var headers []lsaHeader

	areaDB, ok := db.areas[areaID]
	if !ok {
		return nil
	}

	for _, lsa := range areaDB {
		headers = append(headers, *lsa.copyHeader())
	}

	// TODO: I think we need to copy the external headers as well

	return headers
}

func fletcher16(data ...[]byte) (r0, r1 int) {
	var c0, c1 int

	for _, d := range data {
		for _, b := range d {
			c0 = (c0 + int(b)) % 255
			c1 = (c1 + c0) % 255
		}
	}

	return c0, c1
}

func fletcher16Checksum(data []byte) uint16 {
	c0, c1 := fletcher16(data)
	return uint16(c1<<8 | c0)
}

// offset is the offset of the checksum field in the data
func fletcher16Checkbytes(data []byte, offset int) uint16 {
	c0, c1 := fletcher16(data[:offset], []byte{0, 0}, data[offset+2:])

	x := ((len(data)-offset-1)*c0 - c1) % 255
	if x <= 0 {
		x += 255
	}

	y := 510 - c0 - x
	if y > 255 {
		y -= 255
	}

	return uint16(x<<8 | y)
}

func lsaCheckbytes(data []byte) uint16 {
	// Checksum offset is 16, but we skip age (the first 2 bytes of the LSA),
	// so we have to subtract 2.
	return fletcher16Checkbytes(data[2:], 14)
}

type lsType uint8

const (
	lsTypeUnknown lsType = 0

	lsTypeRouter      lsType = 1
	lsTypeNetwork     lsType = 2
	lsTypeSummary     lsType = 3
	lsTypeASBRSummary lsType = 4
	lsTypeASExternal  lsType = 5
)

type lsaMetadata interface {
	Type() lsType
	Length() int
	LSID() netip.Addr
	Key() lsdbKey
	AdvertisingRouter() netip.Addr
	Age() uint16
	SequenceNumber() int32
	Checksum() uint16
	Options() uint8
	Compare(lsaMetadata) int
}

type lsa interface {
	lsaMetadata
	Bytes() []byte
	SetAge(uint16)
	copyHeader() *lsaHeader
	checksumIsValid() bool
}

type installedLSA struct {
	lsa
	installedAt time.Time
}

// Rules of LSAs:
//
// - With the exception of age, all fields are immutable.
// - Age must be changed using SetAge(), which will also update the encoded bytes.
// - LsaBase.bytes and lsaHeader.bytes are never nil.

type lsaHeader struct {
	age               uint16
	options           uint8
	lsType            lsType
	lsID              netip.Addr
	advertisingRouter netip.Addr
	seqNumber         int32
	lsChecksum        uint16
	length            uint16

	bytes []byte // the encoded header
}

const lsaHeaderSize = 20

type lsaBase struct {
	lsaHeader
	bytes []byte // the entire LSA, including the header
}

// Called when parsing DD and LSAck packets. Copies data so as to not retain the whole
// packet longer than necessary. len(data) must be exactly lsaHeaderSize.
func decodeLSAHeader(data []byte) (*lsaHeader, error) {
	if len(data) != lsaHeaderSize {
		return nil, fmt.Errorf("lsa header must be exactly %d bytes", lsaHeaderSize)
	}

	bytes := make([]byte, lsaHeaderSize)
	copy(bytes, data)

	return decodeLSAHeaderNoCopy(bytes)
}

// Called from decodeLSA and decodeLSAHeader, which both perform their own copies.
// When called from decodeLSA, data contains the entire LSA, not just the header.
func decodeLSAHeaderNoCopy(data []byte) (*lsaHeader, error) {
	if len(data) < lsaHeaderSize {
		return nil, fmt.Errorf("lsa header too short")
	}

	var t lsType
	if data[3] < 1 || data[3] > 5 {
		t = lsTypeUnknown
	} else {
		t = lsType(data[3])
	}

	return &lsaHeader{
		age:               binary.BigEndian.Uint16(data[0:2]),
		options:           data[2],
		lsType:            t,
		lsID:              mustAddrFromSlice(data[4:8]),
		advertisingRouter: mustAddrFromSlice(data[8:12]),
		seqNumber:         int32(binary.BigEndian.Uint32(data[12:16])),
		lsChecksum:        binary.BigEndian.Uint16(data[16:18]),
		length:            binary.BigEndian.Uint16(data[18:20]),
		bytes:             data[:lsaHeaderSize],
	}, nil
}

// Should only be called from within the new{type}LSA constructors. Calculates the
// checksum, so it should always be called after the LSA body is encoded.
func (h *lsaHeader) encodeTo(data []byte) {
	if len(data) < lsaHeaderSize {
		panic("lsaHeader.encodeTo: data is too short")
	}

	binary.BigEndian.PutUint16(data[0:2], h.age)
	data[2] = h.options
	data[3] = byte(h.lsType)
	copy(data[4:8], to4(h.lsID))
	copy(data[8:12], to4(h.advertisingRouter))
	binary.BigEndian.PutUint32(data[12:16], uint32(h.seqNumber))
	// checksum is calculated after the full header is encoded
	binary.BigEndian.PutUint16(data[18:20], h.length)

	h.lsChecksum = lsaCheckbytes(data[:h.length])
	binary.BigEndian.PutUint16(data[16:18], h.lsChecksum)

	h.bytes = data[:lsaHeaderSize]
}

func (h *lsaHeader) Age() uint16 {
	return h.age
}

func (h *lsaHeader) SetAge(age uint16) {
	h.age = age
	binary.BigEndian.PutUint16(h.bytes[0:2], age)
}

func (h *lsaHeader) Bytes() []byte {
	return h.bytes
}

func (h *lsaHeader) Type() lsType {
	return h.lsType
}

func (h *lsaHeader) Length() int {
	return int(h.length)
}

func (h *lsaHeader) LSID() netip.Addr {
	return h.lsID
}

func (h *lsaHeader) AdvertisingRouter() netip.Addr {
	return h.advertisingRouter
}

func (h *lsaHeader) Key() lsdbKey {
	return lsdbKey{
		lsType:            h.lsType,
		lsID:              h.lsID,
		advertisingRouter: h.advertisingRouter,
	}
}

func (h *lsaHeader) SequenceNumber() int32 {
	return h.seqNumber
}

func (h *lsaHeader) Checksum() uint16 {
	return h.lsChecksum
}

func (h *lsaHeader) Options() uint8 {
	return h.options
}

func (h *lsaHeader) Compare(other lsaMetadata) int {
	// lessRecent := -1
	// moreRecent := 1

	s1, s2 := h.SequenceNumber(), other.SequenceNumber()
	if s1 < s2 {
		return -1
	} else if s1 > s2 {
		return 1
	}

	c1, c2 := h.Checksum(), other.Checksum()
	if c1 < c2 {
		return -1
	} else if c1 > c2 {
		return 1
	}

	a1, a2 := int(h.Age()), int(other.Age())
	if a1 != maxAge && a2 == maxAge {
		return -1
	} else if a1 == maxAge && a2 != maxAge {
		return 1
	}

	diff := abs(a1 - a2)
	if diff > maxAgeDiff && a1 < a2 {
		return 1
	} else if diff > maxAgeDiff && a1 > a2 {
		return -1
	}

	return 0
}

func (base *lsaBase) Bytes() []byte {
	return base.bytes
}

func (base *lsaBase) checksumIsValid() bool {
	// skip lsAge (first two bytes)
	return fletcher16Checksum(base.bytes[2:]) == 0
}

func (base *lsaBase) copyHeader() *lsaHeader {
	h := base.lsaHeader

	bytes := make([]byte, len(h.bytes))
	copy(bytes, h.bytes)

	return &lsaHeader{
		age:               h.age,
		options:           h.options,
		lsType:            h.lsType,
		lsID:              h.lsID,
		advertisingRouter: h.advertisingRouter,
		seqNumber:         h.seqNumber,
		lsChecksum:        h.lsChecksum,
		length:            h.length,

		bytes: bytes,
	}
}

type tosMetric struct {
	tos    uint8
	metric uint16
}

const tosMetricSize = 4

func decodeTosMetric(data []byte) (*tosMetric, error) {
	if len(data) < tosMetricSize {
		return nil, fmt.Errorf("decodeTosMetric: data is too short")
	}

	return &tosMetric{
		tos:    data[0],
		metric: binary.BigEndian.Uint16(data[2:4]),
	}, nil
}

func (tm *tosMetric) encodeTo(data []byte) {
	if len(data) < tosMetricSize {
		panic("tosMetric.encodeTo: data is too short")
	}

	data[0] = tm.tos
	binary.BigEndian.PutUint16(data[2:4], tm.metric)
}

type linkType uint8

const (
	lPointToPoint linkType = 1
	lTransit      linkType = 2
	lStub         linkType = 3
	lVirtual      linkType = 4
)

type link struct {
	linkID     netip.Addr
	linkData   uint32 // TODO: for unnumbered interfaces, this is a MIB-II ifIndex. Otherwise it's an IP address.
	linkType   linkType
	metric     uint16
	tosMetrics []tosMetric
}

const minLinkSize = 12

func newLink(linkType linkType, linkID netip.Addr, linkData uint32, metric uint16) *link {
	return &link{
		linkID:     linkID,
		linkData:   linkData,
		linkType:   linkType,
		metric:     metric,
		tosMetrics: nil,
	}
}

func (l *link) size() int {
	return minLinkSize + len(l.tosMetrics)*4
}

func (l *link) encodeTo(data []byte) {
	if len(data) < l.size() {
		panic("link.encodeTo: data is too short")
	}

	copy(data[0:4], to4(l.linkID))
	binary.BigEndian.PutUint32(data[4:8], l.linkData)
	data[8] = byte(l.linkType)
	data[9] = byte(len(l.tosMetrics))
	binary.BigEndian.PutUint16(data[10:12], l.metric)

	for i, tos := range l.tosMetrics {
		tos.encodeTo(data[12+i*4:])
	}
}

func decodeLink(data []byte) (*link, error) {
	if len(data) < minLinkSize {
		return nil, fmt.Errorf("link too short")
	}

	l := &link{
		linkID:   mustAddrFromSlice(data[0:4]),
		linkData: binary.BigEndian.Uint32(data[4:8]),
		linkType: linkType(data[8]),
		metric:   binary.BigEndian.Uint16(data[10:12]),
	}

	nTOS := data[9]

	if len(data) < minLinkSize+int(nTOS)*4 {
		return nil, fmt.Errorf("link too short (TOS metrics)")
	}

	for i := 0; i < int(nTOS); i++ {
		tos, err := decodeTosMetric(data[12+i*4:])
		if err != nil {
			return nil, err
		}

		l.tosMetrics = append(l.tosMetrics, *tos)
	}

	return l, nil
}

type routerLSA struct {
	lsaBase
	virtual  bool
	external bool
	border   bool
	links    []link
}

const (
	// Router LSA flags
	rlfBorder   = 1 << 0
	rlfExternal = 1 << 1
	rlfVirtual  = 1 << 2
)

const minRouterLSASize = lsaHeaderSize + 4

func newRouterLSA(inst *Instance, area *area) (*routerLSA, error) {
	lsa := routerLSA{
		lsaBase: lsaBase{
			lsaHeader: lsaHeader{
				age:               0,
				options:           capE, // TODO: don't set this in stub areas
				lsType:            lsTypeRouter,
				lsID:              inst.routerID,
				advertisingRouter: inst.routerID,
				seqNumber:         initialSequenceNumber,
				lsChecksum:        0,
				length:            0,
			},
		},
		virtual:  false,
		external: false,
		border:   false,
		links:    nil,
	}

	for _, iface := range inst.interfaces {
		if iface.areaID != area.id {
			continue
		}

		switch iface.networkType {
		case networkPointToPoint, networkPointToMultipoint:
			for _, n := range iface.neighbors {
				if n.state != nFull {
					continue
				}

				// TODO: handle unnumbered point to points, which should use the MIB-II ifIndex.
				link := newLink(lPointToPoint, n.neighborID, binary.BigEndian.Uint32(to4(n.addr)), 1)
				lsa.links = append(lsa.links, *link)
			}
		default:
			return nil, fmt.Errorf("unsupported interface type %v", iface.networkType)
		}

		// TODO: there are more states to handle here (ptp, address is a /32, iface.state == loopback)
		// TODO: there's also an else case in bird, which I need to find in the spec.
		if iface.networkType == networkPointToMultipoint {
			lsa.links = append(lsa.links, *newLink(lStub, iface.Address, 0xffffffff, 0))
		}
	}

	size := minRouterLSASize
	for _, link := range lsa.links {
		size += link.size()
	}
	lsa.length = uint16(size)

	lsa.encode()

	return &lsa, nil
}

func decodeRouterLSA(base *lsaBase) (*routerLSA, error) {
	if len(base.bytes) < minRouterLSASize {
		return nil, fmt.Errorf("router LSA too short")
	}

	lsa := &routerLSA{
		lsaBase:  *base,
		virtual:  base.bytes[20]&rlfVirtual != 0,
		external: base.bytes[20]&rlfExternal != 0,
		border:   base.bytes[20]&rlfBorder != 0,
	}

	nLinks := binary.BigEndian.Uint16(base.bytes[22:24])

	for i := 0; i < int(nLinks); i++ {
		link, err := decodeLink(base.bytes[lsaHeaderSize+4+i*12:])
		if err != nil {
			return nil, err
		}

		lsa.links = append(lsa.links, *link)
	}

	return lsa, nil
}

func (lsa *routerLSA) encode() {
	buf := make([]byte, lsa.length)

	if lsa.virtual {
		buf[20] |= rlfVirtual
	}

	if lsa.external {
		buf[20] |= rlfExternal
	}

	if lsa.border {
		buf[20] |= rlfBorder
	}

	binary.BigEndian.PutUint16(buf[22:24], uint16(len(lsa.links)))

	i := 24
	for _, link := range lsa.links {
		link.encodeTo(buf[i:])
		i += link.size()
	}

	// encode header last, since it calculates the checksum of the whole LSA.
	lsa.lsaHeader.encodeTo(buf)
	lsa.bytes = buf
}

type networkLSA struct {
	lsaBase
}

func decodeNetworkLSA(base *lsaBase) (*networkLSA, error) {
	return nil, fmt.Errorf("network LSA not implemented")
}

type summaryLSA struct {
	lsaBase
}

func decodeSummaryLSA(base *lsaBase) (*summaryLSA, error) {
	return nil, fmt.Errorf("summary LSA not implemented")
}

type asbrSummaryLSA struct {
	lsaBase
}

func decodeASBRSummaryLSA(base *lsaBase) (*asbrSummaryLSA, error) {
	return nil, fmt.Errorf("ASBR summary LSA not implemented")
}

type asExternalLSA struct {
	lsaBase
}

func decodeASExternalLSA(base *lsaBase) (*asExternalLSA, error) {
	return nil, fmt.Errorf("AS external LSA not implemented")
}

type unknownLSA struct {
	lsaBase
}

func decodeLSA(data []byte) (lsa, error) {
	length := binary.BigEndian.Uint16(data[18:20])

	if len(data) < int(length) {
		return nil, fmt.Errorf("LSA too short")
	}

	bytes := make([]byte, length)
	copy(bytes, data[:length])

	h, err := decodeLSAHeaderNoCopy(data)
	if err != nil {
		return nil, err
	}

	base := &lsaBase{
		lsaHeader: *h,
		bytes:     bytes,
	}

	switch h.lsType {
	case lsTypeRouter:
		return decodeRouterLSA(base)
	case lsTypeNetwork:
		return decodeNetworkLSA(base)
	case lsTypeSummary:
		return decodeSummaryLSA(base)
	case lsTypeASBRSummary:
		return decodeASBRSummaryLSA(base)
	case lsTypeASExternal:
		return decodeASExternalLSA(base)
	case lsTypeUnknown:
		return &unknownLSA{*base}, nil
	}

	return nil, fmt.Errorf("unknown LSA type %d", h.lsType)
}
