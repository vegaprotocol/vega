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

	// ErrInvalidRiskModelType is returned if invalid risk model type is selected
	ErrInvalidRiskModelType = errors.New("invalid risk model selected")
	// ErrRiskModelTypeNotSupported is returned if selected risk model has not yet been implemented
	ErrRiskModelTypeNotSupported = errors.New("selected risk model is not supported")
	// ErrIncompatibleRiskParameters is returned if selected risk model is not
	// compatible with supplied risk model parameters
	ErrIncompatibleRiskParameters = errors.New("risk model parameters are not compatible with selected risk model")

	// ErrNoTradingMode is returned if trading mode is nil
	ErrNoTradingMode = errors.New("no trading mode has been selected")
	// ErrTradingModeInvalid is returned if selected trading mode is not supported
	ErrTradingModeInvalid = errors.New("selected trading mode is not supported")
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/governance TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
}

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

func validateInstrument(timeSvc TimeService, instrument *types.IntrumentConfiguration) error {
	if instrument.BaseName == instrument.QuoteName {
		return ErrInvalidSecurity
	}
	if instrument.GetProduct() == nil {
		return ErrNoProduct
	}
	if future := instrument.GetFuture(); future != nil {
		if err := validateFuture(timeSvc, future); err != nil {
			return err
		}
	} else {
		return ErrProductInvalid
	}
	return nil
}

func validateRiskModel(risk *types.RiskConfiguration) error {
	if risk.Model == types.RiskConfiguration_MODEL_UNSPECIFIED {
		return ErrInvalidRiskModelType
	} else if risk.Model == types.RiskConfiguration_MODEL_SIMPLE {
		if risk.GetSimple() == nil {
			return ErrIncompatibleRiskParameters
		}
	} else if risk.Model == types.RiskConfiguration_MODEL_LOG_NORMAL {
		if risk.GetLogNormal() == nil {
			return ErrIncompatibleRiskParameters
		}
	} else {
		return ErrRiskModelTypeNotSupported
	}
	return nil
}

func validateTradingMode(terms *types.NewMarketConfiguration) error {
	if terms.GetTradingMode() == nil {
		return ErrNoTradingMode
	}
	if terms.GetContinuous() == nil && terms.GetDiscrete() == nil {
		return ErrTradingModeInvalid
	}
	return nil
}

// ValidateNewMarket checks new market proposal terms
func ValidateNewMarket(time TimeService, terms *types.NewMarketConfiguration) error {
	if err := validateInstrument(time, terms.Instrument); err != nil {
		return err
	}
	if err := validateRiskModel(terms.Risk); err != nil {
		return err
	}
	if err := validateTradingMode(terms); err != nil {
		return err
	}
	return nil
}
