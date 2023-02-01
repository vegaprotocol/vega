// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"context"
	"errors"
	"strconv"

	types "code.vegaprotocol.io/vega/protos/vega"
)

type newMarketResolver VegaResolverRoot

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

	lmp := &LiquidityMonitoringParameters{
		TriggeringRatio:      params.TriggeringRatio,
		AuctionExtensionSecs: int(params.AuctionExtension),
	}

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

func (r *newMarketResolver) LpPriceRange(_ context.Context, obj *types.NewMarket) (string, error) {
	return obj.Changes.LpPriceRange, nil
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
