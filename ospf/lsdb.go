package ospf

type LSAMetadata struct {
}

type LSA interface {
	LSAMetadata
}

type lsaHeader struct {
	age   uint16
	type_ lsType
}

type lsType uint8

const (
	LSTypeUnknown lsType = 0

	LSTypeRouter      lsType = 1
	LSTypeNetwork     lsType = 2
	LSTypeSummary     lsType = 3
	LSTypeASBRSummary lsType = 4
	LSTypeASExternal  lsType = 5
)

type lsaBase struct {
}

type RouterLSA struct {
	lsaBase
}

type NetworkLSA struct {
	lsaBase
}

type SummaryLSA struct {
	lsaBase
}

type ASBRSummaryLSA struct {
	lsaBase
}

type ASExternalLSA struct {
	lsaBase
}
