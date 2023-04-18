package common

import (
	"encoding/binary"
	"net/netip"
)

type RouterID uint32
type AreaID uint32

func (r RouterID) String() string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(r))
	addr := netip.AddrFrom4(b)

	return addr.String()
}

func (a AreaID) String() string {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(a))
	addr := netip.AddrFrom4(b)

	return addr.String()
}
