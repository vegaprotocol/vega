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
	"strconv"

	"code.vegaprotocol.io/vega/libs/ptr"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type newMarketResolver VegaResolverRoot

func (r *newMarketResolver) EnableTxReordering(ctx context.Context, obj *types.NewMarket) (bool, error) {
	return obj.Changes.EnableTransactionReordering, nil
}

func (r *newMarketResolver) TickSize(_ context.Context, obj *types.NewMarket) (string, error) {
	return obj.Changes.TickSize, nil
}

func (r *newMarketResolver) Instrument(_ context.Context, obj *types.NewMarket) (*types.InstrumentConfiguration, error) {
	return obj.Changes.Instrument, nil
}

func (r *newMarketResolver) DecimalPlaces(_ context.Context, obj *types.NewMarket) (int, error) {
	return int(obj.Changes.DecimalPlaces), nil
}

func (r *newMarketResolver) PriceMonitoringParameters(_ context.Context, obj *types.NewMarket) (*PriceMonitoringParameters, error) {
	triggers := make([]*PriceMonitoringTrigger, len(obj.Changes.PriceMonitoringParameters.Triggers))
	for i, t := range obj.Changes.PriceMonitoringParameters.Triggers {
		probability, err := strconv.ParseFloat(t.Probability, 64)
		if err != nil {
			return nil, err
		}
		triggers[i] = &PriceMonitoringTrigger{
			HorizonSecs:          int(t.Horizon),
			Probability:          probability,
			AuctionExtensionSecs: int(t.AuctionExtension),
		}
	}
	return &PriceMonitoringParameters{Triggers: triggers}, nil
}

func (r *newMarketResolver) LiquidityMonitoringParameters(_ context.Context, obj *types.NewMarket) (*LiquidityMonitoringParameters, error) {
	params := obj.Changes.LiquidityMonitoringParameters
	if params == nil {
		return nil, nil
	}

	lmp := &LiquidityMonitoringParameters{}

	if params.TargetStakeParameters != nil {
		lmp.TargetStakeParameters = &TargetStakeParameters{
			TimeWindow:    int(params.TargetStakeParameters.TimeWindow),
			ScalingFactor: params.TargetStakeParameters.ScalingFactor,
		}
	}
	return lmp, nil
}

func (r *newMarketResolver) PositionDecimalPlaces(_ context.Context, obj *types.NewMarket) (int, error) {
	return int(obj.Changes.PositionDecimalPlaces), nil
}

func (r *newMarketResolver) LinearSlippageFactor(_ context.Context, obj *types.NewMarket) (string, error) {
	return obj.Changes.LinearSlippageFactor, nil
}

func (r *newMarketResolver) QuadraticSlippageFactor(_ context.Context, obj *types.NewMarket) (string, error) {
	return obj.Changes.QuadraticSlippageFactor, nil
}

func (r *newMarketResolver) RiskParameters(_ context.Context, obj *types.NewMarket) (RiskModel, error) {
	switch rm := obj.Changes.RiskParameters.(type) {
	case *types.NewMarketConfiguration_LogNormal:
		return rm.LogNormal, nil
	case *types.NewMarketConfiguration_Simple:
		return rm.Simple, nil
	default:
		return nil, errors.New("invalid risk model")
	}
}

func (r *newMarketResolver) Metadata(_ context.Context, obj *types.NewMarket) ([]string, error) {
	return obj.Changes.Metadata, nil
}

func (r *newMarketResolver) SuccessorConfiguration(ctx context.Context, obj *types.NewMarket) (*types.SuccessorConfiguration, error) {
	return obj.Changes.Successor, nil
}

func (r *newMarketResolver) LiquiditySLAParameters(ctx context.Context, obj *types.NewMarket) (*types.LiquiditySLAParameters, error) {
	return obj.Changes.LiquiditySlaParameters, nil
}

func (r *newMarketResolver) LiquidityFeeSettings(ctx context.Context, obj *types.NewMarket) (*types.LiquidityFeeSettings, error) {
	return obj.Changes.LiquidityFeeSettings, nil
}

func (r *newMarketResolver) LiquidationStrategy(ctx context.Context, obj *types.NewMarket) (*types.LiquidationStrategy, error) {
	return obj.Changes.LiquidationStrategy, nil
}

func (r *newMarketResolver) MarkPriceConfiguration(ctx context.Context, obj *types.NewMarket) (*types.CompositePriceConfiguration, error) {
	return obj.Changes.MarkPriceConfiguration, nil
}

func (r *newMarketResolver) AllowedEmptyAMMLevels(ctx context.Context, obj *types.NewMarket) (*int, error) {
	v := obj.Changes.AllowedEmptyAmmLevels
	if v == nil {
		return nil, nil
	}
	return ptr.From(int(*obj.Changes.AllowedEmptyAmmLevels)), nil
}
