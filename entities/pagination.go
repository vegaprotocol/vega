package entities

import v2 "code.vegaprotocol.io/protos/data-node/api/v2"

type Pagination struct {
	Skip       uint64
	Limit      uint64
	Descending bool
}

func PaginationFromProto(pp *v2.Pagination) Pagination {
	return Pagination{
		Skip:       pp.Skip,
		Limit:      pp.Limit,
		Descending: pp.Descending,
	}
}
