package main

import (
	"fmt"
	"net/netip"
	"time"
)

type Neighbor struct {
	iface          *Interface
	neighborID     netip.Addr
	ip             netip.Addr
	routerPriority uint8
	dRouter        netip.Addr
	bdRouter       netip.Addr
	state          neighborState

	key netip.Addr // the key of the neighbor in the interface's neighbor map

	previouslyAdjacent bool
	ddSequenceNumber   uint32
	master             bool
}

type neighborState int

const (
	nDown neighborState = iota
	nAttempt
	nInit
	n2Way
	nExStart
	nExchange
	nLoading
	nFull
)

func (s neighborState) String() string {
	switch s {
	case nDown:
		return "Down"
	case nAttempt:
		return "Attempt"
	case nInit:
		return "Init"
	case n2Way:
		return "2-Way"
	case nExStart:
		return "ExStart"
	case nExchange:
		return "Exchange"
	case nLoading:
		return "Loading"
	case nFull:
		return "Full"
	default:
		return "Unknown"
	}
}

type neighborEvent int

const (
	neHelloReceived neighborEvent = iota
	neStart
	ne2WayReceived
	neNegotiationDone
	neExchangeDone
	neBadLSReq
	neLoadingDone
	neAdjOK
	neSeqNumberMismatch
	ne1WayReceived
	neKillNbr
	neInactivityTimer
	neLLDown
)

func (e neighborEvent) String() string {
	switch e {
	case neHelloReceived:
		return "HelloReceived"
	case neStart:
		return "Start"
	case ne2WayReceived:
		return "2-WayReceived"
	case neNegotiationDone:
		return "NegotiationDone"
	case neExchangeDone:
		return "ExchangeDone"
	case neBadLSReq:
		return "BadLSReq"
	case neLoadingDone:
		return "LoadingDone"
	case neAdjOK:
		return "AdjOK?"
	case neSeqNumberMismatch:
		return "SeqNumberMismatch"
	case ne1WayReceived:
		return "1-WayReceived"
	case neKillNbr:
		return "KillNbr"
	case neInactivityTimer:
		return "InactivityTimer"
	case neLLDown:
		return "LLDown"
	default:
		return "Unknown"
	}
}

func (n *Neighbor) setState(state neighborState) {
	fmt.Printf("%v: state %v -> %v\n", n.neighborID, n.state, state)
	n.state = state
}

func (n *Neighbor) handleCommonEvents(event neighborEvent) (handled bool) {
	switch event {
	case neKillNbr:
		n.flushLSAs()
		n.disableInactivityTimer()
		n.iface.removeNeighbor(n)
		n.setState(nDown)
		return true
	case neLLDown:
		n.flushLSAs()
		n.disableInactivityTimer()
		n.iface.removeNeighbor(n)
		n.setState(nDown)
		return true
	case neInactivityTimer:
		n.flushLSAs()
		n.iface.removeNeighbor(n)
		n.setState(nDown)
		return true
	default:
		return false
	}
}

func (n *Neighbor) handleCommonExchangeEvents(event neighborEvent) (handled bool) {
	switch event {
	case neSeqNumberMismatch:
		n.flushLSAs()
		// TODO:
		// 4. Increment DD sequence number
		// 5. Declare ourselves the master
		// 6. Start sending DD packets with I, M and MS set.
		n.setState(nExStart)
		return true
	case neBadLSReq:
		n.flushLSAs()
		n.flushLSAs()
		// TODO:
		// 4. Increment DD sequence number
		// 5. Declare ourselves the master
		// 6. Start sending DD packets with I, M and MS set.
		n.setState(nExStart)
		return true
	case ne1WayReceived:
		n.flushLSAs()

		// We should also do the above if we're in 2-Way too. Don't forget.
		n.setState(nAttempt)
		return true
	case neAdjOK:
		// Event AdjOK? leads to adjacency forming/breaking
		// not sure what to do here yet
		return false
	default:
		return false
	}
}

func (n *Neighbor) handleEvent(event neighborEvent) {
	fmt.Printf("%v: event %v\n", n.neighborID, event)

	if n.handleCommonEvents(event) {
		return
	}

	if n.state >= nExStart && n.handleCommonExchangeEvents(event) {
		return
	}

	switch n.state {
	case nDown:
		switch event {
		case neStart:
			// NBMA only
			// TODO: Send hello packet to the neighbor
			n.startInactivityTimer()
			n.setState(nAttempt)
		case neHelloReceived:
			n.startInactivityTimer()
			n.setState(nInit)
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state Down\n", n.neighborID, event)
		}
	case nAttempt:
		switch event {
		case neHelloReceived:
			n.restartInactivityTimer()
			n.setState(nInit)
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state Attempt\n", n.neighborID, event)
		}
	case nInit:
		switch event {
		case neHelloReceived:
			n.restartInactivityTimer()
			n.setState(nInit)
		case ne1WayReceived:
			// do nothing
		case ne2WayReceived:
			if n.shouldBecomeAdjacent() {
				n.setState(nExStart)
			} else {
				n.setState(n2Way)
			}
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state Init\n", n.neighborID, event)
		}
	case n2Way:
		// TODO?
	case nExStart:
		// Upon entering this state, the router increments the DD
		// sequence number in the neighbor data structure.  If
		// this is the first time that an adjacency has been
		// attempted, the DD sequence number should be assigned
		// some unique value (like the time of day clock).  It
		// then declares itself master (sets the master/slave
		// bit to master), and starts sending Database
		// Description Packets, with the initialize (I), more
		// (M) and master (MS) bits set.  This Database
		// Description Packet should be otherwise empty.  This
		// Database Description Packet should be retransmitted
		// at intervals of RxmtInterval until the next state is
		// entered (see Section 10.8).

		if !n.previouslyAdjacent {
			n.ddSequenceNumber = uint32(time.Now().Unix())
		}

		n.ddSequenceNumber++
		n.master = true

	case nExchange:
		// TODO
	case nLoading:
		// TODO
	case nFull:
		// TODO
	default:
		fmt.Printf("neighbor state machine: unexpected state %v\n", n.state)
	}
}

func NewNeighbor(source netip.Addr, iface *Interface, h *Hello) *Neighbor {
	return &Neighbor{
		iface:          iface,
		neighborID:     h.routerID,
		ip:             h.src,
		routerPriority: h.routerPriority,
		dRouter:        h.dRouter,
		bdRouter:       h.bdRouter,
		state:          nDown,

		key: source,
	}
}

func (n *Neighbor) flushLSAs() {
	// TODO:
	// 1. Flush the retransmission list.
	// 2. Flush the database summary list.
	// 3. Flush the link state request list
}

func (n *Neighbor) startInactivityTimer() {
	// TODO
}

func (n *Neighbor) restartInactivityTimer() {
	// TODO
}

func (n *Neighbor) disableInactivityTimer() {
	// TODO
}

func (n *Neighbor) shouldBecomeAdjacent() bool {
	t := n.iface.networkType

	// TODO: brodcast and nbma networks
	return t == networkPointToPoint || t == networkPointToMultipoint || t == networkVirtualLink
}
