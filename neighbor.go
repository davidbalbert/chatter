package main

import (
	"fmt"
	"net/netip"
	"time"
)

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

type Neighbor struct {
	iface *Interface
	state neighborState
	// TODO: inactivityTimer
	master           bool
	ddSequenceNumber uint32
	lastDD           *databaseDescriptionPacket
	neighborID       netip.Addr
	routerPriority   uint8
	addr             netip.Addr
	options          uint8
	dRouter          netip.Addr
	bdRouter         netip.Addr

	firstAdjancencyAttempt bool

	events  chan neighborEvent
	packets chan Packet

	done chan struct{}

	rxmtTimer *time.Timer
}

func newNeighbor(iface *Interface, h *helloPacket) *Neighbor {
	n := Neighbor{
		iface:            iface,
		state:            nDown,
		master:           true,
		ddSequenceNumber: 0,
		lastDD:           nil,
		neighborID:       h.routerID,
		routerPriority:   h.routerPriority,
		addr:             h.src,
		options:          h.options,
		dRouter:          h.dRouter,
		bdRouter:         h.bdRouter,

		firstAdjancencyAttempt: true,

		events:  make(chan neighborEvent),
		packets: make(chan Packet),

		done: make(chan struct{}),

		rxmtTimer: time.NewTimer(iface.RxmtDuration()),
	}

	n.stopRxmtTimer()

	return &n
}

func (n *Neighbor) setState(state neighborState) {
	if n.state == state {
		return
	}

	fmt.Printf("%v: state %v -> %v\n", n.neighborID, n.state, state)
	n.state = state

	switch n.state {
	case nDown:
		n.iface.rm <- n
	}
}

func (n *Neighbor) handleCommonEvents(event neighborEvent) (handled bool) {
	switch event {
	case neKillNbr:
		n.flushLSAs()
		n.disableInactivityTimer()
		n.setState(nDown)
		return true
	case neLLDown:
		n.flushLSAs()
		n.disableInactivityTimer()
		n.setState(nDown)
		return true
	case neInactivityTimer:
		n.flushLSAs()
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
	fmt.Printf("%v: event %v state %v\n", n.neighborID, event, n.state)

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
			if !n.shouldBecomeAdjacent() {
				n.setState(n2Way)
				return
			}

			n.setState(nExStart)

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

			if n.firstAdjancencyAttempt {
				now := time.Now()
				h, m, s := now.Clock()
				ns := now.Nanosecond()
				ms := ns / 1_000_000
				n.ddSequenceNumber = uint32(h*3600*1000 + m*60*1000 + s*1000 + ms)

				n.firstAdjancencyAttempt = false
			}

			n.ddSequenceNumber++
			n.master = true

			n.sendExStartDatabaseDescriptionPacket()
			n.startRxmtTimer()
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state Init\n", n.neighborID, event)
		}
	case n2Way:
		// TODO
	case nExStart:
		switch event {
		case neNegotiationDone:
			n.setState(nExchange)
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state ExStart\n", n.neighborID, event)
		}
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

func (n *Neighbor) handleHello(h *helloPacket) {
	// NOOP
}

func (n *Neighbor) handleDatabaseDescriptionInExStart(dd *databaseDescriptionPacket) {
	if dd.init && dd.more && dd.master && len(dd.lsaHeaders) == 0 && n.iface.routerID().Less(n.neighborID) {

		n.handleEvent(neNegotiationDone)
		n.options = dd.options

		n.master = false
		n.ddSequenceNumber = dd.sequenceNumber

		fmt.Printf("%v: ExStart became slave\n", n.neighborID)
		n.stopRxmtTimer()

		// TODO: send LSA headers in our acknowledgement
		n.send(newDatabaseDescription(n.iface, n.ddSequenceNumber, false, false, false, nil))
	} else if !dd.init && !dd.master && dd.sequenceNumber == n.ddSequenceNumber && n.neighborID.Less(n.iface.routerID()) {
		fmt.Printf("%v: ExStart became master\n", n.neighborID)

		n.handleEvent(neNegotiationDone)
		n.options = dd.options

		n.resetRxmtTimer()

		// TODO: start polling for DD packets
	}
}

func (n *Neighbor) handleDatabaseDescription(dd *databaseDescriptionPacket) {
	switch n.state {
	case nDown:
		fmt.Printf("%v: neighbor state machine: unexpected database description packet in state Down\n", n.neighborID)
	case nAttempt:
		fmt.Printf("%v: neighbor state machine: unexpected database description packet in state Attempt\n", n.neighborID)
	case nInit:
		n.handleEvent(ne2WayReceived)

		if n.state != nExStart {
			return
		}

		n.handleDatabaseDescriptionInExStart(dd)
	case n2Way:
		fmt.Printf("%v: neighbor state machine: unexpected database description packet in state 2-Way\n", n.neighborID)
	case nExStart:
		n.handleDatabaseDescriptionInExStart(dd)
	case nExchange:
		// TODO
	case nLoading, nFull:
		// TODO
	default:
		fmt.Printf("handleDatabaseDescription: unexpected state %v\n", n.state)
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

func (n *Neighbor) stopRxmtTimer() {
	fmt.Printf("%v: stopping rxmt timer\n", n.neighborID)
	if !n.rxmtTimer.Stop() {
		<-n.rxmtTimer.C
	}
}

func (n *Neighbor) startRxmtTimer() {
	fmt.Printf("%v: starting rxmt timer\n", n.neighborID)
	n.rxmtTimer.Reset(n.iface.RxmtDuration())
}

func (n *Neighbor) resetRxmtTimer() {
	if !n.rxmtTimer.Stop() {
		<-n.rxmtTimer.C
	}
	n.rxmtTimer.Reset(n.iface.RxmtDuration())
}

func (n *Neighbor) handleRxmtTimer() {
	fmt.Printf("%v: rxmt timer expired\n", n.neighborID)
	switch n.state {
	case nExStart:
		n.sendExStartDatabaseDescriptionPacket()
	case nExchange:
		// TODO
		fmt.Printf("%v: state=Exchange handleRxmtTimer: not implemented\n", n.neighborID)
	default:
		fmt.Printf("handleRxmtTimer: unexpected state %v\n", n.state)
	}

	n.rxmtTimer.Reset(n.iface.RxmtDuration())
}

func (n *Neighbor) sendExStartDatabaseDescriptionPacket() {
	n.send(newDatabaseDescription(n.iface, n.ddSequenceNumber, true, true, n.master, nil))
}

func (n *Neighbor) send(p Packet) {
	n.iface.send(n.addr, p)
}

func (n *Neighbor) shouldBecomeAdjacent() bool {
	t := n.iface.networkType

	// TODO: brodcast and nbma networks
	return t == networkPointToPoint || t == networkPointToMultipoint || t == networkVirtualLink
}

func (n *Neighbor) sendEvent(event neighborEvent) {
	n.events <- event
}

func (n *Neighbor) sendPacket(packet Packet) {
	n.packets <- packet
}

func (n *Neighbor) run() {
	for {
		select {
		case event := <-n.events:
			n.handleEvent(event)
		case packet := <-n.packets:
			fmt.Printf("neighbor: received packet: %v\n", packet)
			packet.handleOn(n)
		case <-n.rxmtTimer.C:
			n.handleRxmtTimer()
		case <-n.done:
			return
		}
	}
}

func (n *Neighbor) shutdown() {
	close(n.done)
}
