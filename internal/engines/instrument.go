package engines

import (
	"time"

	"code.vegaprotocol.io/vega/internal/products"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrNoMarketClosingTime = errors.New("no market closing time")
)

type Instrument struct {
	ID       string
	Code     string
	Name     string
	Metadata *types.InstrumentMetadata
	Product  products.Product
}

func NewInstrument(pi *types.Instrument) (*Instrument, error) {
	product, err := products.New(pi.Product)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instanciate product from instrument configuration")
	}
	return &Instrument{
		ID:       pi.Id,
		Code:     pi.Code,
		Name:     pi.Name,
		Metadata: pi.Metadata,
		Product:  product,
	}, err
}

func (i *Instrument) GetMarketClosingTime() (time.Time, error) {
	switch p := i.Product.(type) {
	case *products.Future:
		return p.Maturity, nil
	default:
		return time.Time{}, ErrNoMarketClosingTime
	}
}
