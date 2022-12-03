package main

import (
	"fmt"
	"net/netip"
	"reflect"
	"runtime"
	"strings"
)

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

func stateName(state neighborState) string {
	name := runtime.FuncForPC(reflect.ValueOf(state).Pointer()).Name()
	unqualifiedName := name[strings.LastIndex(name, ".")+1:]

	return strings.TrimPrefix(unqualifiedName, "neighbor")
}

type neighborState func(*Neighbor) neighborState

func neighborDown(n *Neighbor) neighborState {
	for event := range n.events {
		fmt.Printf("%v: event %v\n", n.neighborID, event)

		if nextState := n.handleCommonHelloEvents(event); nextState != nil {
			return nextState
		}

		switch event {
		case neStart:
			// NBMA only
			// TODO: Send hello packet to the neighbor
			n.startInactivityTimer()
			return neighborAttempt
		case neHelloReceived:
			n.startInactivityTimer()
			return neighborInit
		default:
			fmt.Printf("neighbor state machine: unexpected event %v in state Down\n", event)
		}
	}

	return nil
}

func neighborAttempt(n *Neighbor) neighborState {
	for event := range n.events {
		fmt.Printf("%v: event %v\n", n.neighborID, event)

		if nextState := n.handleCommonHelloEvents(event); nextState != nil {
			return nextState
		}

		switch event {
		case neHelloReceived:
			n.restartInactivityTimer()
			return neighborInit
		default:
			fmt.Printf("neighbor state machine: unexpected event %v in state Attempt\n", event)
		}
	}

	return nil
}

func neighborInit(n *Neighbor) neighborState {
	for event := range n.events {
		fmt.Printf("%v: event %v\n", n.neighborID, event)

		if nextState := n.handleCommonHelloEvents(event); nextState != nil {
			return nextState
		}

		switch event {
		case neHelloReceived:
			n.restartInactivityTimer()
			return neighborInit
		case ne1WayReceived:
			// do nothing
		case ne2WayReceived:
			// if adjacencyShouldBeEstablished {

			// 	return neighborExStart
			// } else {
			return neighbor2Way
			// }
		default:
			fmt.Printf("neighbor state machine: unexpected event %v in state Init\n", event)
		}

	}

	return nil
}

func neighbor2Way(n *Neighbor) neighborState {
	return nil
}

func neighborExStart(n *Neighbor) neighborState {
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

	return nil
}

func neighborExchange(n *Neighbor) neighborState {
	return nil
}

func neighborLoading(n *Neighbor) neighborState {
	return nil
}

func neighborFull(n *Neighbor) neighborState {
	return nil
}

type Neighbor struct {
	iface          *Interface
	neighborID     netip.Addr
	addr           netip.Addr
	routerPriority uint8
	dRouter        netip.Addr
	bdRouter       netip.Addr
	events         chan neighborEvent
	stateName      string
}

func NewNeighbor(iface *Interface, h *Hello) *Neighbor {
	return &Neighbor{
		iface:          iface,
		neighborID:     h.routerID,
		addr:           h.src,
		routerPriority: h.routerPriority,
		dRouter:        h.dRouter,
		bdRouter:       h.bdRouter,
		events:         make(chan neighborEvent),
		stateName:      "Down",
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

// The common event handlers are methods, not states. Unlike state
// functions, where a nil return value means end execution, a nil return
// value here means continue processing the event in the state we're in
func (n *Neighbor) handleCommonHelloEvents(event neighborEvent) neighborState {
	switch event {
	case neKillNbr:
		n.flushLSAs()
		n.disableInactivityTimer()
		return neighborDown
	case neLLDown:
		n.flushLSAs()
		n.disableInactivityTimer()
		return neighborDown
	case neInactivityTimer:
		n.flushLSAs()
		return neighborDown
	default:
		return nil
	}
}

func (n *Neighbor) handleCommonExchangeEvents(event neighborEvent) neighborState {
	switch event {
	case neSeqNumberMismatch:
		n.flushLSAs()
		// TODO:
		// 4. Increment DD sequence number
		// 5. Declare ourselves the master
		// 6. Start sending DD packets with I, M and MS set.
		return neighborExStart
	case neBadLSReq:
		n.flushLSAs()
		// TODO:
		// 4. Increment DD sequence number
		// 5. Declare ourselves the master
		// 6. Start sending DD packets with I, M and MS set.
		return neighborExStart
	case ne1WayReceived:
		n.flushLSAs()

		// We should also do the above if we're in 2-Way too. Don't forget.
		return neighborInit
	case neAdjOK:
		// Event AdjOK? leads to adjacency forming/breaking
		// not sure what to do here yet
		return nil
	case neKillNbr:
		n.flushLSAs()
		n.disableInactivityTimer()
		return neighborDown
	case neLLDown:
		n.flushLSAs()
		n.disableInactivityTimer()
		return neighborDown
	case neInactivityTimer:
		n.flushLSAs()
		return neighborDown
	default:
		return nil
	}
}

func (n *Neighbor) run() {
	for state := neighborDown; state != nil; {
		state = state(n)
		oldStateName := n.stateName
		n.stateName = stateName(state)
		fmt.Printf("%v: neighbor state machine: %v -> %v\n", n.neighborID, oldStateName, n.stateName)
	}
}

func (n *Neighbor) executeEvent(event neighborEvent) {
	n.events <- event
}
