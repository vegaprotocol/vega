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
	"code.vegaprotocol.io/vega/protos/vega"
)

type updateMarketConfigurationResolver VegaResolverRoot

func (r *updateMarketConfigurationResolver) EnableTxReordering(ctx context.Context, obj *vega.UpdateMarketConfiguration) (bool, error) {
	return obj.EnableTransactionReordering, nil
}

func (r *updateMarketConfigurationResolver) Instrument(ctx context.Context,
	obj *vega.UpdateMarketConfiguration,
) (*UpdateInstrumentConfiguration, error) {
	if obj == nil {
		return nil, errors.New("no market configuration update provided")
	}
	protoInstrument := obj.Instrument

	var product UpdateProductConfiguration

	switch p := protoInstrument.Product.(type) {
	case *vega.UpdateInstrumentConfiguration_Future:
		product = &vega.UpdateFutureProduct{
			QuoteName:                           p.Future.QuoteName,
			DataSourceSpecForSettlementData:     p.Future.DataSourceSpecForSettlementData,
			DataSourceSpecForTradingTermination: p.Future.DataSourceSpecForTradingTermination,
			DataSourceSpecBinding:               p.Future.DataSourceSpecBinding,
		}
	case *vega.UpdateInstrumentConfiguration_Perpetual:
		product = &vega.UpdatePerpetualProduct{
			QuoteName:                           p.Perpetual.QuoteName,
			MarginFundingFactor:                 p.Perpetual.MarginFundingFactor,
			InterestRate:                        p.Perpetual.InterestRate,
			ClampLowerBound:                     p.Perpetual.ClampLowerBound,
			ClampUpperBound:                     p.Perpetual.ClampUpperBound,
			FundingRateScalingFactor:            p.Perpetual.FundingRateScalingFactor,
			FundingRateLowerBound:               p.Perpetual.FundingRateLowerBound,
			FundingRateUpperBound:               p.Perpetual.FundingRateUpperBound,
			DataSourceSpecForSettlementSchedule: p.Perpetual.DataSourceSpecForSettlementSchedule,
			DataSourceSpecForSettlementData:     p.Perpetual.DataSourceSpecForSettlementData,
			DataSourceSpecBinding:               p.Perpetual.DataSourceSpecBinding,
		}
	default:
		return nil, ErrUnsupportedProduct
	}

	updateInstrumentConfiguration := &UpdateInstrumentConfiguration{
		Code:    protoInstrument.Code,
		Name:    protoInstrument.Name,
		Product: product,
	}

	return updateInstrumentConfiguration, nil
}

func (r *updateMarketConfigurationResolver) PriceMonitoringParameters(ctx context.Context,
	obj *vega.UpdateMarketConfiguration,
) (*PriceMonitoringParameters, error) {
	if obj == nil {
		return nil, errors.New("no market configuration update provided")
	}

	if obj.PriceMonitoringParameters == nil {
		return nil, nil
	}

	triggers := make([]*PriceMonitoringTrigger, 0, len(obj.PriceMonitoringParameters.Triggers))

	for _, trigger := range obj.PriceMonitoringParameters.Triggers {
		probability, err := strconv.ParseFloat(trigger.Probability, 64)
		if err != nil {
			continue
		}
		triggers = append(triggers, &PriceMonitoringTrigger{
			HorizonSecs:          int(trigger.Horizon),
			Probability:          probability,
			AuctionExtensionSecs: int(trigger.AuctionExtension),
		})
	}

	params := &PriceMonitoringParameters{
		Triggers: triggers,
	}

	return params, nil
}

func (r *updateMarketConfigurationResolver) LiquidityMonitoringParameters(ctx context.Context,
	obj *vega.UpdateMarketConfiguration,
) (*LiquidityMonitoringParameters, error) {
	if obj == nil {
		return nil, errors.New("no market configuration update provided")
	}

	if obj.LiquidityMonitoringParameters == nil {
		return nil, nil
	}

	return &LiquidityMonitoringParameters{
		TargetStakeParameters: &TargetStakeParameters{
			TimeWindow:    int(obj.LiquidityMonitoringParameters.TargetStakeParameters.TimeWindow),
			ScalingFactor: obj.LiquidityMonitoringParameters.TargetStakeParameters.ScalingFactor,
		},
	}, nil
}

func (r *updateMarketConfigurationResolver) RiskParameters(ctx context.Context,
	obj *vega.UpdateMarketConfiguration,
) (UpdateMarketRiskParameters, error) {
	if obj == nil {
		return nil, errors.New("no market configuration update provided")
	}

	if obj.RiskParameters == nil {
		return nil, errors.New("no risk configuration provided")
	}

	var params UpdateMarketRiskParameters

	switch rp := obj.RiskParameters.(type) {
	case *vega.UpdateMarketConfiguration_Simple:
		params = rp
	case *vega.UpdateMarketConfiguration_LogNormal:
		params = rp
	default:
		return nil, errors.New("invalid risk configuration provided")
	}

	return params, nil
}

func (r *updateMarketConfigurationResolver) AllowedEmptyAMMLevels(ctx context.Context, obj *vega.UpdateMarketConfiguration) (*int, error) {
	v := obj.AllowedEmptyAmmLevels
	if v == nil {
		return nil, nil
	}
	return ptr.From(int(*obj.AllowedEmptyAmmLevels)), nil
}
