package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type myFutureResolver VegaResolverRoot

func (r *myFutureResolver) SettlementAsset(ctx context.Context, obj *types.Future) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, obj.SettlementAsset)
}

func (r *myFutureResolver) Oracle(ctx context.Context, obj *types.Future) (Oracle, error) {
	return OracleFromProto(obj.Oracle)
}
