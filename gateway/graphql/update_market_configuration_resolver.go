package gql

import (
	"context"
	"errors"
	"strconv"

	"code.vegaprotocol.io/protos/vega"
)

type updateMarketConfigurationResolver VegaResolverRoot

func (r *updateMarketConfigurationResolver) Instrument(ctx context.Context,
	obj *vega.UpdateMarketConfiguration) (*UpdateInstrumentConfiguration, error) {
	if obj == nil {
		return nil, errors.New("no market configuration update provided")
	}
	protoInstrument := obj.Instrument

	var product *vega.UpdateFutureProduct

	switch p := protoInstrument.Product.(type) {
	case *vega.UpdateInstrumentConfiguration_Future:
		product = &vega.UpdateFutureProduct{
			QuoteName:                       p.Future.QuoteName,
			OracleSpecForSettlementPrice:    p.Future.OracleSpecForSettlementPrice,
			OracleSpecForTradingTermination: p.Future.OracleSpecForTradingTermination,
			OracleSpecBinding:               p.Future.OracleSpecBinding,
		}
	default:
		return nil, ErrUnsupportedProduct
	}

	updateInstrumentConfiguration := &UpdateInstrumentConfiguration{
		Code:    protoInstrument.Code,
		Product: product,
	}

	return updateInstrumentConfiguration, nil
}

func (r *updateMarketConfigurationResolver) PriceMonitoringParameters(ctx context.Context,
	obj *vega.UpdateMarketConfiguration) (*PriceMonitoringParameters, error) {
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
	obj *vega.UpdateMarketConfiguration) (*LiquidityMonitoringParameters, error) {
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
		TriggeringRatio: obj.LiquidityMonitoringParameters.TriggeringRatio,
	}, nil
}

func (r *updateMarketConfigurationResolver) RiskParameters(ctx context.Context,
	obj *vega.UpdateMarketConfiguration) (UpdateMarketRiskParameters, error) {
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
