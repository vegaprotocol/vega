package gql

import (
	"context"

	"code.vegaprotocol.io/vega/proto"
)

type myInstrumentResolver VegaResolverRoot

func (r *myInstrumentResolver) Metadata(ctx context.Context, obj *proto.Instrument) (*InstrumentMetadata, error) {
	return InstrumentMetadataFromProto(obj.Metadata)
}
func (r *myInstrumentResolver) Product(ctx context.Context, obj *proto.Instrument) (Product, error) {
	return obj.GetFuture(), nil
}
