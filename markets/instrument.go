package markets

import (
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/products"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"

	"github.com/pkg/errors"
)

var (
	// ErrNoMarketClosingTime signal that the instrument is invalid as missing
	// a market closing time
	ErrNoMarketClosingTime = errors.New("no market closing time")
)

// Instrument represent an instrument used in a market
type Instrument struct {
	ID               string
	Code             string
	Name             string
	Metadata         *types.InstrumentMetadata
	InitialMarkPrice uint64
	Product          products.Product

	Quote string
}

// TradableInstrument represent an instrument to be trade in a market
type TradableInstrument struct {
	Instrument       *Instrument
	MarginCalculator *types.MarginCalculator
	RiskModel        risk.Model
}

// NewTradableInstrument will instantiate a new tradable instrument
// using a market framework configuration for a tradable instrument
func NewTradableInstrument(log *logging.Logger, pti *types.TradableInstrument) (*TradableInstrument, error) {
	instrument, err := NewInstrument(pti.Instrument)
	if err != nil {
		return nil, err
	}
	asset := instrument.Product.GetAsset()
	if err != nil {
		return nil, err
	}
	riskModel, err := risk.NewModel(log, pti.RiskModel, asset)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate risk model")
	}
	return &TradableInstrument{
		Instrument:       instrument,
		MarginCalculator: pti.MarginCalculator,
		RiskModel:        riskModel,
	}, nil
}

// NewInstrument will instantiate a new instrument
// using a market framework configuration for a instrument
func NewInstrument(pi *types.Instrument) (*Instrument, error) {
	product, err := products.New(pi.Product)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate product from instrument configuration")
	}
	return &Instrument{
		ID:               pi.Id,
		Code:             pi.Code,
		Name:             pi.Name,
		Metadata:         pi.Metadata,
		Product:          product,
		InitialMarkPrice: pi.InitialMarkPrice,
	}, err
}

// GetMarketClosingTime return the maturity of the product
func (i *Instrument) GetMarketClosingTime() (time.Time, error) {
	switch p := i.Product.(type) {
	case *products.Future:
		return p.Maturity, nil
	default:
		return time.Time{}, ErrNoMarketClosingTime
	}
}
