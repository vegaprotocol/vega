// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package gql

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
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
	return r.r.getAssetByID(ctx, obj.QuoteAsset)
}

type updateSpotMarketResolver VegaResolverRoot

func (r updateSpotMarketResolver) UpdateSpotMarketConfiguration(ctx context.Context, obj *vega.UpdateSpotMarket) (*vega.UpdateSpotMarketConfiguration, error) {
	return obj.Changes, nil
}

type updateSpotMarketConfigurationResolver VegaResolverRoot

func (r *updateSpotMarketConfigurationResolver) EnableTxReordering(ctx context.Context, obj *types.UpdateSpotMarketConfiguration) (bool, error) {
	return obj.EnableTransactionReordering, nil
}

// Instrument implements UpdateSpotMarketConfigurationResolver.
func (r *updateSpotMarketConfigurationResolver) Instrument(ctx context.Context, obj *types.UpdateSpotMarketConfiguration) (*UpdateSpotInstrumentConfiguration, error) {
	return &UpdateSpotInstrumentConfiguration{
		Name: obj.Instrument.Name,
		Code: obj.Instrument.Code,
	}, nil
}

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
		return model.Simple, nil
	case *vega.UpdateSpotMarketConfiguration_LogNormal:
		return model.LogNormal, nil
	default:
		return nil, errors.New("unknown risk model")
	}
}

func (r updateSpotMarketConfigurationResolver) LiquiditySLAParams(ctx context.Context, obj *vega.UpdateSpotMarketConfiguration) (*vega.LiquiditySLAParameters, error) {
	return obj.SlaParams, nil
}

type newSpotMarketResolver VegaResolverRoot

func (r *newSpotMarketResolver) EnableTxReordering(ctx context.Context, obj *types.NewSpotMarket) (bool, error) {
	return obj.Changes.EnableTransactionReordering, nil
}

func (r *newSpotMarketResolver) AllowedSellers(ctx context.Context, obj *types.NewSpotMarket) ([]string, error) {
	return obj.Changes.AllowedSellers, nil
}

func (r *newSpotMarketResolver) TickSize(_ context.Context, obj *types.NewSpotMarket) (string, error) {
	return obj.Changes.TickSize, nil
}

func (r newSpotMarketResolver) Instrument(ctx context.Context, obj *vega.NewSpotMarket) (*vega.InstrumentConfiguration, error) {
	return obj.Changes.Instrument, nil
}

func (r newSpotMarketResolver) PriceDecimalPlaces(ctx context.Context, obj *vega.NewSpotMarket) (int, error) {
	return int(obj.Changes.PriceDecimalPlaces), nil
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
		return model.Simple, nil
	case *vega.NewSpotMarketConfiguration_LogNormal:
		return model.LogNormal, nil
	default:
		return nil, errors.New("unknown risk model")
	}
}

func (r newSpotMarketResolver) SizeDecimalPlaces(ctx context.Context, obj *vega.NewSpotMarket) (int, error) {
	return int(obj.Changes.SizeDecimalPlaces), nil
}

func (r newSpotMarketResolver) LiquiditySLAParams(ctx context.Context, obj *vega.NewSpotMarket) (*vega.LiquiditySLAParameters, error) {
	return obj.Changes.SlaParams, nil
}

func (r newSpotMarketResolver) LiquidityFeeSettings(ctx context.Context, obj *vega.NewSpotMarket) (*vega.LiquidityFeeSettings, error) {
	return obj.Changes.LiquidityFeeSettings, nil
}
