package execution

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
)

func validateAsset(asset string) error {
	//@TODO: call proper asset validation (once implemented)
	return nil
}

func validateFuture(timeSvc TimeService, future *types.FutureProduct) error {
	maturity, err := time.Parse(time.RFC3339, future.Maturity)
	if err != nil {
		errors.Wrap(err, "future product maturity timestamp")
	}
	now, err := timeSvc.GetTimeNow()
	if err != nil {
		return errors.Wrap(err, "failed to get current Vega network time")
	}
	if maturity.UnixNano() < now.UnixNano() {
		return ErrProductMaturityIsPast
	}
	return validateAsset(future.Asset)
}

func validateInstrument(timeSvc TimeService, instrument *types.InstrumentConfiguration) error {
	if instrument.BaseName == instrument.QuoteName {
		return ErrInvalidSecurity
	}

	switch product := instrument.Product.(type) {
	case nil:
		return ErrNoProduct
	case *types.InstrumentConfiguration_Future:
		return validateFuture(timeSvc, product.Future)
	default:
		return ErrProductInvalid
	}
}

func validateTradingMode(terms *types.NewMarketConfiguration) error {
	switch terms.TradingMode.(type) {
	case nil:
		return ErrNoTradingMode
	case *types.NewMarketConfiguration_Continuous, *types.NewMarketConfiguration_Discrete:
		return nil
	default:
		return ErrTradingModeInvalid
	}
}

// ValidateNewMarket checks new market proposal terms
func validateNewMarket(time TimeService, terms *types.NewMarketConfiguration) error {
	if err := validateInstrument(time, terms.Instrument); err != nil {
		return err
	}
	if err := validateTradingMode(terms); err != nil {
		return err
	}
	return nil
}
