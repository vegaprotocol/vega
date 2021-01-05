package gql

import (
	"context"
	"errors"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

var (
	ErrUnsupportedProduct = errors.New("unsupported product")
)

type myInstrumentResolver VegaResolverRoot

func (r *myInstrumentResolver) Metadata(ctx context.Context, obj *types.Instrument) (*InstrumentMetadata, error) {
	return InstrumentMetadataFromProto(obj.Metadata)
}
func (r *myInstrumentResolver) Product(ctx context.Context, obj *types.Instrument) (Product, error) {
	switch obj.GetProduct().(type) {
	case *types.Instrument_Future:
		return obj.GetFuture(), nil
	default:
		return nil, ErrUnsupportedProduct
	}
}
