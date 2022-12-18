package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"net/netip"
)

const (
	initialSequenceNumber = math.MinInt32 + 1
	maxSequenceNumber     = math.MaxInt32
)

type lsdbKey struct {
	lsType lsType
	lsID   netip.Addr
	advRtr netip.Addr
}

type lsdb struct {
	areas      map[netip.Addr]map[lsdbKey]lsa
	externalDB map[lsdbKey]lsa
}

func newLSDB() *lsdb {
	return &lsdb{
		areas:      make(map[netip.Addr]map[lsdbKey]lsa),
		externalDB: make(map[lsdbKey]lsa),
	}
}

func (db *lsdb) set(areaID netip.Addr, l lsa) {
	key := lsdbKey{
		lsType: l.header().lsType,
		lsID:   l.header().lsID,
		advRtr: l.header().advertisingRouter,
	}

	if l.header().lsType == lsTypeASExternal {
		db.externalDB[key] = l
		return
	}

	if _, ok := db.areas[areaID]; !ok {
		db.areas[areaID] = make(map[lsdbKey]lsa)
	}

	db.areas[areaID][key] = l
}

func (db *lsdb) get(areaID netip.Addr, lsType lsType, lsID netip.Addr, advRtr netip.Addr) lsa {
	key := lsdbKey{
		lsType: lsType,
		lsID:   lsID,
		advRtr: advRtr,
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
		lsType: lsType,
		lsID:   lsID,
		advRtr: advRtr,
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
		_ = lsa.encode() // ensure checksum is in the header
		headers = append(headers, *lsa.header())
	}

	return headers
}

type lsa interface {
	header() *lsaHeader
	encode() []byte
}

type lsType uint8

const (
	lsTypeRouter      lsType = 1
	lsTypeNetwork     lsType = 2
	lsTypeSummary     lsType = 3
	lsTypeASBRSummary lsType = 4
	lsTypeASExternal  lsType = 5
)

type lsaHeader struct {
	age               uint16
	options           uint8
	lsType            lsType
	lsID              netip.Addr
	advertisingRouter netip.Addr
	seqNumber         int32
	lsChecksum        uint16
	length            uint16

	data []byte
}

const lsaHeaderSize = 20

func decodeLSAHeader(data []byte) (*lsaHeader, error) {
	if len(data) < lsaHeaderSize {
		return nil, fmt.Errorf("lsa header too short")
	}

	if data[3] < 1 || data[3] > 5 {
		return nil, fmt.Errorf("unknown LSA type %d", data[3])
	}

	length := binary.BigEndian.Uint16(data[18:20])

	return &lsaHeader{
		age:               binary.BigEndian.Uint16(data[0:2]),
		options:           data[2],
		lsType:            lsType(data[3]),
		lsID:              mustAddrFromSlice(data[4:8]),
		advertisingRouter: mustAddrFromSlice(data[8:12]),
		seqNumber:         int32(binary.BigEndian.Uint32(data[12:16])),
		lsChecksum:        binary.BigEndian.Uint16(data[16:18]),
		length:            length,
		data:              data[:length],
	}, nil
}

func (lsa *lsaHeader) encodeTo(data []byte) {
	if len(data) < lsaHeaderSize {
		panic("lsaHeader.encodeTo: data is too short")
	}

	binary.BigEndian.PutUint16(data[0:2], lsa.age)
	data[2] = lsa.options
	data[3] = byte(lsa.lsType)
	copy(data[4:8], to4(lsa.lsID))
	copy(data[8:12], to4(lsa.advertisingRouter))
	binary.BigEndian.PutUint32(data[12:16], uint32(lsa.seqNumber))
	binary.BigEndian.PutUint16(data[16:18], lsa.lsChecksum)
	binary.BigEndian.PutUint16(data[18:20], lsa.length)
}

func fletcherChecksum(data ...[]byte) uint16 {
	var c0, c1 byte

	for _, d := range data {
		for _, b := range d {
			c0 += b
			c1 += c0
		}
	}

	return uint16(c1)<<8 | uint16(c0)
}

type tosMetric struct {
	tos    uint8
	metric uint16
}

const tosMetricSize = 4

func decodeTosMetric(data []byte) (*tosMetric, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("tos metric too short")
	}

	return &tosMetric{
		tos:    data[0],
		metric: binary.BigEndian.Uint16(data[2:4]),
	}, nil
}

func (tm *tosMetric) encodeTo(data []byte) {
	if len(data) < 4 {
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
	lsaHeader
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

// TODO: LSA checksum

func newRouterLSA(inst *Instance, area *area) (*routerLSA, error) {
	lsa := routerLSA{
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

	linksSize := 0
	for _, link := range lsa.links {
		linksSize += link.size()
	}

	lsa.length = minRouterLSASize + uint16(linksSize)

	return &lsa, nil
}

func decodeRouterLSA(header *lsaHeader, data []byte) (*routerLSA, error) {
	if len(data) < minRouterLSASize {
		return nil, fmt.Errorf("router LSA too short")
	}

	lsa := &routerLSA{
		lsaHeader: *header,
		virtual:   data[20]&rlfVirtual != 0,
		external:  data[20]&rlfExternal != 0,
		border:    data[20]&rlfBorder != 0,
	}

	nLinks := binary.BigEndian.Uint16(data[22:24])

	for i := 0; i < int(nLinks); i++ {
		link, err := decodeLink(data[lsaHeaderSize+4+i*12:])
		if err != nil {
			return nil, err
		}

		lsa.links = append(lsa.links, *link)
	}

	return lsa, nil
}

func (lsa *routerLSA) header() *lsaHeader {
	return &lsa.lsaHeader
}

func (lsa *routerLSA) encode() []byte {
	size := minRouterLSASize

	for _, link := range lsa.links {
		size += link.size()
	}

	buf := make([]byte, size)

	lsa.lsaHeader.encodeTo(buf)

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

	lsa.lsChecksum = fletcherChecksum(buf[2:16], buf[18:])
	binary.BigEndian.PutUint16(buf[16:18], lsa.lsChecksum)

	lsa.lsaHeader.data = buf

	return buf
}

type networkLSA struct {
	lsaHeader
}

func decodeNetworkLSA(header *lsaHeader, data []byte) (*networkLSA, error) {
	return nil, fmt.Errorf("network LSA not implemented")
}

func (lsa *networkLSA) header() *lsaHeader {
	return &lsa.lsaHeader
}

func (lsa *networkLSA) encode() []byte {
	return nil
}

type summaryLSA struct {
	lsaHeader
}

func decodeSummaryLSA(header *lsaHeader, data []byte) (*summaryLSA, error) {
	return nil, fmt.Errorf("summary LSA not implemented")
}

func (lsa *summaryLSA) header() *lsaHeader {
	return &lsa.lsaHeader
}

func (lsa *summaryLSA) encode() []byte {
	return nil
}

type asbrSummaryLSA struct {
	lsaHeader
}

func decodeASBRSummaryLSA(header *lsaHeader, data []byte) (*asbrSummaryLSA, error) {
	return nil, fmt.Errorf("ASBR summary LSA not implemented")
}

func (lsa *asbrSummaryLSA) header() *lsaHeader {
	return &lsa.lsaHeader
}

func (lsa *asbrSummaryLSA) encode() []byte {
	return nil
}

type asExternalLSA struct {
	lsaHeader
}

func decodeASExternalLSA(header *lsaHeader, data []byte) (*asExternalLSA, error) {
	return nil, fmt.Errorf("AS external LSA not implemented")
}

func (lsa *asExternalLSA) header() *lsaHeader {
	return &lsa.lsaHeader
}

func (lsa *asExternalLSA) encode() []byte {
	return nil
}

func decodeLSA(data []byte) (lsa, error) {
	h, err := decodeLSAHeader(data)
	if err != nil {
		return nil, err
	}

	switch h.lsType {
	case lsTypeRouter:
		return decodeRouterLSA(h, data[:h.length])
	case lsTypeNetwork:
		return decodeNetworkLSA(h, data[:h.length])
	case lsTypeSummary:
		return decodeSummaryLSA(h, data[:h.length])
	case lsTypeASBRSummary:
		return decodeASBRSummaryLSA(h, data[:h.length])
	case lsTypeASExternal:
		return decodeASExternalLSA(h, data[:h.length])
	}

	return nil, fmt.Errorf("unknown LSA type %d", h.lsType)
}
