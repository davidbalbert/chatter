package ospf

import (
	"fmt"
	"net/netip"

	"github.com/davidbalbert/chatter/chatterd/common"
	"github.com/davidbalbert/chatter/config"
)

type interfaceID struct {
	name   string
	prefix netip.Prefix
}

type Router struct {
	ID   common.RouterID
	Addr netip.Addr
}

func (r Router) IsValid() bool {
	return r.ID != 0
}

type Interface struct {
	Type               interfaceType
	State              interfaceState
	Prefix             netip.Prefix // IP interface address and IP interface mask
	AreaID             common.AreaID
	HelloInterval      uint16
	RouterDeadInterval uint32
	InfTransDelay      int
	RouterPriority     uint8

	// TODO: HelloTimer
	// TODO: WaitTimer

	Neighbors map[common.RouterID]Neighbor
	DR        Router
	BDR       Router

	Cost              uint16
	RxmtInterval      int
	AuType            uint16
	AuthenticationKey uint64

	name string
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

func newInterface(conf config.OSPFInterfaceConfig, areaID common.AreaID, name string, prefix netip.Prefix) *Interface {
	return &Interface{
		Type:               InterfacePointToPoint,
		State:              iDown,
		Prefix:             prefix,
		HelloInterval:      conf.HelloInterval,
		RouterDeadInterval: conf.RouterDeadInterval,
		// InfTransDelay:      conf.InfTransDelay,
		// RouterPriority:     conf.RouterPriority,
		Neighbors: make(map[common.RouterID]Neighbor),
		Cost:      conf.Cost,
		// RxmtInterval:       conf.RxmtInterval,
		// AuType:             conf.AuType,
		// AuthenticationKey:  conf.AuthenticationKey,

		name: name,
	}
}

func (i *Interface) isUp() bool {
	return i.State != iDown
}

func (i *Interface) isLoopback() bool {
	return i.State == iLoopback
}

func (i *Interface) handleEvent(e interfaceEvent) {
	fmt.Printf("interface event: %s %s: %s\n", i.name, i.Prefix, e)
	switch e {
	case ieInterfaceUp:
		i.State = iPointToPoint
	case ieWaitTimer:
		// TODO
	case ieBackupSeen:
		// TODO
	case ieNeighborChange:
		// TODO
	case ieLoopInd:
		i.State = iLoopback
	case ieUnloopInd:
		i.State = iPointToPoint
	case ieInterfaceDown:
		i.State = iDown
	}
}
