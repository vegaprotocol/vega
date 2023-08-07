// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package governance

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	proto "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

var (
	// ErrMissingProduct is returned if selected product is nil.
	ErrMissingProduct = errors.New("missing product")
	// ErrUnsupportedProduct is returned if selected product is not supported.
	ErrUnsupportedProduct = errors.New("product type is not supported")
	// ErrUnsupportedRiskParameters is returned if risk parameters supplied via governance are not yet supported.
	ErrUnsupportedRiskParameters = errors.New("risk model parameters are not supported")
	// ErrMissingRiskParameters ...
	ErrMissingRiskParameters = errors.New("missing risk parameters")
	// ErrMissingDataSourceSpecBinding is returned when the data source spec binding is absent.
	ErrMissingDataSourceSpecBinding = errors.New("missing data source spec binding")
	// ErrMissingDataSourceSpecForSettlementData is returned when the data source spec for settlement data is absent.
	ErrMissingDataSourceSpecForSettlementData = errors.New("missing data source spec for settlement data")
	// ErrMissingDataSourceSpecForSettlementData is returned when the data source spec for settlement data is absent.
	ErrSettlementWithInternalDataSourceIsNotAllowed = errors.New("settlement with internal data source is not allwed")
	// ErrMissingDataSourceSpecForTradingTermination is returned when the data source spec for trading termination is absent.
	ErrMissingDataSourceSpecForTradingTermination = errors.New("missing data source spec for trading termination")
	// ErrMissingDataSourceSpecForSettlementSchedule is returned when the data source spec for trading termination is absent.
	ErrMissingDataSourceSpecForSettlementSchedule = errors.New("missing data source spec for settlement schedule")
	// ErrInternalTimeTriggerForFuturesInNotAllowed is returned when a proposal containing timetrigger terminaiton type of data is received.
	ErrInternalTimeTriggerForFuturesInNotAllowed = errors.New("setting internal time trigger for future termination is not allowed")
	// ErrDataSourceSpecTerminationTimeBeforeEnactment is returned when termination time is before enactment
	// for time triggered termination condition.
	ErrDataSourceSpecTerminationTimeBeforeEnactment = errors.New("data source spec termination time before enactment")
	// ErrMissingPerpsProduct is returned when perps product is absent from the instrument.
	ErrMissingPerpsProduct = errors.New("missing perps product")
	// ErrMissingFutureProduct is returned when future product is absent from the instrument.
	ErrMissingFutureProduct = errors.New("missing future product")
	// ErrMissingSpotProduct is returned when spot product is absent from the instrument.
	ErrMissingSpotProduct = errors.New("missing spot product")
	// ErrInvalidRiskParameter ...
	ErrInvalidRiskParameter = errors.New("invalid risk parameter")
	// ErrInvalidInsurancePoolFraction is returned if the insurance pool fraction parameter is outside of the 0-1 range.
	ErrInvalidInsurancePoolFraction = errors.New("insurnace pool fraction invalid")
)

func assignProduct(
	source *types.InstrumentConfiguration,
	target *types.Instrument,
) (proto.ProposalError, error) {
	switch product := source.Product.(type) {
	case *types.InstrumentConfigurationFuture:
		if product.Future == nil {
			return types.ProposalErrorInvalidFutureProduct, ErrMissingFutureProduct
		}
		settlData := &product.Future.DataSourceSpecForSettlementData
		if settlData == nil {
			return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForSettlementData
		}

		tterm := &product.Future.DataSourceSpecForTradingTermination
		if tterm == nil {
			return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForTradingTermination
		}
		if product.Future.DataSourceSpecBinding == nil {
			return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecBinding
		}

		target.Product = &types.InstrumentFuture{
			Future: &types.Future{
				SettlementAsset:                     product.Future.SettlementAsset,
				QuoteName:                           product.Future.QuoteName,
				DataSourceSpecForSettlementData:     datasource.SpecFromDefinition(product.Future.DataSourceSpecForSettlementData),
				DataSourceSpecForTradingTermination: datasource.SpecFromDefinition(product.Future.DataSourceSpecForTradingTermination),
				DataSourceSpecBinding:               product.Future.DataSourceSpecBinding,
			},
		}
	case *types.InstrumentConfigurationPerps:
		if product.Perps == nil {
			return types.ProposalErrorInvalidPerpsProduct, ErrMissingPerpsProduct
		}
		settlData := &product.Perps.DataSourceSpecForSettlementData
		if settlData == nil {
			return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementData
		}

		settlSchedule := &product.Perps.DataSourceSpecForSettlementSchedule
		if settlSchedule == nil {
			return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForTradingTermination
		}
		if product.Perps.DataSourceSpecBinding == nil {
			return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecBinding
		}

		target.Product = &types.InstrumentPerps{
			Perps: &types.Perps{
				SettlementAsset:                     product.Perps.SettlementAsset,
				QuoteName:                           product.Perps.QuoteName,
				InterestRate:                        product.Perps.InterestRate,
				MarginFundingFactor:                 product.Perps.MarginFundingFactor,
				ClampLowerBound:                     product.Perps.ClampLowerBound,
				ClampUpperBound:                     product.Perps.ClampUpperBound,
				DataSourceSpecForSettlementData:     datasource.SpecFromDefinition(product.Perps.DataSourceSpecForSettlementData),
				DataSourceSpecForSettlementSchedule: datasource.SpecFromDefinition(product.Perps.DataSourceSpecForSettlementSchedule),
				DataSourceSpecBinding:               product.Perps.DataSourceSpecBinding,
			},
		}
	case *types.InstrumentConfigurationSpot:
		if product.Spot == nil {
			return types.ProposalErrorInvalidSpot, ErrMissingSpotProduct
		}

		target.Product = &types.InstrumentSpot{
			Spot: &types.Spot{
				Name:       product.Spot.Name,
				BaseAsset:  product.Spot.BaseAsset,
				QuoteAsset: product.Spot.QuoteAsset,
			},
		}

	default:
		return types.ProposalErrorUnsupportedProduct, ErrUnsupportedProduct
	}
	return types.ProposalErrorUnspecified, nil
}

func createInstrument(
	input *types.InstrumentConfiguration,
	tags []string,
) (*types.Instrument, types.ProposalError, error) {
	result := &types.Instrument{
		Name: input.Name,
		Code: input.Code,
		Metadata: &types.InstrumentMetadata{
			Tags: tags,
		},
	}

	if perr, err := assignProduct(input, result); err != nil {
		return nil, perr, err
	}
	return result, types.ProposalErrorUnspecified, nil
}

func assignRiskModel(definition *types.NewMarketConfiguration, target *types.TradableInstrument) error {
	switch parameters := definition.RiskParameters.(type) {
	case *types.NewMarketConfigurationSimple:
		target.RiskModel = &types.TradableInstrumentSimpleRiskModel{
			SimpleRiskModel: &types.SimpleRiskModel{
				Params: parameters.Simple,
			},
		}
	case *types.NewMarketConfigurationLogNormal:
		target.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
			LogNormalRiskModel: parameters.LogNormal,
		}
	default:
		return ErrUnsupportedRiskParameters
	}
	return nil
}

func assignSpotRiskModel(definition *types.NewSpotMarketConfiguration, target *types.TradableInstrument) error {
	switch parameters := definition.RiskParameters.(type) {
	case *types.NewSpotMarketConfigurationSimple:
		target.RiskModel = &types.TradableInstrumentSimpleRiskModel{
			SimpleRiskModel: &types.SimpleRiskModel{
				Params: parameters.Simple,
			},
		}
	case *types.NewSpotMarketConfigurationLogNormal:
		target.RiskModel = &types.TradableInstrumentLogNormalRiskModel{
			LogNormalRiskModel: parameters.LogNormal,
		}
	default:
		return ErrUnsupportedRiskParameters
	}
	return nil
}

func buildMarketFromProposal(
	marketID string,
	definition *types.NewMarket,
	netp NetParams,
	openingAuctionDuration time.Duration,
) (*types.Market, types.ProposalError, error) {
	instrument, perr, err := createInstrument(definition.Changes.Instrument, definition.Changes.Metadata)
	if err != nil {
		return nil, perr, err
	}

	// get factors for the market
	makerFee, _ := netp.Get(netparams.MarketFeeFactorsMakerFee)
	infraFee, _ := netp.Get(netparams.MarketFeeFactorsInfrastructureFee)
	// get the margin scaling factors
	scalingFactors := proto.ScalingFactors{}
	_ = netp.GetJSONStruct(netparams.MarketMarginScalingFactors, &scalingFactors)
	// get price monitoring parameters
	if definition.Changes.PriceMonitoringParameters == nil {
		pmParams := &proto.PriceMonitoringParameters{}
		_ = netp.GetJSONStruct(netparams.MarketPriceMonitoringDefaultParameters, pmParams)
		definition.Changes.PriceMonitoringParameters = types.PriceMonitoringParametersFromProto(pmParams)
	}

	if definition.Changes.LiquidityMonitoringParameters == nil ||
		definition.Changes.LiquidityMonitoringParameters.TargetStakeParameters == nil {
		// get target stake parameters
		tsTimeWindow, _ := netp.GetDuration(netparams.MarketTargetStakeTimeWindow)
		tsScalingFactor, _ := netp.GetDecimal(netparams.MarketTargetStakeScalingFactor)
		// get triggering ratio
		triggeringRatio, _ := netp.GetDecimal(netparams.MarketLiquidityTargetStakeTriggeringRatio)

		params := &types.TargetStakeParameters{
			TimeWindow:    int64(tsTimeWindow.Seconds()),
			ScalingFactor: tsScalingFactor,
		}

		if definition.Changes.LiquidityMonitoringParameters == nil {
			definition.Changes.LiquidityMonitoringParameters = &types.LiquidityMonitoringParameters{
				TargetStakeParameters: params,
				TriggeringRatio:       triggeringRatio,
			}
		} else {
			definition.Changes.LiquidityMonitoringParameters.TargetStakeParameters = params
		}
	}

	makerFeeDec, _ := num.DecimalFromString(makerFee)
	infraFeeDec, _ := num.DecimalFromString(infraFee)
	market := &types.Market{
		ID:                    marketID,
		DecimalPlaces:         definition.Changes.DecimalPlaces,
		PositionDecimalPlaces: definition.Changes.PositionDecimalPlaces,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          makerFeeDec,
				InfrastructureFee: infraFeeDec,
			},
		},
		OpeningAuction: &types.AuctionDuration{
			Duration: int64(openingAuctionDuration.Seconds()),
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: instrument,
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: types.ScalingFactorsFromProto(&scalingFactors),
			},
		},
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters: definition.Changes.PriceMonitoringParameters,
		},
		LiquidityMonitoringParameters: definition.Changes.LiquidityMonitoringParameters,
		LPPriceRange:                  definition.Changes.LpPriceRange,
		LinearSlippageFactor:          definition.Changes.LinearSlippageFactor,
		QuadraticSlippageFactor:       definition.Changes.QuadraticSlippageFactor,
	}
	// successor proposal
	if suc := definition.Successor(); suc != nil {
		market.ParentMarketID = suc.ParentID
		market.InsurancePoolFraction = suc.InsurancePoolFraction
	}
	if err := assignRiskModel(definition.Changes, market.TradableInstrument); err != nil {
		return nil, types.ProposalErrorUnspecified, err
	}
	return market, types.ProposalErrorUnspecified, nil
}

func buildSpotMarketFromProposal(
	marketID string,
	definition *types.NewSpotMarket,
	netp NetParams,
	openingAuctionDuration time.Duration,
) (*types.Market, types.ProposalError, error) {
	instrument, perr, err := createInstrument(definition.Changes.Instrument, definition.Changes.Metadata)
	if err != nil {
		return nil, perr, err
	}

	// get factors for the market
	makerFee, _ := netp.Get(netparams.MarketFeeFactorsMakerFee)
	infraFee, _ := netp.Get(netparams.MarketFeeFactorsInfrastructureFee)
	// get price monitoring parameters
	if definition.Changes.PriceMonitoringParameters == nil {
		pmParams := &proto.PriceMonitoringParameters{}
		_ = netp.GetJSONStruct(netparams.MarketPriceMonitoringDefaultParameters, pmParams)
		definition.Changes.PriceMonitoringParameters = types.PriceMonitoringParametersFromProto(pmParams)
	}

	if definition.Changes.TargetStakeParameters == nil {
		// get target stake parameters
		tsTimeWindow, _ := netp.GetDuration(netparams.MarketTargetStakeTimeWindow)
		tsScalingFactor, _ := netp.GetDecimal(netparams.MarketTargetStakeScalingFactor)
		params := &types.TargetStakeParameters{
			TimeWindow:    int64(tsTimeWindow.Seconds()),
			ScalingFactor: tsScalingFactor,
		}
		definition.Changes.TargetStakeParameters = params
	}

	liquidityMonitoring := &types.LiquidityMonitoringParameters{
		TargetStakeParameters: definition.Changes.TargetStakeParameters,
		TriggeringRatio:       num.DecimalZero(),
		AuctionExtension:      0,
	}

	makerFeeDec, _ := num.DecimalFromString(makerFee)
	infraFeeDec, _ := num.DecimalFromString(infraFee)
	market := &types.Market{
		ID:                    marketID,
		DecimalPlaces:         definition.Changes.DecimalPlaces,
		PositionDecimalPlaces: definition.Changes.PositionDecimalPlaces,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          makerFeeDec,
				InfrastructureFee: infraFeeDec,
			},
		},
		OpeningAuction: &types.AuctionDuration{
			Duration: int64(openingAuctionDuration.Seconds()),
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: instrument,
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       num.DecimalZero(),
					InitialMargin:     num.DecimalZero(),
					CollateralRelease: num.DecimalZero(),
				},
			},
		},
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters: definition.Changes.PriceMonitoringParameters,
		},
		LiquidityMonitoringParameters: liquidityMonitoring,
		LinearSlippageFactor:          num.DecimalZero(),
		QuadraticSlippageFactor:       num.DecimalZero(),
		LiquiditySLAParams:            definition.Changes.SLAParams,
	}
	if err := assignSpotRiskModel(definition.Changes, market.TradableInstrument); err != nil {
		return nil, types.ProposalErrorUnspecified, err
	}
	return market, types.ProposalErrorUnspecified, nil
}

func validateAssetBasic(assetID string, assets Assets, deepCheck bool) (types.ProposalError, error) {
	if len(assetID) <= 0 {
		return types.ProposalErrorInvalidAsset, errors.New("missing asset ID")
	}

	if !deepCheck {
		return types.ProposalErrorUnspecified, nil
	}

	_, err := assets.Get(assetID)
	if err != nil {
		return types.ProposalErrorInvalidAsset, err
	}
	if !assets.IsEnabled(assetID) {
		return types.ProposalErrorInvalidAsset,
			fmt.Errorf("asset is not enabled %v", assetID)
	}
	return types.ProposalErrorUnspecified, nil
}

func validateAsset(assetID string, decimals uint64, assets Assets, deepCheck bool) (types.ProposalError, error) {
	if len(assetID) <= 0 {
		return types.ProposalErrorInvalidAsset, errors.New("missing asset ID")
	}

	if !deepCheck {
		return types.ProposalErrorUnspecified, nil
	}

	asset, err := assets.Get(assetID)
	if err != nil {
		return types.ProposalErrorInvalidAsset, err
	}
	if !assets.IsEnabled(assetID) {
		return types.ProposalErrorInvalidAsset,
			fmt.Errorf("assets is not enabled %v", assetID)
	}
	// decimal places asset less than market -> invalid.
	// @TODO add a specific error for this validation?
	if asset.DecimalPlaces() < decimals {
		return types.ProposalErrorTooManyMarketDecimalPlaces, errors.New("market cannot have more decimal places than assets")
	}

	return types.ProposalErrorUnspecified, nil
}

func validateSpot(spot *types.SpotProduct, decimals uint64, assets Assets, deepCheck bool) (types.ProposalError, error) {
	propError, err := validateAsset(spot.QuoteAsset, decimals, assets, deepCheck)
	if err != nil {
		return propError, err
	}
	return validateAssetBasic(spot.BaseAsset, assets, deepCheck)
}

func validateFuture(future *types.FutureProduct, decimals uint64, assets Assets, et *enactmentTime, deepCheck bool) (types.ProposalError, error) {
	settlData := &future.DataSourceSpecForSettlementData
	if settlData == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForSettlementData
	}

	if settlData.Content() == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForSettlementData
	}

	ext, err := settlData.IsExternal()
	if err != nil {
		return types.ProposalErrorInvalidFutureProduct, err
	}

	if !ext {
		return types.ProposalErrorInvalidFutureProduct, ErrSettlementWithInternalDataSourceIsNotAllowed
	}

	tterm := &future.DataSourceSpecForTradingTermination
	if tterm == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForTradingTermination
	}

	if tterm.Content() == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForTradingTermination
	}

	tp, _ := tterm.Type()
	if tp == datasource.ContentTypeInternalTimeTriggerTermination {
		return types.ProposalErrorInvalidFutureProduct, ErrInternalTimeTriggerForFuturesInNotAllowed
	}

	filters := future.DataSourceSpecForTradingTermination.GetFilters()
	for i, f := range filters {
		if f.Key.Type == datapb.PropertyKey_TYPE_TIMESTAMP {
			for j, cond := range f.Conditions {
				v, err := strconv.ParseInt(cond.Value, 10, 64)
				if err != nil {
					return types.ProposalErrorInvalidFutureProduct, err
				}

				filters[i].Conditions[j].Value = strconv.FormatInt(v, 10)
				if !et.shouldNotVerify {
					if v <= et.current {
						return types.ProposalErrorInvalidFutureProduct, ErrDataSourceSpecTerminationTimeBeforeEnactment
					}
				}
			}
		}
	}
	future.DataSourceSpecForTradingTermination.UpdateFilters(filters)

	if future.DataSourceSpecBinding == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecBinding
	}

	// ensure the oracle spec for settlement data can be constructed
	ospec, err := spec.New(*datasource.SpecFromDefinition(future.DataSourceSpecForSettlementData))
	if err != nil {
		return types.ProposalErrorInvalidFutureProduct, err
	}
	switch future.DataSourceSpecBinding.SettlementDataProperty {
	case datapb.PropertyKey_TYPE_DECIMAL.String():
		err := ospec.EnsureBoundableProperty(future.DataSourceSpecBinding.SettlementDataProperty, datapb.PropertyKey_TYPE_DECIMAL)
		if err != nil {
			return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}

	default:
		err := ospec.EnsureBoundableProperty(future.DataSourceSpecBinding.SettlementDataProperty, datapb.PropertyKey_TYPE_INTEGER)
		if err != nil {
			return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}
	}

	// ensure the oracle spec for market termination can be constructed
	ospec, err = spec.New(*datasource.SpecFromDefinition(future.DataSourceSpecForTradingTermination))
	if err != nil {
		return types.ProposalErrorInvalidFutureProduct, err
	}

	switch future.DataSourceSpecBinding.TradingTerminationProperty {
	case spec.BuiltinTimestamp:
		if err := ospec.EnsureBoundableProperty(future.DataSourceSpecBinding.TradingTerminationProperty, datapb.PropertyKey_TYPE_TIMESTAMP); err != nil {
			return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid oracle spec binding for trading termination: %w", err)
		}
	default:
		if err := ospec.EnsureBoundableProperty(future.DataSourceSpecBinding.TradingTerminationProperty, datapb.PropertyKey_TYPE_BOOLEAN); err != nil {
			return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid oracle spec binding for trading termination: %w", err)
		}
	}

	return validateAsset(future.SettlementAsset, decimals, assets, deepCheck)
}

func validatePerps(perps *types.PerpsProduct, decimals uint64, assets Assets, et *enactmentTime, currentTime time.Time, deepCheck bool) (types.ProposalError, error) {
	settlData := &perps.DataSourceSpecForSettlementData
	if settlData == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementData
	}

	if settlData.Content() == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementData
	}

	ext, err := settlData.IsExternal()
	if err != nil {
		return types.ProposalErrorInvalidPerpsProduct, err
	}

	if !ext {
		return types.ProposalErrorInvalidPerpsProduct, ErrSettlementWithInternalDataSourceIsNotAllowed
	}

	settlSchedule := &perps.DataSourceSpecForSettlementSchedule
	if settlSchedule == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementSchedule
	}

	if settlSchedule.Content() == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementSchedule
	}

	if perps.DataSourceSpecBinding == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecBinding
	}

	// ensure the oracle spec for settlement data can be constructed
	ospec, err := spec.New(*datasource.SpecFromDefinition(perps.DataSourceSpecForSettlementData))
	if err != nil {
		return types.ProposalErrorInvalidPerpsProduct, err
	}
	switch perps.DataSourceSpecBinding.SettlementDataProperty {
	case datapb.PropertyKey_TYPE_DECIMAL.String():
		err := ospec.EnsureBoundableProperty(perps.DataSourceSpecBinding.SettlementDataProperty, datapb.PropertyKey_TYPE_DECIMAL)
		if err != nil {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}
	default:
		err := ospec.EnsureBoundableProperty(perps.DataSourceSpecBinding.SettlementDataProperty, datapb.PropertyKey_TYPE_INTEGER)
		if err != nil {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}
	}

	// ensure the oracle spec for market termination can be constructed
	ospec, err = spec.New(*datasource.SpecFromDefinition(perps.DataSourceSpecForSettlementSchedule))
	if err != nil {
		return types.ProposalErrorInvalidPerpsProduct, err
	}

	switch perps.DataSourceSpecBinding.SettlementScheduleProperty {
	case spec.BuiltinTimeTrigger:
		tt := perps.DataSourceSpecForSettlementSchedule.GetInternalTimeTriggerSpecConfiguration()
		if len(tt.Triggers) != 1 {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("invalid settlement schedule, only 1 trigger allowed")
		}

		if tt.Triggers[0] == nil {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("at least 1 time trigger is required")
		}

		if tt.Triggers[0].Initial == nil {
			tt.SetInitial(time.Unix(et.current, 0), currentTime)
		}
		tt.SetNextTrigger(currentTime)

		// can't have the first trigger in the past
		if tt.Triggers[0].Initial.Before(currentTime) {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("time trigger starts in the past")
		}

		if err := ospec.EnsureBoundableProperty(perps.DataSourceSpecBinding.SettlementScheduleProperty, datapb.PropertyKey_TYPE_TIMESTAMP); err != nil {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("invalid oracle spec binding for settlement schedule: %w", err)
		}
	default:
		return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("time trigger only supported for now")
	}

	return validateAsset(perps.SettlementAsset, decimals, assets, deepCheck)
}

func validateNewInstrument(instrument *types.InstrumentConfiguration, decimals uint64, assets Assets, et *enactmentTime, deepCheck bool, currentTime *time.Time) (types.ProposalError, error) {
	switch product := instrument.Product.(type) {
	case nil:
		return types.ProposalErrorNoProduct, ErrMissingProduct
	case *types.InstrumentConfigurationFuture:
		return validateFuture(product.Future, decimals, assets, et, deepCheck)
	case *types.InstrumentConfigurationPerps:
		return validatePerps(product.Perps, decimals, assets, et, *currentTime, deepCheck)
	case *types.InstrumentConfigurationSpot:
		return validateSpot(product.Spot, decimals, assets, deepCheck)
	default:
		return types.ProposalErrorUnsupportedProduct, ErrUnsupportedProduct
	}
}

func validateLogNormalRiskParams(lnm *types.LogNormalRiskModel) (types.ProposalError, error) {
	if lnm.Params == nil {
		return types.ProposalErrorInvalidRiskParameter, ErrInvalidRiskParameter
	}

	if lnm.RiskAversionParameter.LessThan(num.DecimalFromFloat(1e-8)) || lnm.RiskAversionParameter.GreaterThan(num.DecimalFromFloat(0.1)) || // 1e-8 <= lambda <= 0.1
		lnm.Tau.LessThan(num.DecimalFromFloat(1e-8)) || lnm.Tau.GreaterThan(num.DecimalOne()) || // 1e-8 <= tau <=1
		lnm.Params.Mu.LessThan(num.DecimalFromFloat(-1e-6)) || lnm.Params.Mu.GreaterThan(num.DecimalFromFloat(1e-6)) || // -1e-6 <= mu <= 1e-6
		lnm.Params.R.LessThan(num.DecimalFromInt64(-1)) || lnm.Params.R.GreaterThan(num.DecimalFromInt64(1)) || // -1 <= r <= 1
		lnm.Params.Sigma.LessThan(num.DecimalFromFloat(1e-3)) || lnm.Params.Sigma.GreaterThan(num.DecimalFromInt64(50)) { // 1e-3 <= sigma <= 50
		return types.ProposalErrorInvalidRiskParameter, ErrInvalidRiskParameter
	}
	return types.ProposalErrorUnspecified, nil
}

func validateRiskParameters(rp interface{}) (types.ProposalError, error) {
	switch r := rp.(type) {
	case *types.NewMarketConfigurationSimple:
		return types.ProposalErrorUnspecified, nil
	case *types.UpdateMarketConfigurationSimple:
		return types.ProposalErrorUnspecified, nil
	case *types.NewMarketConfigurationLogNormal:
		return validateLogNormalRiskParams(r.LogNormal)
	case *types.UpdateMarketConfigurationLogNormal:
		return validateLogNormalRiskParams(r.LogNormal)
	case *types.NewSpotMarketConfigurationLogNormal:
		return validateLogNormalRiskParams(r.LogNormal)
	case *types.UpdateSpotMarketConfigurationLogNormal:
		return validateLogNormalRiskParams(r.LogNormal)
	case nil:
		return types.ProposalErrorNoRiskParameters, ErrMissingRiskParameters
	default:
		return types.ProposalErrorUnknownRiskParameterType, ErrUnsupportedRiskParameters
	}
}

func validateLPSLAParams(slaParams *types.LiquiditySLAParams) (types.ProposalError, error) {
	if slaParams == nil {
		return types.ProposalErrorMissingSLAParams, fmt.Errorf("liquidity provision SLA must be provided")
	}
	if slaParams.PriceRange.LessThanOrEqual(num.DecimalZero()) {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("price range must be in range (0, 100]")
	}
	if slaParams.CommitmentMinTimeFraction.LessThan(num.DecimalZero()) || slaParams.CommitmentMinTimeFraction.GreaterThan(num.DecimalOne()) {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("commitment min time fraction must be in range [0, 1]")
	}
	if slaParams.ProvidersFeeCalculationTimeStep.Seconds() < 1 {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("provider fee calculation time step must be positive")
	}
	if slaParams.SlaCompetitionFactor.LessThan(num.DecimalZero()) || slaParams.CommitmentMinTimeFraction.GreaterThan(num.DecimalOne()) {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("sla competition factor must be in range [0, 1]")
	}
	if slaParams.PerformanceHysteresisEpochs < 1 {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("provider performance hysteresis epochs must be positive")
	}
	return types.ProposalErrorUnspecified, nil
}

func validateAuctionDuration(proposedDuration time.Duration, netp NetParams) (types.ProposalError, error) {
	minAuctionDuration, _ := netp.GetDuration(netparams.MarketAuctionMinimumDuration)
	if proposedDuration < minAuctionDuration {
		// Auction duration is too small
		return types.ProposalErrorOpeningAuctionDurationTooSmall,
			fmt.Errorf("proposal opening auction duration is too short, expected > %v, got %v", minAuctionDuration, proposedDuration)
	}
	maxAuctionDuration, _ := netp.GetDuration(netparams.MarketAuctionMaximumDuration)
	if proposedDuration > maxAuctionDuration {
		// Auction duration is too large
		return types.ProposalErrorOpeningAuctionDurationTooLarge,
			fmt.Errorf("proposal opening auction duration is too long, expected < %v, got %v", maxAuctionDuration, proposedDuration)
	}
	return types.ProposalErrorUnspecified, nil
}

func validateSlippageFactor(slippageFactor num.Decimal, isLinear bool) (types.ProposalError, error) {
	err := types.ProposalErrorLinearSlippageOutOfRange
	if !isLinear {
		err = types.ProposalErrorQuadraticSlippageOutOfRange
	}
	if slippageFactor.IsNegative() {
		return err, fmt.Errorf("proposal slippage factor has incorrect value, expected value in [0,1000000], got %s", slippageFactor.String())
	}
	if slippageFactor.GreaterThan(num.DecimalFromInt64(1000000)) {
		return err, fmt.Errorf("proposal slippage factor has incorrect value, expected value in [0,1000000], got %s", slippageFactor.String())
	}
	return types.ProposalErrorUnspecified, nil
}

func validateLpPriceRange(lpPriceRange num.Decimal) (types.ProposalError, error) {
	if lpPriceRange.IsZero() || lpPriceRange.IsNegative() || lpPriceRange.GreaterThan(num.DecimalFromInt64(100)) {
		return types.ProposalErrorLpPriceRangeNonpositive, fmt.Errorf("proposal LP price range has incorrect value, expected value in (0,100], got %s", lpPriceRange.String())
	}
	return types.ProposalErrorUnspecified, nil
}

func validateNewSpotMarketChange(
	terms *types.NewSpotMarket,
	assets Assets,
	deepCheck bool,
	netp NetParams,
	openingAuctionDuration time.Duration,
	etu *enactmentTime,
) (types.ProposalError, error) {
	if perr, err := validateNewInstrument(terms.Changes.Instrument, terms.Changes.DecimalPlaces, assets, etu, deepCheck, nil); err != nil {
		return perr, err
	}
	if perr, err := validateAuctionDuration(openingAuctionDuration, netp); err != nil {
		return perr, err
	}
	if terms.Changes.PriceMonitoringParameters != nil && len(terms.Changes.PriceMonitoringParameters.Triggers) > 5 {
		return types.ProposalErrorTooManyPriceMonitoringTriggers,
			fmt.Errorf("%v price monitoring triggers set, maximum allowed is 5", len(terms.Changes.PriceMonitoringParameters.Triggers) > 5)
	}
	if perr, err := validateRiskParameters(terms.Changes.RiskParameters); err != nil {
		return perr, err
	}
	if perr, err := validateLPSLAParams(terms.Changes.SLAParams); err != nil {
		return perr, err
	}
	return types.ProposalErrorUnspecified, nil
}

// ValidateNewMarket checks new market proposal terms.
func validateNewMarketChange(
	terms *types.NewMarket,
	assets Assets,
	deepCheck bool,
	netp NetParams,
	openingAuctionDuration time.Duration,
	etu *enactmentTime,
	parent *types.Market,
	currentTime time.Time,
	restore bool,
) (types.ProposalError, error) {
	// in all cases, the instrument must be specified and validated, successor markets included.
	if perr, err := validateNewInstrument(terms.Changes.Instrument, terms.Changes.DecimalPlaces, assets, etu, deepCheck, ptr.From(currentTime)); err != nil {
		return perr, err
	}
	// verify opening auction duration, works the same for successor markets
	if perr, err := validateAuctionDuration(openingAuctionDuration, netp); !etu.cpLoad && err != nil {
		return perr, err
	}
	// if this is a successor market, check if that's set up fine:
	if perr, err := validateSuccessorMarket(terms, parent, restore); err != nil {
		return perr, err
	}
	if perr, err := validateRiskParameters(terms.Changes.RiskParameters); err != nil {
		return perr, err
	}
	if terms.Changes.PriceMonitoringParameters != nil && len(terms.Changes.PriceMonitoringParameters.Triggers) > 5 {
		return types.ProposalErrorTooManyPriceMonitoringTriggers,
			fmt.Errorf("%v price monitoring triggers set, maximum allowed is 5", len(terms.Changes.PriceMonitoringParameters.Triggers) > 5)
	}
	if perr, err := validateLpPriceRange(terms.Changes.LpPriceRange); err != nil {
		return perr, err
	}
	if perr, err := validateSlippageFactor(terms.Changes.LinearSlippageFactor, true); err != nil {
		return perr, err
	}
	if perr, err := validateSlippageFactor(terms.Changes.QuadraticSlippageFactor, false); err != nil {
		return perr, err
	}
	return types.ProposalErrorUnspecified, nil
}

func validateSuccessorMarket(terms *types.NewMarket, parent *types.Market, restore bool) (types.ProposalError, error) {
	suc := terms.Successor()
	if (parent == nil && suc == nil) || (parent == nil && restore) {
		return types.ProposalErrorUnspecified, nil
	}
	// if parent is not nil, then terms.Successor() was not nil and vice-versa. Either both are set or neither is.
	if perr, err := validateInsurancePoolFraction(suc.InsurancePoolFraction); err != nil {
		return perr, err
	}
	if perr, err := validateParentProduct(terms, parent); err != nil {
		return perr, err
	}
	return types.ProposalErrorUnspecified, nil
}

func validateParentProduct(prop *types.NewMarket, parent *types.Market) (types.ProposalError, error) {
	// make sure parent and successor are future markets
	parentFuture := parent.GetFuture()
	propFuture := prop.Changes.GetFuture()
	if propFuture == nil || parentFuture == nil {
		return types.ProposalErrorInvalidSuccessorMarket, fmt.Errorf("parent and successor markets must both be future markets")
	}
	if propFuture.Future.SettlementAsset != parentFuture.Future.SettlementAsset {
		return types.ProposalErrorInvalidSuccessorMarket, fmt.Errorf("successor market must use asset %s", parentFuture.Future.SettlementAsset)
	}
	if propFuture.Future.QuoteName != parentFuture.Future.QuoteName {
		return types.ProposalErrorInvalidSuccessorMarket, fmt.Errorf("successor market must use quote name %s", parentFuture.Future.QuoteName)
	}
	return types.ProposalErrorUnspecified, nil
}

func validateInsurancePoolFraction(frac num.Decimal) (types.ProposalError, error) {
	one := num.DecimalFromInt64(1)
	if frac.IsNegative() || frac.GreaterThan(one) {
		return types.ProposalErrorInvalidSuccessorMarket, fmt.Errorf("insurance pool fraction should be in range 0-1, was %s", frac.String())
	}
	return types.ProposalErrorUnspecified, nil
}

// validateUpdateMarketChange checks market update proposal terms.
func validateUpdateSpotMarketChange(terms *types.UpdateSpotMarket) (types.ProposalError, error) {
	if perr, err := validateRiskParameters(terms.Changes.RiskParameters); err != nil {
		return perr, err
	}
	if perr, err := validateLPSLAParams(terms.Changes.SLAParams); err != nil {
		return perr, err
	}
	return types.ProposalErrorUnspecified, nil
}

// validateUpdateMarketChange checks market update proposal terms.
func validateUpdateMarketChange(terms *types.UpdateMarket, etu *enactmentTime, currentTime time.Time) (types.ProposalError, error) {
	if perr, err := validateUpdateInstrument(terms.Changes.Instrument, etu, currentTime); err != nil {
		return perr, err
	}
	if perr, err := validateRiskParameters(terms.Changes.RiskParameters); err != nil {
		return perr, err
	}
	if perr, err := validateLpPriceRange(terms.Changes.LpPriceRange); err != nil {
		return perr, err
	}
	if perr, err := validateSlippageFactor(terms.Changes.LinearSlippageFactor, true); err != nil {
		return perr, err
	}
	if perr, err := validateSlippageFactor(terms.Changes.QuadraticSlippageFactor, false); err != nil {
		return perr, err
	}
	return types.ProposalErrorUnspecified, nil
}

func validateUpdateInstrument(instrument *types.UpdateInstrumentConfiguration, et *enactmentTime, currentTime time.Time) (types.ProposalError, error) {
	switch product := instrument.Product.(type) {
	case nil:
		return types.ProposalErrorNoProduct, ErrMissingProduct
	case *types.UpdateInstrumentConfigurationFuture:
		return validateUpdateFuture(product.Future, et)
	case *types.UpdateInstrumentConfigurationPerps:
		return validateUpdatePerps(product.Perps, et, currentTime)
	default:
		return types.ProposalErrorUnsupportedProduct, ErrUnsupportedProduct
	}
}

func validateUpdateFuture(future *types.UpdateFutureProduct, et *enactmentTime) (types.ProposalError, error) {
	settlData := &future.DataSourceSpecForSettlementData
	if settlData == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForSettlementData
	}

	if settlData.Content() == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForSettlementData
	}

	ext, err := settlData.IsExternal()
	if err != nil {
		return types.ProposalErrorInvalidFutureProduct, err
	}

	if !ext {
		return types.ProposalErrorInvalidFutureProduct, ErrSettlementWithInternalDataSourceIsNotAllowed
	}

	tterm := &future.DataSourceSpecForTradingTermination
	if tterm == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForTradingTermination
	}

	if tterm.Content() == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForTradingTermination
	}

	tp, _ := tterm.Type()
	if tp == datasource.ContentTypeInternalTimeTriggerTermination {
		return types.ProposalErrorInvalidFutureProduct, ErrInternalTimeTriggerForFuturesInNotAllowed
	}

	filters := future.DataSourceSpecForTradingTermination.GetFilters()

	for i, f := range filters {
		if f.Key.Type == datapb.PropertyKey_TYPE_TIMESTAMP {
			for j, cond := range f.Conditions {
				v, err := strconv.ParseInt(cond.Value, 10, 64)
				if err != nil {
					return types.ProposalErrorInvalidFutureProduct, err
				}

				filters[i].Conditions[j].Value = strconv.FormatInt(v, 10)
				if !et.shouldNotVerify {
					if v <= et.current {
						return types.ProposalErrorInvalidFutureProduct, ErrDataSourceSpecTerminationTimeBeforeEnactment
					}
				}
			}
		}
	}

	future.DataSourceSpecForTradingTermination.UpdateFilters(filters)

	if future.DataSourceSpecBinding == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecBinding
	}

	// ensure the oracle spec for settlement data can be constructed
	ospec, err := spec.New(*datasource.SpecFromDefinition(future.DataSourceSpecForSettlementData))
	if err != nil {
		return types.ProposalErrorInvalidFutureProduct, err
	}
	switch future.DataSourceSpecBinding.SettlementDataProperty {
	case datapb.PropertyKey_TYPE_DECIMAL.String():
		err := ospec.EnsureBoundableProperty(future.DataSourceSpecBinding.SettlementDataProperty, datapb.PropertyKey_TYPE_DECIMAL)
		if err != nil {
			return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}

	default:
		err := ospec.EnsureBoundableProperty(future.DataSourceSpecBinding.SettlementDataProperty, datapb.PropertyKey_TYPE_INTEGER)
		if err != nil {
			return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}
	}

	// ensure the oracle spec for market termination can be constructed
	ospec, err = spec.New(*datasource.SpecFromDefinition(future.DataSourceSpecForTradingTermination))
	if err != nil {
		return types.ProposalErrorInvalidFutureProduct, err
	}

	switch future.DataSourceSpecBinding.TradingTerminationProperty {
	case spec.BuiltinTimestamp:
		if err := ospec.EnsureBoundableProperty(future.DataSourceSpecBinding.TradingTerminationProperty, datapb.PropertyKey_TYPE_TIMESTAMP); err != nil {
			return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid oracle spec binding for trading termination: %w", err)
		}
	default:
		if err := ospec.EnsureBoundableProperty(future.DataSourceSpecBinding.TradingTerminationProperty, datapb.PropertyKey_TYPE_BOOLEAN); err != nil {
			return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid oracle spec binding for trading termination: %w", err)
		}
	}

	return types.ProposalErrorUnspecified, nil
}

func validateUpdatePerps(perps *types.UpdatePerpsProduct, et *enactmentTime, currentTime time.Time) (types.ProposalError, error) {
	settlData := &perps.DataSourceSpecForSettlementData
	if settlData == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementData
	}

	if settlData.Content() == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementData
	}

	ext, err := settlData.IsExternal()
	if err != nil {
		return types.ProposalErrorInvalidPerpsProduct, err
	}

	if !ext {
		return types.ProposalErrorInvalidPerpsProduct, ErrSettlementWithInternalDataSourceIsNotAllowed
	}

	settlSchedule := &perps.DataSourceSpecForSettlementSchedule
	if settlSchedule == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementSchedule
	}

	if settlSchedule.Content() == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementSchedule
	}

	if perps.DataSourceSpecBinding == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecBinding
	}

	// ensure the oracle spec for settlement data can be constructed
	ospec, err := spec.New(*datasource.SpecFromDefinition(perps.DataSourceSpecForSettlementData))
	if err != nil {
		return types.ProposalErrorInvalidPerpsProduct, err
	}
	switch perps.DataSourceSpecBinding.SettlementDataProperty {
	case datapb.PropertyKey_TYPE_DECIMAL.String():
		err := ospec.EnsureBoundableProperty(perps.DataSourceSpecBinding.SettlementDataProperty, datapb.PropertyKey_TYPE_DECIMAL)
		if err != nil {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}
	default:
		err := ospec.EnsureBoundableProperty(perps.DataSourceSpecBinding.SettlementDataProperty, datapb.PropertyKey_TYPE_INTEGER)
		if err != nil {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}
	}

	// ensure the oracle spec for market termination can be constructed
	ospec, err = spec.New(*datasource.SpecFromDefinition(perps.DataSourceSpecForSettlementSchedule))
	if err != nil {
		return types.ProposalErrorInvalidPerpsProduct, err
	}

	switch perps.DataSourceSpecBinding.SettlementScheduleProperty {
	case spec.BuiltinTimeTrigger:
		tt := perps.DataSourceSpecForSettlementSchedule.GetInternalTimeTriggerSpecConfiguration()
		if len(tt.Triggers) != 1 {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("invalid settlement schedule, only 1 trigger allowed")
		}

		if tt.Triggers[0] == nil {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("at least 1 time trigger is required")
		}

		if tt.Triggers[0].Initial == nil {
			tt.SetInitial(time.Unix(et.current, 0), currentTime)
		}
		tt.SetNextTrigger(currentTime)

		// can't have the first trigger in the past
		if tt.Triggers[0].Initial.Before(currentTime) {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("time trigger starts in the past")
		}

		if err := ospec.EnsureBoundableProperty(perps.DataSourceSpecBinding.SettlementScheduleProperty, datapb.PropertyKey_TYPE_TIMESTAMP); err != nil {
			return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("invalid oracle spec binding for settlement schedule: %w", err)
		}
	default:
		return types.ProposalErrorInvalidPerpsProduct, fmt.Errorf("time trigger only supported for now")
	}

	return types.ProposalErrorUnspecified, nil
}
