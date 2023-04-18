package ospf

import (
	"encoding/binary"
	"math"
	"net/netip"
	"time"

	"github.com/davidbalbert/chatter/chatterd/common"
)

const (
	initialSequenceNumber = math.MinInt32 + 1
	maxSequenceNumber     = math.MaxInt32
	maxAge                = 3600 // 1 hour
	maxAgeDiff            = 900  // 15 minutes
	minLSArrival          = 1    // 1 second
)

type LSAMetadata interface {
	Age() uint16
	Options() uint8
	Type() lsType
	ID() netip.Addr
	AdvertisingRouter() common.RouterID
	SequenceNumber() int32
	Checksum() uint16
	Length() uint16

	Compare(LSAMetadata) int
	Key() lsdbKey
}

type LSA interface {
	LSAMetadata
	SetAge(uint16)
	Bytes() []byte
	IsChecksumValid() bool
}

type lsdbKey struct {
	Type              lsType
	ID                netip.Addr
	AdvertisingRouter common.RouterID
}

type lsdb map[lsdbKey]*installedLSA

type installedLSA struct {
	LSA
	installedAt time.Time
}

func newLSDB() lsdb {
	return lsdb(make(map[lsdbKey]*installedLSA))
}

// Rules of LSAs:
//
// - With the exception of age, all fields are immutable.
// - Age must be changed using SetAge(), which will also update the encoded bytes.
// - lsaBase.bytes and lsaHeader.bytes are never nil.

type lsaHeader struct {
	age               uint16
	options           uint8
	type_             lsType
	id                netip.Addr
	advertisingRouter common.RouterID
	sequenceNumber    int32
	checksum          uint16
	length            uint16

	bytes []byte // the encoded header
}

const lsaHeaderLen = 20

type lsType uint8

const (
	lsTypeUnknown lsType = 0

	lsTypeRouter      lsType = 1
	lsTypeNetwork     lsType = 2
	lsTypeSummary     lsType = 3
	lsTypeASBRSummary lsType = 4
	lsTypeASExternal  lsType = 5
)

type lsaBase struct {
	lsaHeader
	bytes []byte // the entire LSA, including the header
}

func (h *lsaHeader) Age() uint16 {
	return h.age
}

func (h *lsaHeader) Options() uint8 {
	return h.options
}

func (h *lsaHeader) Type() lsType {
	return h.type_
}

func (h *lsaHeader) ID() netip.Addr {
	return h.id
}

func (h *lsaHeader) AdvertisingRouter() common.RouterID {
	return h.advertisingRouter
}

func (h *lsaHeader) SequenceNumber() int32 {
	return h.sequenceNumber
}

func (h *lsaHeader) Checksum() uint16 {
	return h.checksum
}

func (h *lsaHeader) Length() uint16 {
	return h.length
}

func (h *lsaHeader) Compare(other LSAMetadata) int {
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

func (h *lsaHeader) Key() lsdbKey {
	return lsdbKey{
		Type:              h.type_,
		ID:                h.id,
		AdvertisingRouter: h.advertisingRouter,
	}
}

func (base *lsaBase) SetAge(age uint16) {
	base.age = age
	binary.BigEndian.PutUint16(base.bytes[0:2], age)
}

func (base *lsaBase) Bytes() []byte {
	return base.bytes
}

func (base *lsaBase) IsChecksumValid() bool {
	return fletcher16Checksum(base.bytes[2:]) == 0
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
func fletcher16GenerateChecksum(data []byte, offset int) uint16 {
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

type routerLSA struct {
	lsaBase
}

type networkLSA struct {
	lsaBase
}

type summaryLSA struct {
	lsaBase
}

type asbrSummaryLSA struct {
	lsaBase
}

type asExternalLSA struct {
	lsaBase
}
