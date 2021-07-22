package gql

import (
	"context"

	types "code.vegaprotocol.io/data-node/proto/vega"
)

type updateMarketResolver VegaResolverRoot

func (r *updateMarketResolver) MarketID(ctx context.Context, obj *types.UpdateMarket) (string, error) {
	return "not implemented", nil
}
