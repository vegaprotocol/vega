package vega

import (
	"errors"
	fmt "fmt"
	"strconv"
)

var (
	ErrNilTradableInstrument = errors.New("nil tradable instrument")
	ErrNilInstrument         = errors.New("nil instrument")
	ErrNilProduct            = errors.New("nil product")
	ErrUnknownAsset          = errors.New("unknown asset")
)

func (m *Market) GetAsset() (string, error) {
	if m.TradableInstrument == nil {
		return "", ErrNilTradableInstrument
	}
	if m.TradableInstrument.Instrument == nil {
		return "", ErrNilInstrument
	}
	if m.TradableInstrument.Instrument.Product == nil {
		return "", ErrNilProduct
	}

	switch pimpl := m.TradableInstrument.Instrument.Product.(type) {
	case *Instrument_Future:
		return pimpl.Future.SettlementAsset, nil
	default:
		return "", ErrUnknownAsset
	}
}

func (p *PriceMonitoringTrigger) Validate() error {
	if !(p.Horizon > 0) {
		return fmt.Errorf("invalid field Triggers.Horizon: value '%v' must be greater than '0'", p.Horizon)
	}

	probability, err := strconv.ParseFloat(p.Probability, 64)

	if err != nil {
		return fmt.Errorf("invalid field Triggers.Probability: value '%v' must be numeric and between 0 and 1", p.Probability)
	}

	if !(probability > 0) {
		return fmt.Errorf("invalid field Triggers.Probability: value '%v' must be strictly greater than '0'", p.Probability)
	}
	if !(probability < 1) {
		return fmt.Errorf("invalid field Triggers.Probability: value '%v' must be strictly lower than '1'", p.Probability)
	}
	if !(p.AuctionExtension > 0) {
		return fmt.Errorf("invalid field Triggers.AuctionExtension: value '%v' must be greater than '0'", p.AuctionExtension)
	}
	return nil
}

func (p *PriceMonitoringParameters) Validate() error {
	for _, item := range p.Triggers {
		if item != nil {
			if err := item.Validate(); err != nil {
				return err
			}
		}
	}
	return nil
}
