package ospf

import "net/netip"

type Interface struct {
	Type               interfaceType
	State              interfaceState
	Prefix             netip.Prefix // IP interface address and IP interface mask
	AreaID             AreaID
	HelloInterval      uint16
	RouterDeadInterval uint16
	InfTransDelay      int
	RouterPriority     uint8

	// TODO: HelloTimer
	// TODO: WaitTimer

	Neighbors []Neighbor // TODO: probably change to map
	DRID      RouterID
	DRAddr    netip.Addr
	BDRID     RouterID
	BDRAddr   netip.Addr

	Cost              uint16
	RxmtInterval      int
	AuType            uint16
	AuthenticationKey uint64
}

type interfaceType int

const (
	InterfacePointToPoint interfaceType = iota
	InterfaceBroadcast
	InterfaceNBMA
	InterfacePointToMultipoint
	InterfacePointToMultipointBroadcast
	InterfaceVirtualLink
)

func (it interfaceType) String() string {
	switch it {
	case InterfacePointToPoint:
		return "Point-to-point"
	case InterfaceBroadcast:
		return "Broadcast"
	case InterfaceNBMA:
		return "NBMA"
	case InterfacePointToMultipoint:
		return "Point-to-MultiPoint"
	case InterfacePointToMultipointBroadcast:
		return "Point-to-MultiPoint Broadcast"
	case InterfaceVirtualLink:
		return "Virtual Link"
	default:
		return "Unknown"
	}
}

type interfaceState int

const (
	iDown interfaceState = iota
	iLoopback
	iWaiting
	iPointToPoint
	iDROther
	iBackup
	iDR
)

func (is interfaceState) String() string {
	switch is {
	case iDown:
		return "Down"
	case iLoopback:
		return "Loopback"
	case iWaiting:
		return "Waiting"
	case iPointToPoint:
		return "Point-to-point"
	case iDROther:
		return "DROther"
	case iBackup:
		return "Backup"
	case iDR:
		return "DR"
	default:
		return "Unknown"
	}
}

type interfaceEvent int

const (
	ieInterfaceUp interfaceEvent = iota
	ieWaitTimer
	ieBackupSeen
	ieNeighborChange
	ieLoopInd
	ieUnloopInd
	ieInterfaceDown
)

func (ie interfaceEvent) String() string {
	switch ie {
	case ieInterfaceUp:
		return "InterfaceUp"
	case ieWaitTimer:
		return "WaitTimer"
	case ieBackupSeen:
		return "BackupSeen"
	case ieNeighborChange:
		return "NeighborChange"
	case ieLoopInd:
		return "LoopInd"
	case ieUnloopInd:
		return "UnloopInd"
	case ieInterfaceDown:
		return "InterfaceDown"
	default:
		return "Unknown"
	}
}
