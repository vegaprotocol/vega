package governance

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

var (
	// ErrInvalidSecurity is returned if invalid risk model type is selected
	ErrInvalidSecurity = errors.New("selected same base and quote security")

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
)

func assignProduct(
	parameters *NetworkParameters,
	source *types.InstrumentConfiguration,
	target *types.Instrument,
) error {

	switch product := source.Product.(type) {
	case *types.InstrumentConfiguration_Future:
		target.Product = &types.Instrument_Future{
			Future: &types.Future{
				Asset:    product.Future.Asset,
				Maturity: product.Future.Maturity,
				Oracle: &types.Future_EthereumEvent{
					EthereumEvent: &types.EthereumEvent{
						ContractID: parameters.FutureOracle.ContractID,
						Event:      parameters.FutureOracle.Event,
						Value:      parameters.FutureOracle.Value,
					},
				},
			},
		}
	default:
		return ErrProductTypeNotSupported
	}
	return nil
}

func assignTradingMode(definition *types.NewMarketConfiguration, target *types.Market) error {
	switch mode := definition.TradingMode.(type) {
	case *types.NewMarketConfiguration_Continuous:
		target.TradingMode = &types.Market_Continuous{
			Continuous: mode.Continuous,
		}
	case *types.NewMarketConfiguration_Discrete:
		target.TradingMode = &types.Market_Discrete{
			Discrete: mode.Discrete,
		}
	default:
		return ErrInvalidTradingMode
	}
	return nil
}

func createInstrument(
	parameters *NetworkParameters,
	input *types.InstrumentConfiguration,
	tags []string,
) (*types.Instrument, error) {

	result := &types.Instrument{
		Name:      input.Name,
		Code:      input.Code,
		BaseName:  input.BaseName,
		QuoteName: input.QuoteName,
		Metadata: &types.InstrumentMetadata{
			Tags: tags,
		},
		InitialMarkPrice: parameters.InitialMarkPrice,
	}

	if err := assignProduct(parameters, input, result); err != nil {
		return nil, err
	}
	return result, nil
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
	parameters *NetworkParameters,
	currentTime time.Time,
) (*types.Market, types.ProposalError, error) {
	if perr, err := validateNewMarket(currentTime, definition); err != nil {
		return nil, perr, err
	}
	instrument, err := createInstrument(parameters, definition.Instrument, definition.Metadata)
	if err != nil {
		return nil, types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, err
	}
	market := &types.Market{
		Id:            marketID,
		DecimalPlaces: definition.DecimalPlaces,
		TradableInstrument: &types.TradableInstrument{
			Instrument: instrument,
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					CollateralRelease: parameters.MarginConfiguration.CollateralRelease,
					InitialMargin:     parameters.MarginConfiguration.InitialMargin,
					SearchLevel:       parameters.MarginConfiguration.SearchLevel,
				},
			},
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

func validateAsset(asset string) (types.ProposalError, error) {
	//@TODO: call proper asset validation (once implemented)
	return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}

func validateFuture(currentTime time.Time, future *types.FutureProduct) (types.ProposalError, error) {
	maturity, err := time.Parse(time.RFC3339, future.Maturity)
	if err != nil {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT_TIMESTAMP, errors.Wrap(err, "future product maturity timestamp")
	}
	if maturity.UnixNano() < currentTime.UnixNano() {
		return types.ProposalError_PROPOSAL_ERROR_PRODUCT_MATURITY_IS_PASSED, ErrProductMaturityIsPast
	}
	return validateAsset(future.Asset)
}

func validateInstrument(currentTime time.Time, instrument *types.InstrumentConfiguration) (types.ProposalError, error) {
	if instrument.BaseName == instrument.QuoteName {
		return types.ProposalError_PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY, ErrInvalidSecurity
	}

	switch product := instrument.Product.(type) {
	case nil:
		return types.ProposalError_PROPOSAL_ERROR_NO_PRODUCT, ErrNoProduct
	case *types.InstrumentConfiguration_Future:
		return validateFuture(currentTime, product.Future)
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

// ValidateNewMarket checks new market proposal terms
func validateNewMarket(currentTime time.Time, terms *types.NewMarketConfiguration) (types.ProposalError, error) {
	if perr, err := validateInstrument(currentTime, terms.Instrument); err != nil {
		return perr, err
	}
	if perr, err := validateTradingMode(terms); err != nil {
		return perr, err
	}
	return types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED, nil
}
