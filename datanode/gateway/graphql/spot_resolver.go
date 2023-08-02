package gql

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/protos/vega"
)

type spotResolver VegaResolverRoot

func (r *spotResolver) BaseAsset(ctx context.Context, obj *vega.Spot) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.BaseAsset)
}

func (r *spotResolver) QuoteAsset(ctx context.Context, obj *vega.Spot) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.QuoteAsset)
}

type spotProductResolver VegaResolverRoot

func (r spotProductResolver) BaseAsset(ctx context.Context, obj *vega.SpotProduct) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.BaseAsset)
}

func (r spotProductResolver) QuoteAsset(ctx context.Context, obj *vega.SpotProduct) (*vega.Asset, error) {
	return r.r.getAssetByID(ctx, obj.BaseAsset)
}

type updateSpotMarketResolver VegaResolverRoot

func (r updateSpotMarketResolver) UpdateSpotMarketConfiguration(ctx context.Context, obj *vega.UpdateSpotMarket) (*vega.UpdateSpotMarketConfiguration, error) {
	return obj.Changes, nil
}

type updateSpotMarketConfigurationResolver VegaResolverRoot

func (r updateSpotMarketConfigurationResolver) PriceMonitoringParameters(ctx context.Context, obj *vega.UpdateSpotMarketConfiguration) (*PriceMonitoringParameters, error) {
	return PriceMonitoringParametersFromProto(obj.PriceMonitoringParameters)
}

func (r updateSpotMarketConfigurationResolver) TargetStakeParameters(ctx context.Context, obj *vega.UpdateSpotMarketConfiguration) (*TargetStakeParameters, error) {
	return &TargetStakeParameters{
		TimeWindow:    int(obj.TargetStakeParameters.TimeWindow),
		ScalingFactor: obj.TargetStakeParameters.ScalingFactor,
	}, nil
}

func (r updateSpotMarketConfigurationResolver) RiskParameters(ctx context.Context, obj *vega.UpdateSpotMarketConfiguration) (RiskModel, error) {
	switch model := obj.RiskParameters.(type) {
	case *vega.UpdateSpotMarketConfiguration_Simple:
		return model, nil
	case *vega.UpdateSpotMarketConfiguration_LogNormal:
		return model, nil
	default:
		return nil, errors.New("unknown risk model")
	}
}

func (r updateSpotMarketConfigurationResolver) LiquiditySLAParams(ctx context.Context, obj *vega.UpdateSpotMarketConfiguration) (*vega.LiquiditySLAParameters, error) {
	return obj.SlaParams, nil
}

type newSpotMarketResolver VegaResolverRoot

func (r newSpotMarketResolver) Instrument(ctx context.Context, obj *vega.NewSpotMarket) (*vega.InstrumentConfiguration, error) {
	return obj.Changes.Instrument, nil
}

func (r newSpotMarketResolver) DecimalPlaces(ctx context.Context, obj *vega.NewSpotMarket) (int, error) {
	return int(obj.Changes.DecimalPlaces), nil
}

func (r newSpotMarketResolver) Metadata(ctx context.Context, obj *vega.NewSpotMarket) ([]string, error) {
	return obj.Changes.Metadata, nil
}

func (r newSpotMarketResolver) PriceMonitoringParameters(ctx context.Context, obj *vega.NewSpotMarket) (*PriceMonitoringParameters, error) {
	return PriceMonitoringParametersFromProto(obj.Changes.PriceMonitoringParameters)
}

func (r newSpotMarketResolver) TargetStakeParameters(ctx context.Context, obj *vega.NewSpotMarket) (*TargetStakeParameters, error) {
	return &TargetStakeParameters{
		TimeWindow:    int(obj.Changes.TargetStakeParameters.TimeWindow),
		ScalingFactor: obj.Changes.TargetStakeParameters.ScalingFactor,
	}, nil
}

func (r newSpotMarketResolver) RiskParameters(ctx context.Context, obj *vega.NewSpotMarket) (RiskModel, error) {
	switch model := obj.Changes.RiskParameters.(type) {
	case *vega.NewSpotMarketConfiguration_Simple:
		return model, nil
	case *vega.NewSpotMarketConfiguration_LogNormal:
		return model, nil
	default:
		return nil, errors.New("unknown risk model")
	}
}

func (r newSpotMarketResolver) PositionDecimalPlaces(ctx context.Context, obj *vega.NewSpotMarket) (int, error) {
	return int(obj.Changes.PositionDecimalPlaces), nil
}

func (r newSpotMarketResolver) LiquiditySLAParams(ctx context.Context, obj *vega.NewSpotMarket) (*vega.LiquiditySLAParameters, error) {
	return obj.Changes.SlaParams, nil
}
