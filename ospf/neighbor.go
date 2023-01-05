package ospf

import "net/netip"

type Neighbor struct {
	state neighborState
	// TODO: InactivityTimer - single shot timer
	Master                 bool
	DDSequenceNumber       uint32
	LastReceivedDD         *DD
	ID                     RouterID
	Priority               uint8
	Addr                   netip.Addr
	Options                uint8
	DesignatedRouter       RouterID
	BackupDesignatedRouter RouterID

	// TODO: make a type for these lists?
	RetransmissionList   []*lsaHeader
	DatabaseSummaryList  []*lsaHeader
	LinkStateRequestList []*lsaHeader
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

func (ns neighborState) String() string {
	switch ns {
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

func (ne neighborEvent) String() string {
	switch ne {
	case neHelloReceived:
		return "HelloReceived"
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
