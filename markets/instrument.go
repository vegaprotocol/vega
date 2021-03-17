package markets

import (
	"context"
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
func NewTradableInstrument(ctx context.Context, log *logging.Logger, pti *types.TradableInstrument, oe products.OracleEngine) (*TradableInstrument, error) {
	instrument, err := NewInstrument(ctx, log, pti.Instrument, oe)
	if err != nil {
		return nil, err
	}
	asset := instrument.Product.GetAsset()
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
func NewInstrument(ctx context.Context, log *logging.Logger, pi *types.Instrument, oe products.OracleEngine) (*Instrument, error) {
	product, err := products.New(ctx, log, pi.Product, oe)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate product from instrument configuration")
	}
	return &Instrument{
		ID:               pi.Id,
		Code:             pi.Code,
		Name:             pi.Name,
		Metadata:         pi.Metadata,
		Product:          product,
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
