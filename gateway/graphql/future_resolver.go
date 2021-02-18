package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto"
)

type myFutureResolver VegaResolverRoot

func (r *myFutureResolver) SettlementAsset(ctx context.Context, obj *types.Future) (*Asset, error) {
	return r.r.getAssetByID(ctx, obj.SettlementAsset)
}
