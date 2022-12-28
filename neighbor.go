package main

import (
	"fmt"
	"net/netip"
	"time"

	"golang.org/x/net/ipv4"
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
	lastReceivedDD   *databaseDescriptionPacket
	neighborID       netip.Addr
	routerPriority   uint8
	addr             netip.Addr
	options          uint8
	dRouter          netip.Addr
	bdRouter         netip.Addr

	databaseSummaryList  []lsaHeader
	linkStateRequestList []lsaHeader

	firstAdjancencyAttempt bool
	outstandingLSReq       *linkStateRequestPacket

	events  chan neighborEvent
	packets chan Packet

	done chan struct{}

	ddRxmtTimer    *time.Timer
	lsReqRxmtTimer *time.Timer
}

func newNeighbor(iface *Interface, h *helloPacket) *Neighbor {
	n := Neighbor{
		iface:            iface,
		state:            nDown,
		master:           true,
		ddSequenceNumber: 0,
		lastReceivedDD:   nil,
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

		ddRxmtTimer:    time.NewTimer(iface.RxmtDuration()),
		lsReqRxmtTimer: time.NewTimer(iface.RxmtDuration()),
	}

	n.stopDDRxmtTimer()

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
	case neKillNbr, neLLDown:
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
	case neSeqNumberMismatch, neBadLSReq:
		n.flushLSAs()
		n.ddSequenceNumber++
		n.master = true

		n.setState(nExStart)

		n.sendDatabaseDescription()
		n.startDDRxmtTimer()

		return true
	case ne1WayReceived:
		n.flushLSAs()
		n.setState(nInit)
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

			n.sendDatabaseDescription()
			n.startDDRxmtTimer()
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state Init\n", n.neighborID, event)
		}
	case n2Way:
		switch event {
		case ne1WayReceived:
			n.flushLSAs()
			n.setState(nInit)
		case ne2WayReceived:
			// NOOP
		case neAdjOK:
			// TODO
			fmt.Printf("%v: UNHANDLED neighbor state machine: AdjOK? in state 2Way\n", n.neighborID)
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state 2Way\n", n.neighborID, event)
		}
	case nExStart:
		switch event {
		case neNegotiationDone:
			n.databaseSummaryList = n.iface.instance.db.copyHeaders(n.iface.areaID)
			n.setState(nExchange)
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state ExStart\n", n.neighborID, event)
		}
	case nExchange:
		switch event {
		case neExchangeDone:
			if len(n.linkStateRequestList) > 0 {
				n.setState(nLoading)
				// TODO: keep sending LSReq packets until the list is empty
			} else {
				n.setState(nFull)
			}
		default:
			fmt.Printf("%v: neighbor state machine: unexpected event %v in state Exchange\n", n.neighborID, event)
		}
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
	// TODO: maybe move some code from Interface.handleHello here.
}

func (n *Neighbor) handleDatabaseDescriptionInExStart(dd *databaseDescriptionPacket) {
	if dd.init && dd.more && dd.master && len(dd.lsaHeaders) == 0 && n.iface.routerID().Less(n.neighborID) {
		fmt.Printf("%v: ExStart became slave\n", n.neighborID)

		n.handleEvent(neNegotiationDone)
		n.options = dd.options

		n.master = false
		n.ddSequenceNumber = dd.sequenceNumber

		n.stopDDRxmtTimer()
		n.sendDatabaseDescription()
	} else if !dd.init && !dd.master && dd.sequenceNumber == n.ddSequenceNumber && n.neighborID.Less(n.iface.routerID()) {
		fmt.Printf("%v: ExStart became master\n", n.neighborID)

		n.handleEvent(neNegotiationDone)
		n.options = dd.options

		n.handleDatabaseDescriptionInExchange(dd)
	}
}

func (n *Neighbor) isDuplicateDatabaseDescription(dd *databaseDescriptionPacket) bool {
	prev := n.lastReceivedDD

	if prev == nil {
		return false
	}

	return prev.init == dd.init && prev.more == dd.more && prev.master == dd.master && prev.options == dd.options && prev.sequenceNumber == dd.sequenceNumber
}

func (n *Neighbor) handleDatabaseDescriptionInExchangeMaster(dd *databaseDescriptionPacket) {
	if n.isDuplicateDatabaseDescription(dd) {
		fmt.Printf("%v: received duplicate database description packet, discarding\n", n.neighborID)
		return
	}

	if dd.sequenceNumber != n.ddSequenceNumber {
		fmt.Printf("%v: received database description packet with unexpected sequence number, discarding\n", n.neighborID)
		n.handleEvent(neSeqNumberMismatch)
		return
	}

	for _, h := range dd.lsaHeaders {
		// TODO: also reject if it's a Type 5 LSA and we're in a stub area.
		if h.lsType < 1 || h.lsType > 5 {
			fmt.Printf("%v: received database description packet with invalid LSA header, discarding\n", n.neighborID)
			n.handleEvent(neSeqNumberMismatch)
			return
		}

		db := n.iface.instance.db
		existing := db.get(n.iface.areaID, h.lsType, h.lsID, h.advertisingRouter)
		if existing != nil && existing.header().age >= h.age {
			continue
		}

		n.linkStateRequestList = append(n.linkStateRequestList, h)
	}

	n.lastReceivedDD = dd
	n.ddSequenceNumber++
	nHeadersInLastPacket := len(n.lsaHeadersForDatabaseDescription())
	n.databaseSummaryList = n.databaseSummaryList[nHeadersInLastPacket:]

	if !dd.more && len(n.databaseSummaryList) == 0 {
		n.stopDDRxmtTimer()
		n.handleEvent(neExchangeDone)
	} else {
		n.resetRxmtTimer()
		n.sendDatabaseDescription()
	}
}

func (n *Neighbor) handleDatabaseDescriptionInExchangeSlave(dd *databaseDescriptionPacket) {
	if n.isDuplicateDatabaseDescription(dd) {
		fmt.Printf("%v: received duplicate database description packet, retransmitting\n", n.neighborID)
		n.sendDatabaseDescription()
		return
	}

	if dd.sequenceNumber != n.ddSequenceNumber+1 {
		fmt.Printf("%v: received database description packet with unexpected sequence number, discarding\n", n.neighborID)
		n.handleEvent(neSeqNumberMismatch)
		return
	}

	for _, h := range dd.lsaHeaders {
		// TODO: also reject if it's a Type 5 LSA and we're in a stub area.
		if h.lsType < 1 || h.lsType > 5 {
			fmt.Printf("%v: received database description packet with invalid LSA header, discarding\n", n.neighborID)
			n.handleEvent(neSeqNumberMismatch)
			return
		}

		db := n.iface.instance.db
		existing := db.get(n.iface.areaID, h.lsType, h.lsID, h.advertisingRouter)
		if existing != nil && existing.header().age >= h.age {
			continue
		}

		n.linkStateRequestList = append(n.linkStateRequestList, h)
	}

	n.lastReceivedDD = dd
	n.ddSequenceNumber = dd.sequenceNumber
	nHeadersInLastPacket := len(n.lsaHeadersForDatabaseDescription())
	n.databaseSummaryList = n.databaseSummaryList[nHeadersInLastPacket:]

	n.sendDatabaseDescription()

	if !dd.more && len(n.databaseSummaryList) == 0 {
		n.handleEvent(neExchangeDone)
	}
}

// TODO: merge Slave and Master handlers back into this function
func (n *Neighbor) handleDatabaseDescriptionInExchange(dd *databaseDescriptionPacket) {
	if n.master == dd.master {
		fmt.Printf("%v: received database description packet with unexpected master bit, discarding\n", n.neighborID)
		n.handleEvent(neSeqNumberMismatch)
		return
	}

	if dd.init {
		fmt.Printf("%v: received database description packet with unexpected init bit, discarding\n", n.neighborID)
		n.handleEvent(neSeqNumberMismatch)
		return
	}

	if n.options != dd.options {
		fmt.Printf("%v: received database description packet with unexpected options, discarding\n", n.neighborID)
		n.handleEvent(neSeqNumberMismatch)
		return
	}

	if n.master {
		n.handleDatabaseDescriptionInExchangeMaster(dd)
	} else {
		n.handleDatabaseDescriptionInExchangeSlave(dd)
	}

	if len(n.linkStateRequestList) > 0 && n.outstandingLSReq == nil {
		n.sendLinkStateRequest()
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
		n.handleDatabaseDescriptionInExchange(dd)
	case nLoading, nFull:
		// TODO
	default:
		fmt.Printf("handleDatabaseDescription: unexpected state %v\n", n.state)
	}
}

func (n *Neighbor) handleLinkStateRequest(lsr *linkStateRequestPacket) {
	if n.state < nExchange {
		fmt.Printf("%v: neighbor state machine: unexpected link state request packet in state %v\n", n.neighborID, n.state)
		return
	}

	db := n.iface.instance.db
	lsas := make([]lsa, 0, len(lsr.reqs))

	for _, req := range lsr.reqs {
		lsa := db.get(n.iface.areaID, req.lsType, req.lsID, req.advertisingRouter)
		if lsa == nil {
			n.handleEvent(neBadLSReq)
			return
		}
		lsas = append(lsas, lsa)
	}

	lsu := newLinkStateUpdate(n.iface, lsas)

	n.send(lsu)
}

func (n *Neighbor) handleLinkStateUpdate(update *linkStateUpdatePacket) {
	// TODO
}

func (n *Neighbor) handleLinkStateAcknowledgment(lsack *linkStateAcknowledgmentPacket) {
	// TODO
}

func (n *Neighbor) flushLSAs() {
	// TODO:
	// 1. Flush the retransmission list.
	n.databaseSummaryList = nil
	n.linkStateRequestList = nil
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

func (n *Neighbor) startDDRxmtTimer() {
	fmt.Printf("%v: starting dd rxmt timer\n", n.neighborID)
	n.ddRxmtTimer.Reset(n.iface.RxmtDuration())
}

func (n *Neighbor) stopDDRxmtTimer() {
	fmt.Printf("%v: stopping dd rxmt timer\n", n.neighborID)
	if !n.ddRxmtTimer.Stop() {
		<-n.ddRxmtTimer.C
	}
}

func (n *Neighbor) resetRxmtTimer() {
	if !n.ddRxmtTimer.Stop() {
		<-n.ddRxmtTimer.C
	}
	n.ddRxmtTimer.Reset(n.iface.RxmtDuration())
}

func (n *Neighbor) handleDDRxmtTimer() {
	fmt.Printf("%v: dd rxmt timer expired\n", n.neighborID)
	switch n.state {
	case nExStart:
		n.sendDatabaseDescription()
	case nExchange:
		if !n.master {
			fmt.Printf("%v: state=Exchange unexpected handleRxmtTimer for non-master\n", n.neighborID)
			return
		}

		n.sendDatabaseDescription()
	default:
		fmt.Printf("handleRxmtTimer: unexpected state %v\n", n.state)
	}

	n.ddRxmtTimer.Reset(n.iface.RxmtDuration())
}

func (n *Neighbor) startLSReqRxmtTimer() {
	fmt.Printf("%v: starting lsreq rxmt timer\n", n.neighborID)
	n.lsReqRxmtTimer.Reset(n.iface.RxmtDuration())
}

func (n *Neighbor) stopLSReqRxmtTimer() {
	fmt.Printf("%v: stopping lsreq rxmt timer\n", n.neighborID)
	if !n.lsReqRxmtTimer.Stop() {
		<-n.lsReqRxmtTimer.C
	}
}

func (n *Neighbor) handleLSReqRxmtTimer() {
	fmt.Printf("%v: lsreq rxmt timer expired\n", n.neighborID)
	if n.state != nExchange && n.state != nLoading {
		fmt.Printf("%v: state=%v unexpected handleRxmtTimer\n", n.neighborID, n.state)
		return
	}

	if n.outstandingLSReq == nil {
		fmt.Printf("%v: unexpected handleRxmtTimer with no outstanding lsreq\n", n.neighborID)
		return
	}

	n.send(n.outstandingLSReq)
	n.lsReqRxmtTimer.Reset(n.iface.RxmtDuration())
}

func (n *Neighbor) lsaHeadersForDatabaseDescription() []lsaHeader {
	mtu := n.iface.netif.MTU
	maxHeaders := (mtu - ipv4.HeaderLen - minDDSize) / lsaHeaderSize
	nHeaders := min(maxHeaders, len(n.databaseSummaryList))

	return n.databaseSummaryList[:nHeaders]
}

func (n *Neighbor) sendDatabaseDescription() {
	if n.state == nExStart {
		n.send(newDatabaseDescription(n.iface, n.ddSequenceNumber, true, true, true, nil))
	} else if n.state == nExchange {
		headers := n.lsaHeadersForDatabaseDescription()
		nHeaders := len(headers)
		more := len(n.databaseSummaryList) > nHeaders

		n.send(newDatabaseDescription(n.iface, n.ddSequenceNumber, false, more, n.master, n.databaseSummaryList[:nHeaders]))
	} else {
		fmt.Printf("sendDatabaseDescription: unexpected state %v\n", n.state)
	}
}

func (n *Neighbor) sendLinkStateRequest() {
	mtu := n.iface.netif.MTU
	maxReqs := (mtu - ipv4.HeaderLen - minLsrSize) / reqSize
	nReqs := min(maxReqs, len(n.linkStateRequestList))

	reqs := make([]req, 0, nReqs)
	for _, h := range n.linkStateRequestList[:nReqs] {
		reqs = append(reqs, req{
			lsType:            h.lsType,
			lsID:              h.lsID,
			advertisingRouter: h.advertisingRouter,
		})
	}

	lsr := newLinkStateRequest(n.iface, reqs)
	n.outstandingLSReq = lsr

	n.startLSReqRxmtTimer()

	n.send(lsr)
}

func (n *Neighbor) send(p Packet) {
	n.iface.send(n.addr, p)
}

func (n *Neighbor) shouldBecomeAdjacent() bool {
	t := n.iface.networkType

	// TODO: brodcast and nbma networks
	return t == networkPointToPoint || t == networkPointToMultipoint || t == networkVirtualLink
}

func (n *Neighbor) dispatchEvent(event neighborEvent) {
	n.events <- event
}

func (n *Neighbor) dispatchPacket(packet Packet) {
	n.packets <- packet
}

func (n *Neighbor) run() {
	for {
		select {
		case event := <-n.events:
			n.handleEvent(event)
		case packet := <-n.packets:
			fmt.Printf("%v: received packet: %v\n", n.neighborID, packet)
			packet.handleOn(n)
		case <-n.ddRxmtTimer.C:
			n.handleDDRxmtTimer()
		case <-n.lsReqRxmtTimer.C:
			n.handleLSReqRxmtTimer()
		case <-n.done:
			return
		}
	}
}

func (n *Neighbor) shutdown() {
	close(n.done)
}
