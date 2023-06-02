package ethcall

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/protos/vega"
)

type Filter interface {
	PassesFilters(result []byte, blockHeight uint64, blockTime uint64) bool
	ToProto() *vega.EthFilter
	Hash() []byte
}

type CallResultFilter struct {
	// Reuse these existing filter definitions and logic?
	Filters types.DataSourceSpecFilters
}

func FilterFromProto(proto *vega.EthFilter) (Filter, error) {
	if proto == nil {
		return nil, fmt.Errorf("filter proto is nil")
	}

	return CallResultFilter{Filters: types.DataSourceSpecFiltersFromProto(proto.Filters)}, nil
}

func (f CallResultFilter) PassesFilters(result []byte, blockHeight uint64, blockTime uint64) bool {
	// Will need the normaliser
	return true
}

func (f CallResultFilter) ToProto() *vega.EthFilter {
	return &vega.EthFilter{Filters: f.Filters.IntoProto()}
}

func (f CallResultFilter) Hash() []byte {
	// TODO hash up the filters etc
	return []byte(" ")
}
