package gql

import (
	"context"

	vega "code.vegaprotocol.io/vega/protos/vega"
)

type updateMarketStateResolver VegaResolverRoot

func (r *updateMarketStateResolver) Market(ctx context.Context, obj *vega.UpdateMarketState) (*vega.Market, error) {
	return r.r.getMarketByID(ctx, obj.Changes.MarketId)
}

func (r *updateMarketStateResolver) UpdateType(ctx context.Context, obj *vega.UpdateMarketState) (MarketUpdateType, error) {
	switch obj.Changes.UpdateType {
	case vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_TERMINATE:
		return MarketUpdateTypeMarketStateUpdateTypeTerminate, nil
	case vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_SUSPEND:
		return MarketUpdateTypeMarketStateUpdateTypeSuspend, nil
	case vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_RESUME:
		return MarketUpdateTypeMarketStateUpdateTypeResume, nil
	default:
		return MarketUpdateTypeMarketStateUpdateTypeUnspecified, nil
	}
}

func (urpd *updateMarketStateResolver) Price(ctx context.Context, obj *vega.UpdateMarketState) (*string, error) {
	return obj.Changes.Price, nil
}
