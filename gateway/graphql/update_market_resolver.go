package gql

import (
	"context"

	"code.vegaprotocol.io/protos/vega"
)

type updateMarketResolver VegaResolverRoot

func (r *updateMarketResolver) UpdateMarketConfiguration(ctx context.Context, obj *vega.UpdateMarket) (*vega.UpdateMarketConfiguration, error) {
	return obj.Changes, nil
}
