package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type myFutureResolver VegaResolverRoot

func (r *myFutureResolver) Asset(ctx context.Context, obj *types.Future) (*Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

func (r *myFutureResolver) Oracle(ctx context.Context, obj *types.Future) (Oracle, error) {
	return OracleFromProto(obj.Oracle)
}
