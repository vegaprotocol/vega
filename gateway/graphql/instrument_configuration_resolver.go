package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type myInstrumentConfigurationResolver VegaResolverRoot

func (r *myInstrumentConfigurationResolver) FutureProduct(ctx context.Context, obj *types.InstrumentConfiguration) (*types.FutureProduct, error) {
	return obj.GetFuture(), nil
}

// func (r *myInstrumentConfigurationResolver) Metadata(ctx context.Context, obj *types.InstrumentConfiguration) (*InstrumentMetadata, error) {
// 	return InstrumentMetadataFromProto(obj.Metadata)
// }
// func (r *myInstrumentResolver) Product(ctx context.Context, obj *proto.Instrument) (Product, error) {
// 	return obj.GetFuture(), nil
//}
