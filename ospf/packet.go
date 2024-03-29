package ospf

import "github.com/davidbalbert/chatter/chatterd/common"

type Packet interface {
}

type PacketHeader struct {
	t        packetType
	length   uint16
	routerID common.RouterID
	areaID   common.AreaID
	checksum uint16
	authType uint16
	authData uint64
}

type packetType uint8

const (
	pHello packetType = iota
	pDD
	pLSReq
	pLSUpd
	pLSAck
)

type Hello struct {
	PacketHeader
}

type DD struct {
	PacketHeader
}

type LSReq struct {
	PacketHeader
}

type LSUpd struct {
	PacketHeader
}

type LSAck struct {
	PacketHeader
}
