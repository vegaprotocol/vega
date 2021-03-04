package governance

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/netparams"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrNoProduct is returned if selected product is nil
	ErrNoProduct = errors.New("no product has been specified")
	// ErrProductInvalid is returned if selected product is not supported
	ErrProductInvalid = errors.New("specified product is not supported")
	// ErrProductMaturityIsPast is returned if product maturity is not in future
	ErrProductMaturityIsPast = errors.New("product maturity date is in the past")

	// ErrNoTradingMode is returned if trading mode is nil
	ErrNoTradingMode = errors.New("no trading mode has been selected")
	// ErrTradingModeInvalid is returned if selected trading mode is not supported
	ErrTradingModeInvalid = errors.New("selected trading mode is not supported")

	// ErrInvalidTradingMode is returned if supplied trading is not valid (has to be either continuous or descrete)
	ErrInvalidTradingMode = errors.New("trading mode is invalid")

	// ErrProductTypeNotSupported is returned if product type supplied via governance is not yet supported
	// (this error should really never occur)
	ErrProductTypeNotSupported = errors.New("product type is not supported")

	// ErrRiskParametersNotSupported is returned if risk parameters supplied via governance are not yet supported
	ErrRiskParametersNotSupported = errors.New("risk model parameters are not supported")
	// ErrMissingRiskParameters ...
	ErrMissingRiskParameters = errors.New("missing risk parameters")

	// ErrMissingOracleSpecBinging is return when the oracle spec binding is absent.
	ErrMissingOracleSpecBinding = errors.New("missing oracle spec binding")
	// ErrMissingOracleSpec is return when the oracle spec is absent.
	ErrMissingOracleSpec = errors.New("missing oracle spec")
	// ErrMissingFutureProduct is return when future product is absent from the instrument.
	ErrMissingFutureProduct = errors.New("missing oracle spec")
)

func assignProduct(
	source *types.InstrumentConfiguration,
	target *types.Instrument,
) (types.ProposalError, error) {
	switch product := source.Product.(type) {
	case *types.InstrumentConfiguration_Future:
		if product.Future == nil {
			return types.ProposalError_PROPOSAL_ERROR_INVALID_FUTUR_PRODUCT, ErrMissingFutureProduct
		}
		if product.Future.OracleSpec == nil {
			return types.ProposalError_PROPOSAL_ERROR_INVALID_FUTUR_PRODUCT, ErrMissingOracleSpec
		}
		if product.Future.OracleSpecBinding == nil {
			return types.ProposalError_PROPOSAL_ERROR_INVALID_FUTUR_PRODUCT, ErrMissingOracleSpecBinding
		}

		target.Product = &types.Instrument_Future{
			Future: &types.Future{
				Maturity:          product.Future.Maturity,
				SettlementAsset:   product.Future.SettlementAsset,
				QuoteName:         product.Future.QuoteName,
				OracleSpec:        product.Future.OracleSpec.ToOracleSpec(),
				OracleSpecBinding: product.Future.OracleSpecBinding,
			},
		}
	default:
		return types.ProposalError_PROPOSAL_ERROR_UNSUPPORTED_PRODUCT, ErrProductTypeNotSupported
	}
	return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func assignTradingMode(definition *types.NewMarketConfiguration, target *types.Market) error {
	switch mode := definition.TradingMode.(type) {
	case *types.NewMarketConfiguration_Continuous:
		target.TradingModeConfig = &types.Market_Continuous{
			Continuous: mode.Continuous,
		}
	case *types.NewMarketConfiguration_Discrete:
		target.TradingModeConfig = &types.Market_Discrete{
			Discrete: mode.Discrete,
		}
	default:
		return ErrInvalidTradingMode
	}
	return nil
}

func createInstrument(
	netp NetParams,
	input *types.InstrumentConfiguration,
	tags []string,
) (*types.Instrument, types.ProposalError, error) {
	initialMarkPrice, _ := netp.GetInt(netparams.MarketInitialMarkPrice)
	result := &types.Instrument{
		Name: input.Name,
		Code: input.Code,
		Metadata: &types.InstrumentMetadata{
			Tags: tags,
		},
		InitialMarkPrice: uint64(initialMarkPrice),
	}

	if perr, err := assignProduct(input, result); err != nil {
		return nil, perr, err
	}
	return result, types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func assignRiskModel(definition *types.NewMarketConfiguration, target *types.TradableInstrument) error {
	switch parameters := definition.RiskParameters.(type) {
	case *types.NewMarketConfiguration_Simple:
		target.RiskModel = &types.TradableInstrument_SimpleRiskModel{
			SimpleRiskModel: &types.SimpleRiskModel{
				Params: parameters.Simple,
			},
		}
	case *types.NewMarketConfiguration_LogNormal:
		target.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
			LogNormalRiskModel: parameters.LogNormal,
		}
	default:
		return ErrRiskParametersNotSupported
	}
	return nil
}

func createMarket(
	marketID string,
	definition *types.NewMarketConfiguration,
	netp NetParams,
	currentTime time.Time,
	assets Assets,
	openingAuctionDuration time.Duration,
) (*types.Market, types.ProposalError, error) {
	if perr, err := validateNewMarket(currentTime, definition, assets, true, netp, openingAuctionDuration); err != nil {
		return nil, perr, err
	}
	instrument, perr, err := createInstrument(netp, definition.Instrument, definition.Metadata)
	if err != nil {
		return nil, perr, err
	}

	// get factors for the market
	makerFee, _ := netp.Get(netparams.MarketFeeFactorsMakerFee)
	infraFee, _ := netp.Get(netparams.MarketFeeFactorsInfrastructureFee)
	liquiFee, _ := netp.Get(netparams.MarketFeeFactorsLiquidityFee)
	// get the margin scaling factors
	scalingFactors := types.ScalingFactors{}
	_ = netp.GetJSONStruct(netparams.MarketMarginScalingFactors, &scalingFactors)
	// get price monitoring parameters
	pmUpdateFreq, _ := netp.GetDuration(netparams.MarketPriceMonitoringUpdateFrequency)
	if definition.PriceMonitoringParameters == nil {
		pmParams := &types.PriceMonitoringParameters{}
		_ = netp.GetJSONStruct(netparams.MarketPriceMonitoringDefaultParameters, pmParams)
		definition.PriceMonitoringParameters = pmParams
	}

	// get target stake parameters
	tsTimeWindow, _ := netp.GetDuration(netparams.MarketTargetStakeTimeWindow)
	tsScalingFactor, _ := netp.GetFloat(netparams.MarketTargetStakeScalingFactor)

	market := &types.Market{
		Id:            marketID,
		DecimalPlaces: definition.DecimalPlaces,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          makerFee,
				InfrastructureFee: infraFee,
				LiquidityFee:      liquiFee,
			},
		},
		OpeningAuction: &types.AuctionDuration{
			Duration: int64(openingAuctionDuration.Seconds()),
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: instrument,
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					CollateralRelease: scalingFactors.CollateralRelease,
					InitialMargin:     scalingFactors.InitialMargin,
					SearchLevel:       scalingFactors.SearchLevel,
				},
			},
		},
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters:      definition.PriceMonitoringParameters,
			UpdateFrequency: int64(pmUpdateFreq.Seconds()),
		},
		TargetStakeParameters: &types.TargetStakeParameters{
			TimeWindow:    int64(tsTimeWindow.Seconds()),
			ScalingFactor: tsScalingFactor,
		},
	}
	if err := assignRiskModel(definition, market.TradableInstrument); err != nil {
		return nil, types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, err
	}
	if err := assignTradingMode(definition, market); err != nil {
		return nil, types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, err
	}
	return market, types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func validateAsset(assetID string, assets Assets, deepCheck bool) (types.ProposalError, error) {
	if len(assetID) <= 0 {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_ASSET,
			errors.New("missing asset ID")
	}

	if !deepCheck {
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
	}

	_, err := assets.Get(assetID)
	if err != nil {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_ASSET, err
	}
	if !assets.IsEnabled(assetID) {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_ASSET,
			fmt.Errorf("assets is not enabled %v", assetID)
	}

	return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func validateFuture(currentTime time.Time, future *types.FutureProduct, assets Assets, deepCheck bool) (types.ProposalError, error) {
	maturity, err := time.Parse(time.RFC3339, future.Maturity)
	if err != nil {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT_TIMESTAMP, errors.Wrap(err, "future product maturity timestamp")
	}

	if deepCheck && maturity.UnixNano() < currentTime.UnixNano() {
		return types.ProposalError_PROPOSAL_ERROR_PRODUCT_MATURITY_IS_PASSED, ErrProductMaturityIsPast
	}

	if future.OracleSpec == nil {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_FUTUR_PRODUCT, ErrMissingOracleSpec
	}
	if future.OracleSpecBinding == nil {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_FUTUR_PRODUCT, ErrMissingOracleSpecBinding
	}

	return validateAsset(future.SettlementAsset, assets, deepCheck)
}

func validateInstrument(currentTime time.Time, instrument *types.InstrumentConfiguration, assets Assets, deepCheck bool) (types.ProposalError, error) {
	switch product := instrument.Product.(type) {
	case nil:
		return types.ProposalError_PROPOSAL_ERROR_NO_PRODUCT, ErrNoProduct
	case *types.InstrumentConfiguration_Future:
		return validateFuture(currentTime, product.Future, assets, deepCheck)
	default:
		return types.ProposalError_PROPOSAL_ERROR_UNSUPPORTED_PRODUCT, ErrProductInvalid
	}
}

func validateTradingMode(terms *types.NewMarketConfiguration) (types.ProposalError, error) {
	switch terms.TradingMode.(type) {
	case nil:
		return types.ProposalError_PROPOSAL_ERROR_NO_TRADING_MODE, ErrNoTradingMode
	case *types.NewMarketConfiguration_Continuous, *types.NewMarketConfiguration_Discrete:
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
	default:
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, ErrTradingModeInvalid
	}
}

func validateRiskParameters(rp interface{}) (types.ProposalError, error) {
	switch rp.(type) {
	case *types.NewMarketConfiguration_Simple,
		*types.NewMarketConfiguration_LogNormal:
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
	case nil:
		return types.ProposalError_PROPOSAL_ERROR_NO_RISK_PARAMETERS, ErrMissingRiskParameters
	default:
		return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, ErrRiskParametersNotSupported
	}
}

func validateAuctionDuration(proposedDuration time.Duration, netp NetParams) (types.ProposalError, error) {
	minAuctionDuration, _ := netp.GetDuration(netparams.MarketAuctionMinimumDuration)
	if proposedDuration != 0 && proposedDuration < minAuctionDuration {
		// Auction duration is too small
		return types.ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL,
			fmt.Errorf("proposal opening auction duration is too short, expected > %v, got %v", minAuctionDuration, proposedDuration)
	}
	maxAuctionDuration, _ := netp.GetDuration(netparams.MarketAuctionMaximumDuration)
	if proposedDuration > maxAuctionDuration {
		// Auction duration is too large
		return types.ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE,
			fmt.Errorf("proposal opening auction duration is too long, expected < %v, got %v", maxAuctionDuration, proposedDuration)
	}
	return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

// ValidateNewMarket checks new market proposal terms
func validateNewMarket(
	currentTime time.Time,
	terms *types.NewMarketConfiguration,
	assets Assets,
	deepCheck bool,
	netp NetParams,
	openingAuctionDuration time.Duration,
) (types.ProposalError, error) {
	if perr, err := validateInstrument(currentTime, terms.Instrument, assets, deepCheck); err != nil {
		return perr, err
	}
	if perr, err := validateTradingMode(terms); err != nil {
		return perr, err
	}
	if perr, err := validateRiskParameters(terms.RiskParameters); err != nil {
		return perr, err
	}
	if perr, err := validateAuctionDuration(openingAuctionDuration, netp); err != nil {
		return perr, err
	}

	return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}
