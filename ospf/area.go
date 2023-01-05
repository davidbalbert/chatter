package ospf

type Area struct {
	ID AreaID
	// TODO: List of address ranges
	Interfaces map[string]Interface
	// TODO: LSDB
	// TODO: ShortestPathTree
	// TODO: TransitCapability bool // calculated when ShortestPathTree is calculated
	// TODO: ExternalRoutingCapability bool
	// TODO: StubDefaultCost
}
