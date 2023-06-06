package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/vega/protos/vega"
	"golang.org/x/crypto/sha3"
)

type EthFilter struct {
	Filters DataSourceSpecFilters
}

func (f *EthFilter) String() string {
	filters := ""
	for i, filter := range f.Filters {
		if i == 0 {
			filters = filter.String()
		} else {
			filters = filters + fmt.Sprintf(", %s", filter.String())
		}
	}

	return filters
}

func (f *EthFilter) IntoProto() (*vegapb.EthFilter, error) {
	if f.Filters == nil {
		return nil, fmt.Errorf("filter proto is nil")
	}

	return &vegapb.EthFilter{
		Filters: f.Filters.IntoProto(),
	}, nil
}

func EthFilterFromProto(protoFilter *vegapb.EthFilter) (*EthFilter, error) {
	f, err := DataSourceSpecFiltersFromProto(protoFilter.Filters)
	if err != nil {
		return nil, err
	}
	return &EthFilter{
		Filters: f,
	}, nil
}

func (f *EthFilter) Hash() []byte {
	hashFunc := sha3.New256()
	ident := fmt.Sprintf("ethfilter: %s", f.Filters.String())
	hashFunc.Write([]byte(ident))
	return hashFunc.Sum(nil)
}
