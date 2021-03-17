package gql

import (
	"context"
	"errors"

	types "code.vegaprotocol.io/vega/proto"
)

type newMarketResolver VegaResolverRoot

func (r *newMarketResolver) Instrument(ctx context.Context, obj *types.NewMarket) (*types.InstrumentConfiguration, error) {
	return obj.Changes.Instrument, nil
}

func (r *newMarketResolver) DecimalPlaces(ctx context.Context, obj *types.NewMarket) (int, error) {
	return int(obj.Changes.DecimalPlaces), nil
}

func (r *newMarketResolver) RiskParameters(ctx context.Context, obj *types.NewMarket) (RiskModel, error) {
	switch rm := obj.Changes.RiskParameters.(type) {
	case *types.NewMarketConfiguration_LogNormal:
		return rm.LogNormal, nil
	case *types.NewMarketConfiguration_Simple:
		return rm.Simple, nil
	default:
		return nil, errors.New("invalid risk model")
	}
}

func (r *newMarketResolver) Metadata(ctx context.Context, obj *types.NewMarket) ([]string, error) {
	return obj.Changes.Metadata, nil
}

func (r *newMarketResolver) TradingMode(ctx context.Context, obj *types.NewMarket) (TradingMode, error) {
	return NewMarketTradingModeFromProto(obj.Changes.TradingMode)
}

func (r *newMarketResolver) Commitment(ctx context.Context, obj *types.NewMarket) (*types.NewMarketCommitment, error) {
	return obj.LiquidityCommitment, nil
}
