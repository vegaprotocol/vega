package gql

import (
	"context"

	types "code.vegaprotocol.io/protos/vega"
)

type myFutureResolver VegaResolverRoot

func (r *myFutureResolver) SettlementAsset(ctx context.Context, obj *types.Future) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, obj.SettlementAsset)
}
