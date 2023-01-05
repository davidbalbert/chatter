package ospf

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"strconv"
)

var (
	AllSPFRouters = netip.MustParseAddr("224.0.0.5")
	AllDRouters   = netip.MustParseAddr("224.0.0.6")
)

type RouterID uint32
type AreaID uint32

func parseID(s string) (uint32, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err == nil {
		return uint32(n), nil
	}

	addr, err := netip.ParseAddr(s)
	if err != nil || !addr.Is4() {
		return 0, fmt.Errorf("must be an IPv4 address or an unsigned 32 bit integer")
	}

	return binary.BigEndian.Uint32(addr.AsSlice()), nil
}

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
