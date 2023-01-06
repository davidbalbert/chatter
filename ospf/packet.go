package ospf

type Packet interface {
}

type PacketHeader struct {
	type_    packetType
	length   uint16
	routerID RouterID
	areaID   AreaID
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
