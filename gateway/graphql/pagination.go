package gql

import (
	"code.vegaprotocol.io/protos/commands"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
)

var defaultPagination = protoapi.Pagination{
	Skip:       0,
	Limit:      50,
	Descending: true,
}

func (p *OffsetPagination) ToProto() (protoapi.Pagination, error) {
	if p == nil {
		return defaultPagination, nil
	}

	if p.Skip < 0 {
		return protoapi.Pagination{}, commands.ErrMustBePositiveOrZero
	}

	if p.Limit < 0 {
		return protoapi.Pagination{}, commands.ErrMustBePositiveOrZero
	}

	return protoapi.Pagination{
		Skip:       uint64(p.Skip),
		Limit:      uint64(p.Limit),
		Descending: p.Descending,
	}, nil
}
