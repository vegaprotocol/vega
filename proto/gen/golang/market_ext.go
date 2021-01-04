package golang

import (
	"errors"
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
		return pimpl.Future.Asset, nil
	default:
		return "", ErrUnknownAsset
	}
}
