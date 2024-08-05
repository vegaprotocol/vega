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

package governance

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	dsdefinition "code.vegaprotocol.io/vega/core/datasource/definition"
	ethcallcommon "code.vegaprotocol.io/vega/core/datasource/external/ethcall/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/execution/liquidation"
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
	ErrInvalidInsurancePoolFraction          = errors.New("insurnace pool fraction invalid")
	ErrUpdateMarketDifferentProduct          = errors.New("cannot update a market to a different product type")
	ErrInvalidEVMChainIDInEthereumOracleSpec = errors.New("invalid source chain id in ethereum oracle spec")
	ErrMaxPriceInvalid                       = errors.New("max price for capped future must be greater than zero")
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
				Cap:                                 product.Future.Cap,
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
				FundingRateScalingFactor:            product.Perps.FundingRateScalingFactor,
				FundingRateLowerBound:               product.Perps.FundingRateLowerBound,
				FundingRateUpperBound:               product.Perps.FundingRateUpperBound,
				DataSourceSpecForSettlementData:     datasource.SpecFromDefinition(product.Perps.DataSourceSpecForSettlementData),
				DataSourceSpecForSettlementSchedule: datasource.SpecFromDefinition(product.Perps.DataSourceSpecForSettlementSchedule),
				DataSourceSpecBinding:               product.Perps.DataSourceSpecBinding,
				InternalCompositePriceConfig:        product.Perps.InternalCompositePriceConfig,
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
	buybackFee, _ := netp.Get(netparams.MarketFeeFactorsBuyBackFee)
	treasuryFee, _ := netp.Get(netparams.MarketFeeFactorsTreasuryFee)

	// get the margin scaling factors
	scalingFactors := proto.ScalingFactors{}
	_ = netp.GetJSONStruct(netparams.MarketMarginScalingFactors, &scalingFactors)
	// get price monitoring parameters
	if definition.Changes.PriceMonitoringParameters == nil {
		pmParams := &proto.PriceMonitoringParameters{}
		_ = netp.GetJSONStruct(netparams.MarketPriceMonitoringDefaultParameters, pmParams)
		definition.Changes.PriceMonitoringParameters = types.PriceMonitoringParametersFromProto(pmParams)
	}

	// if a liquidity fee setting isn't supplied in the proposal, we'll default to margin-cost.
	if definition.Changes.LiquidityFeeSettings == nil {
		definition.Changes.LiquidityFeeSettings = &types.LiquidityFeeSettings{
			Method: proto.LiquidityFeeSettings_METHOD_MARGINAL_COST,
		}
	}

	// this can be nil for market updates.
	var lstrat *types.LiquidationStrategy
	if definition.Changes.LiquidationStrategy != nil {
		lstrat = definition.Changes.LiquidationStrategy.DeepClone()
	}
	makerFeeDec, _ := num.DecimalFromString(makerFee)
	infraFeeDec, _ := num.DecimalFromString(infraFee)
	buybackFeeDec, _ := num.DecimalFromString(buybackFee)
	treasuryFeeDec, _ := num.DecimalFromString(treasuryFee)
	// assign here, we want to update this after assigning market variable
	marginCalc := &types.MarginCalculator{
		ScalingFactors: types.ScalingFactorsFromProto(&scalingFactors),
	}
	market := &types.Market{
		ID:                    marketID,
		DecimalPlaces:         definition.Changes.DecimalPlaces,
		PositionDecimalPlaces: definition.Changes.PositionDecimalPlaces,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          makerFeeDec,
				InfrastructureFee: infraFeeDec,
				TreasuryFee:       treasuryFeeDec,
				BuyBackFee:        buybackFeeDec,
			},
			LiquidityFeeSettings: definition.Changes.LiquidityFeeSettings,
		},
		OpeningAuction: &types.AuctionDuration{
			Duration: int64(openingAuctionDuration.Seconds()),
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument:       instrument,
			MarginCalculator: marginCalc,
		},
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters: definition.Changes.PriceMonitoringParameters,
		},
		LiquidityMonitoringParameters: definition.Changes.LiquidityMonitoringParameters,
		LiquiditySLAParams:            definition.Changes.LiquiditySLAParameters,
		LinearSlippageFactor:          definition.Changes.LinearSlippageFactor,
		QuadraticSlippageFactor:       definition.Changes.QuadraticSlippageFactor,
		LiquidationStrategy:           lstrat,
		MarkPriceConfiguration:        definition.Changes.MarkPriceConfiguration,
		TickSize:                      definition.Changes.TickSize,
		EnableTxReordering:            definition.Changes.EnableTxReordering,
	}
	if fCap := market.TradableInstrument.Instrument.Product.Cap(); fCap != nil {
		marginCalc.FullyCollateralised = fCap.FullyCollateralised
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
	buybackFee, _ := netp.Get(netparams.MarketFeeFactorsBuyBackFee)
	treasuryFee, _ := netp.Get(netparams.MarketFeeFactorsTreasuryFee)
	// get price monitoring parameters
	if definition.Changes.PriceMonitoringParameters == nil {
		pmParams := &proto.PriceMonitoringParameters{}
		_ = netp.GetJSONStruct(netparams.MarketPriceMonitoringDefaultParameters, pmParams)
		definition.Changes.PriceMonitoringParameters = types.PriceMonitoringParametersFromProto(pmParams)
	}

	// if a liquidity fee setting isn't supplied in the proposal, we'll default to margin-cost.
	if definition.Changes.LiquidityFeeSettings == nil {
		definition.Changes.LiquidityFeeSettings = &types.LiquidityFeeSettings{
			Method: proto.LiquidityFeeSettings_METHOD_MARGINAL_COST,
		}
	}

	liquidityMonitoring := &types.LiquidityMonitoringParameters{
		TargetStakeParameters: definition.Changes.TargetStakeParameters,
	}

	makerFeeDec, _ := num.DecimalFromString(makerFee)
	infraFeeDec, _ := num.DecimalFromString(infraFee)
	buybackFeeDec, _ := num.DecimalFromString(buybackFee)
	treasuryFeeDec, _ := num.DecimalFromString(treasuryFee)
	market := &types.Market{
		ID:                    marketID,
		DecimalPlaces:         definition.Changes.PriceDecimalPlaces,
		PositionDecimalPlaces: definition.Changes.SizeDecimalPlaces,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          makerFeeDec,
				InfrastructureFee: infraFeeDec,
				TreasuryFee:       treasuryFeeDec,
				BuyBackFee:        buybackFeeDec,
			},
			LiquidityFeeSettings: definition.Changes.LiquidityFeeSettings,
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
		MarkPriceConfiguration:        defaultMarkPriceConfig,
		TickSize:                      definition.Changes.TickSize,
		EnableTxReordering:            definition.Changes.EnableTxReordering,
	}
	if err := assignSpotRiskModel(definition.Changes, market.TradableInstrument); err != nil {
		return nil, types.ProposalErrorUnspecified, err
	}
	return market, types.ProposalErrorUnspecified, nil
}

func validateAssetBasic(assetID string, assets Assets, positionDecimals int64, deepCheck bool) (types.ProposalError, error) {
	if len(assetID) <= 0 {
		return types.ProposalErrorInvalidAsset, errors.New("missing asset ID")
	}

	if !deepCheck {
		return types.ProposalErrorUnspecified, nil
	}

	as, err := assets.Get(assetID)
	if err != nil {
		return types.ProposalErrorInvalidAsset, err
	}
	if !assets.IsEnabled(assetID) {
		return types.ProposalErrorInvalidAsset,
			fmt.Errorf("asset is not enabled %v", assetID)
	}
	if positionDecimals > int64(as.DecimalPlaces()) {
		return types.ProposalErrorInvalidSizeDecimalPlaces, fmt.Errorf("number of position decimal places must be less than or equal to the number base asset decimal places")
	}

	return types.ProposalErrorUnspecified, nil
}

func validateAsset(assetID string, decimals uint64, positionDecimals int64, assets Assets, deepCheck bool) (types.ProposalError, error) {
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
			fmt.Errorf("asset is not enabled %v", assetID)
	}
	if int64(decimals)+positionDecimals > int64(asset.DecimalPlaces()) {
		return types.ProposalErrorTooManyMarketDecimalPlaces, errors.New("market decimal + position decimals must be less than or equal to asset decimals")
	}

	return types.ProposalErrorUnspecified, nil
}

func validateSpot(spot *types.SpotProduct, decimals uint64, positionDecimals int64, assets Assets, deepCheck bool) (types.ProposalError, error) {
	propError, err := validateAsset(spot.QuoteAsset, decimals, positionDecimals, assets, deepCheck)
	if err != nil {
		return propError, err
	}
	return validateAssetBasic(spot.BaseAsset, assets, positionDecimals, deepCheck)
}

func validateFuture(future *types.FutureProduct, decimals uint64, positionDecimals int64, assets Assets, et *enactmentTime, deepCheck bool, evmChainIDs []uint64, tickSize *num.Uint) (types.ProposalError, error) {
	future.DataSourceSpecForSettlementData = setDatasourceDefinitionDefaults(future.DataSourceSpecForSettlementData, et)
	future.DataSourceSpecForTradingTermination = setDatasourceDefinitionDefaults(future.DataSourceSpecForTradingTermination, et)

	settlData := &future.DataSourceSpecForSettlementData
	if settlData == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForSettlementData
	}

	if !settlData.EnsureValidChainID(evmChainIDs) {
		return types.ProposalErrorInvalidFutureProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
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

	if !tterm.EnsureValidChainID(evmChainIDs) {
		return types.ProposalErrorInvalidFutureProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
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
	if err := validateFutureCap(future.Cap, tickSize); err != nil {
		return types.ProposalErrorInvalidFutureProduct, fmt.Errorf("invalid capped future configuration: %w", err)
	}

	return validateAsset(future.SettlementAsset, decimals, positionDecimals, assets, deepCheck)
}

func validateFutureCap(fCap *types.FutureCap, tickSize *num.Uint) error {
	if fCap == nil {
		return nil
	}
	if fCap.MaxPrice.IsZero() {
		return ErrMaxPriceInvalid
	}
	// tick size of nil, zero, or one are fine for this check
	mod := num.UintOne()
	if tickSize == nil || tickSize.LTE(mod) {
		return nil
	}
	// if maxPrice % tickSize != 0, the max price is invalid
	if !mod.Mod(fCap.MaxPrice, tickSize).IsZero() {
		return ErrMaxPriceInvalid
	}

	return nil
}

func validatePerps(perps *types.PerpsProduct, decimals uint64, positionDecimals int64, assets Assets, et *enactmentTime, currentTime time.Time, deepCheck bool, evmChainIDs []uint64) (types.ProposalError, error) {
	perps.DataSourceSpecForSettlementData = setDatasourceDefinitionDefaults(perps.DataSourceSpecForSettlementData, et)
	perps.DataSourceSpecForSettlementSchedule = setDatasourceDefinitionDefaults(perps.DataSourceSpecForSettlementSchedule, et)

	settlData := &perps.DataSourceSpecForSettlementData
	if settlData == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementData
	}

	if !settlData.EnsureValidChainID(evmChainIDs) {
		return types.ProposalErrorInvalidPerpsProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
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

	if !settlSchedule.EnsureValidChainID(evmChainIDs) {
		return types.ProposalErrorInvalidPerpsProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
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

	if perps.InternalCompositePriceConfig != nil {
		for _, v := range perps.InternalCompositePriceConfig.DataSources {
			if !v.Data.EnsureValidChainID(evmChainIDs) {
				return types.ProposalErrorInvalidFutureProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
			}
		}
	}

	return validateAsset(perps.SettlementAsset, decimals, positionDecimals, assets, deepCheck)
}

func validateNewInstrument(instrument *types.InstrumentConfiguration, decimals uint64, positionDecimals int64, assets Assets, et *enactmentTime, deepCheck bool, currentTime *time.Time, evmChainIDs []uint64, tickSize *num.Uint) (types.ProposalError, error) {
	switch product := instrument.Product.(type) {
	case nil:
		return types.ProposalErrorNoProduct, ErrMissingProduct
	case *types.InstrumentConfigurationFuture:
		return validateFuture(product.Future, decimals, positionDecimals, assets, et, deepCheck, evmChainIDs, tickSize)
	case *types.InstrumentConfigurationPerps:
		return validatePerps(product.Perps, decimals, positionDecimals, assets, et, *currentTime, deepCheck, evmChainIDs)
	case *types.InstrumentConfigurationSpot:
		return validateSpot(product.Spot, decimals, positionDecimals, assets, deepCheck)
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
	case *types.NewSpotMarketConfigurationSimple:
		return types.ProposalErrorUnspecified, nil
	case *types.UpdateSpotMarketConfigurationSimple:
		return types.ProposalErrorUnspecified, nil
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

func validateLiquidationStrategy(ls *types.LiquidationStrategy) (types.ProposalError, error) {
	if ls == nil {
		// @TODO this will become a required parameter, but for now leave it as is
		// this will be implemented in at a later stage
		return types.ProposalErrorUnspecified, nil
	}
	if ls.DisposalFraction.IsZero() || ls.DisposalFraction.IsNegative() || ls.DisposalFraction.GreaterThan(num.DecimalOne()) {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("liquidation strategy disposal fraction must be in the 0-1 range and non-zero")
	}
	if ls.MaxFractionConsumed.IsZero() || ls.DisposalFraction.IsNegative() || ls.DisposalFraction.GreaterThan(num.DecimalOne()) {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("liquidation max fraction must be in the 0-1 range and non-zero")
	}
	if ls.DisposalTimeStep < time.Second {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("liquidation strategy time step has to be 1s or more")
	} else if ls.DisposalTimeStep > time.Hour {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("liquidation strategy time step can't be more than 1h")
	}
	if ls.DisposalSlippage.IsZero() || ls.DisposalSlippage.IsNegative() {
		return types.ProposalErrorInvalidMarket, fmt.Errorf("liquidation strategy must specify a disposal slippage range > 0")
	}
	return types.ProposalErrorUnspecified, nil
}

func validateLPSLAParams(slaParams *types.LiquiditySLAParams) (types.ProposalError, error) {
	if slaParams == nil {
		return types.ProposalErrorMissingSLAParams, fmt.Errorf("liquidity provision SLA must be provided")
	}
	if slaParams.PriceRange.IsZero() || slaParams.PriceRange.LessThan(num.DecimalZero()) || slaParams.PriceRange.GreaterThan(num.DecimalFromFloat(20)) {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("price range must be strictly greater than 0 and less than or equal to 20")
	}
	if slaParams.CommitmentMinTimeFraction.LessThan(num.DecimalZero()) || slaParams.CommitmentMinTimeFraction.GreaterThan(num.DecimalOne()) {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("commitment min time fraction must be in range [0, 1]")
	}
	if slaParams.SlaCompetitionFactor.LessThan(num.DecimalZero()) || slaParams.SlaCompetitionFactor.GreaterThan(num.DecimalOne()) {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("sla competition factor must be in range [0, 1]")
	}

	if slaParams.PerformanceHysteresisEpochs > 366 {
		return types.ProposalErrorInvalidSLAParams, fmt.Errorf("provider performance hysteresis epochs must be less then 366")
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

func getEVMChainIDs(netp NetParams) []uint64 {
	ethCfg := &proto.EthereumConfig{}
	if err := netp.GetJSONStruct(netparams.BlockchainsPrimaryEthereumConfig, ethCfg); err != nil {
		panic(fmt.Sprintf("could not load ethereum config from network parameter, this should never happen: %v", err))
	}
	cID, err := strconv.ParseUint(ethCfg.ChainId, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("could not convert chain id from ethereum config into integer: %v", err))
	}

	allIDs := []uint64{cID}
	l2Cfgs := &proto.EthereumL2Configs{}
	if err := netp.GetJSONStruct(netparams.BlockchainsEthereumL2Configs, l2Cfgs); err != nil {
		panic(fmt.Sprintf("could not load ethereum l2 config from network parameter, this should never happen: %v", err))
	}

	for _, v := range l2Cfgs.Configs {
		l2ID, err := strconv.ParseUint(v.ChainId, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("could not convert chain id from ethereum l2 config into integer: %v", err))
		}
		allIDs = append(allIDs, l2ID)
	}

	return allIDs
}

func validateNewSpotMarketChange(
	terms *types.NewSpotMarket,
	assets Assets,
	deepCheck bool,
	netp NetParams,
	openingAuctionDuration time.Duration,
	etu *enactmentTime,
) (types.ProposalError, error) {
	if perr, err := validateNewInstrument(terms.Changes.Instrument, terms.Changes.PriceDecimalPlaces, terms.Changes.SizeDecimalPlaces, assets, etu, deepCheck, nil, getEVMChainIDs(netp), terms.Changes.TickSize); err != nil {
		return perr, err
	}
	if perr, err := validateAuctionDuration(openingAuctionDuration, netp); err != nil {
		return perr, err
	}
	if terms.Changes.PriceMonitoringParameters != nil && len(terms.Changes.PriceMonitoringParameters.Triggers) > 100 {
		return types.ProposalErrorTooManyPriceMonitoringTriggers,
			fmt.Errorf("%v price monitoring triggers set, maximum allowed is 100", len(terms.Changes.PriceMonitoringParameters.Triggers) > 100)
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
	if perr, err := validateNewInstrument(terms.Changes.Instrument, terms.Changes.DecimalPlaces, terms.Changes.PositionDecimalPlaces, assets, etu, deepCheck, ptr.From(currentTime), getEVMChainIDs(netp), terms.Changes.TickSize); err != nil {
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
	if terms.Changes.PriceMonitoringParameters != nil && len(terms.Changes.PriceMonitoringParameters.Triggers) > 100 {
		return types.ProposalErrorTooManyPriceMonitoringTriggers,
			fmt.Errorf("%v price monitoring triggers set, maximum allowed is 100", len(terms.Changes.PriceMonitoringParameters.Triggers) > 100)
	}
	if perr, err := validateLPSLAParams(terms.Changes.LiquiditySLAParameters); err != nil {
		return perr, err
	}
	if perr, err := validateSlippageFactor(terms.Changes.LinearSlippageFactor, true); err != nil {
		return perr, err
	}
	if perr, err := validateSlippageFactor(terms.Changes.QuadraticSlippageFactor, false); err != nil {
		return perr, err
	}
	if terms.Changes.LiquidationStrategy == nil {
		// @TODO At this stage, we don't require the liquidation strategy to be specified, treating nil as an implied legacy strategy.
		terms.Changes.LiquidationStrategy = liquidation.GetLegacyStrat()
	} else if perr, err := validateLiquidationStrategy(terms.Changes.LiquidationStrategy); err != nil {
		return perr, err
	}

	if terms.Changes.MarkPriceConfiguration != nil {
		for _, v := range terms.Changes.MarkPriceConfiguration.DataSources {
			if !v.Data.EnsureValidChainID(getEVMChainIDs(netp)) {
				return types.ProposalErrorInvalidFutureProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
			}
		}
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
func validateUpdateMarketChange(terms *types.UpdateMarket, mkt types.Market, etu *enactmentTime, currentTime time.Time, netp NetParams) (types.ProposalError, error) {
	if perr, err := validateUpdateInstrument(terms.Changes.Instrument, mkt, etu, currentTime, getEVMChainIDs(netp)); err != nil {
		return perr, err
	}
	if perr, err := validateRiskParameters(terms.Changes.RiskParameters); err != nil {
		return perr, err
	}
	if perr, err := validateLPSLAParams(terms.Changes.LiquiditySLAParameters); err != nil {
		return perr, err
	}
	if perr, err := validateSlippageFactor(terms.Changes.LinearSlippageFactor, true); err != nil {
		return perr, err
	}
	if perr, err := validateSlippageFactor(terms.Changes.QuadraticSlippageFactor, false); err != nil {
		return perr, err
	}
	if perr, err := validateLiquidationStrategy(terms.Changes.LiquidationStrategy); err != nil {
		return perr, err
	}
	return types.ProposalErrorUnspecified, nil
}

func validateUpdateInstrument(instrument *types.UpdateInstrumentConfiguration, mkt types.Market, et *enactmentTime, currentTime time.Time, evmChainIDs []uint64) (types.ProposalError, error) {
	switch product := instrument.Product.(type) {
	case nil:
		return types.ProposalErrorNoProduct, ErrMissingProduct
	case *types.UpdateInstrumentConfigurationFuture:
		return validateUpdateFuture(product.Future, mkt, et, evmChainIDs)
	case *types.UpdateInstrumentConfigurationPerps:
		return validateUpdatePerps(product.Perps, mkt, et, currentTime, evmChainIDs)
	default:
		return types.ProposalErrorUnsupportedProduct, ErrUnsupportedProduct
	}
}

func validateUpdateFuture(future *types.UpdateFutureProduct, mkt types.Market, et *enactmentTime, evmChainIDs []uint64) (types.ProposalError, error) {
	if mkt.GetFuture() == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrUpdateMarketDifferentProduct
	}

	future.DataSourceSpecForSettlementData = setDatasourceDefinitionDefaults(future.DataSourceSpecForSettlementData, et)
	future.DataSourceSpecForTradingTermination = setDatasourceDefinitionDefaults(future.DataSourceSpecForTradingTermination, et)

	settlData := &future.DataSourceSpecForSettlementData
	if settlData == nil {
		return types.ProposalErrorInvalidFutureProduct, ErrMissingDataSourceSpecForSettlementData
	}

	if !settlData.EnsureValidChainID(evmChainIDs) {
		return types.ProposalErrorInvalidFutureProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
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

	if !tterm.EnsureValidChainID(evmChainIDs) {
		return types.ProposalErrorInvalidFutureProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
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

func validateUpdatePerps(perps *types.UpdatePerpsProduct, mkt types.Market, et *enactmentTime, currentTime time.Time, evmChainIDs []uint64) (types.ProposalError, error) {
	if mkt.GetPerps() == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrUpdateMarketDifferentProduct
	}

	perps.DataSourceSpecForSettlementData = setDatasourceDefinitionDefaults(perps.DataSourceSpecForSettlementData, et)
	perps.DataSourceSpecForSettlementSchedule = setDatasourceDefinitionDefaults(perps.DataSourceSpecForSettlementSchedule, et)

	settlData := &perps.DataSourceSpecForSettlementData
	if settlData == nil {
		return types.ProposalErrorInvalidPerpsProduct, ErrMissingDataSourceSpecForSettlementData
	}

	if !settlData.EnsureValidChainID(evmChainIDs) {
		return types.ProposalErrorInvalidPerpsProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
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

	if !settlSchedule.EnsureValidChainID(evmChainIDs) {
		return types.ProposalErrorInvalidPerpsProduct, ErrInvalidEVMChainIDInEthereumOracleSpec
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

		// can't have the first trigger in the past, don't recheck if we've come in from preEnact
		if !et.shouldNotVerify && tt.Triggers[0].Initial.Before(currentTime) {
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

func setDatasourceDefinitionDefaults(def dsdefinition.Definition, et *enactmentTime) dsdefinition.Definition {
	if def.IsEthCallSpec() {
		spec := def.GetEthCallSpec()
		if spec.Trigger != nil {
			switch trigger := spec.Trigger.(type) {
			case ethcallcommon.TimeTrigger:
				if trigger.Initial == 0 {
					trigger.Initial = uint64(et.current)
				}
				spec.Trigger = trigger
			}
		}
		def.DataSourceType = spec
	}

	return def
}
