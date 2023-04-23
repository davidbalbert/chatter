package ospf

import (
	"context"
	"fmt"
	"net/netip"
	"time"

	"github.com/davidbalbert/chatter/chatterd/common"
	"github.com/davidbalbert/chatter/config"
)

type interfaceID struct {
	name   string
	prefix netip.Prefix
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

type Router struct {
	ID   common.RouterID
	Addr netip.Addr
}

func (r Router) IsValid() bool {
	return r.ID != 0
}

type AuthType int

const (
	authTypeNull AuthType = iota
	authTypePlain
	authTypeMD5
)

type dispatch struct {
	e interfaceEvent
	c chan struct{}
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

	HelloTimer *time.Timer
	WaitTimer  *time.Timer

	Neighbors map[common.RouterID]Neighbor
	DR        Router
	BDR       Router

	Cost              uint16
	RxmtInterval      int
	AuType            AuthType
	AuthenticationKey uint64

	name string

	events chan dispatch
}

func newInterface(conf config.OSPFInterfaceConfig, areaID common.AreaID, name string, prefix netip.Prefix) *Interface {
	helloTimer := time.NewTimer(0)
	if !helloTimer.Stop() {
		<-helloTimer.C
	}

	waitTimer := time.NewTimer(0)
	if !waitTimer.Stop() {
		<-waitTimer.C
	}

	return &Interface{
		Type:               InterfacePointToPoint,
		State:              iDown,
		Prefix:             prefix,
		HelloInterval:      conf.HelloInterval,
		RouterDeadInterval: conf.RouterDeadInterval,
		InfTransDelay:      1, // TODO: conf.InfTransDelay,
		RouterPriority:     1, // TODO: conf.RouterPriority,

		// Maybe these should be time.Tickers?
		HelloTimer: helloTimer,
		WaitTimer:  waitTimer,

		Neighbors:         make(map[common.RouterID]Neighbor),
		Cost:              conf.Cost,
		RxmtInterval:      5,            // TODO: conf.RxmtInterval,
		AuType:            authTypeNull, // TODO: conf.AuType,
		AuthenticationKey: 0,            // TODO: conf.AuthenticationKey,

		name: name,

		events: make(chan dispatch),
	}
}

func (i *Interface) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			if !i.HelloTimer.Stop() {
				select {
				case <-i.HelloTimer.C:
				default:
				}
			}

			if !i.WaitTimer.Stop() {
				select {
				case <-i.WaitTimer.C:
				default:
				}
			}

			return nil
		case <-i.HelloTimer.C:
			fmt.Printf("hello timer expired: %s %s\n", i.name, i.Prefix)
			i.HelloTimer.Reset(time.Duration(i.HelloInterval) * time.Second)
		case <-i.WaitTimer.C:
			fmt.Printf("wait timer expired: %s %s\n", i.name, i.Prefix)
		case d := <-i.events:
			i.handleEvent(d.e)

			if d.c != nil {
				d.c <- struct{}{}
			}
		}
	}
}

func (i *Interface) isUp() bool {
	return i.State != iDown
}

func (i *Interface) isLoopback() bool {
	return i.State == iLoopback
}

func (i *Interface) isPTP() bool {
	return i.Type == InterfacePointToPoint
}

func (i *Interface) isPTMP() bool {
	return i.Type == InterfacePointToMultipoint || i.Type == InterfacePointToMultipointBroadcast
}

func (i *Interface) isVirtualLink() bool {
	return i.Type == InterfaceVirtualLink
}

func (i *Interface) handleEvent(e interfaceEvent) {
	fmt.Printf("interface event: %s %s: %s\n", i.name, i.Prefix, e)
	switch e {
	case ieInterfaceUp:
		i.HelloTimer.Reset(time.Duration(i.HelloInterval) * time.Second)

		if i.isPTP() || i.isPTMP() || i.isVirtualLink() {
			i.State = iPointToPoint
		} else if i.RouterPriority == 0 {
			// > Else, if the router is not eligible to
			// > become Designated Router the interface state
			// > transitions to DR Other.
			//
			// TODO: confirm that this is what the above means
			i.State = iDROther
		} else {
			i.WaitTimer.Reset(time.Duration(i.InfTransDelay) * time.Second)
			i.State = iWaiting

			// TODO:
			// > Additionally, if the
			// > network is an NBMA network examine the configured
			// > list of neighbors for this interface and generate
			// > the neighbor event Start for each neighbor that is
			// > also eligible to become Designated Router.
		}

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

func (i *Interface) sendEvent(e interfaceEvent) {
	i.events <- dispatch{e, nil}
}

func (i *Interface) sendEventWait(e interfaceEvent) {
	c := make(chan struct{})
	i.events <- dispatch{e, c}
	<-c
}
