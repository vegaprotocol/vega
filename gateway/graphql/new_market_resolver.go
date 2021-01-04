package gql

import (
	"context"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

type newMarketResolver VegaResolverRoot

func (r *newMarketResolver) Instrument(ctx context.Context, obj *types.NewMarket) (*types.InstrumentConfiguration, error) {
	return obj.Changes.Instrument, nil
}

func (r *newMarketResolver) DecimalPlaces(ctx context.Context, obj *types.NewMarket) (int, error) {
	return int(obj.Changes.DecimalPlaces), nil
}

func (r *newMarketResolver) RiskParameters(ctx context.Context, obj *types.NewMarket) (RiskModel, error) {
	return RiskConfigurationFromProto(obj.Changes)
}

func (r *newMarketResolver) Metadata(ctx context.Context, obj *types.NewMarket) ([]string, error) {
	return obj.Changes.Metadata, nil
}

func (r *newMarketResolver) TradingMode(ctx context.Context, obj *types.NewMarket) (TradingMode, error) {
	return NewMarketTradingModeFromProto(obj.Changes.TradingMode)
}
