package products

import (
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrNilProduct           = errors.New("nil product")
	ErrUnimplementedProduct = errors.New("unimplemented product")
)

type FinancialAmount struct {
	Asset  string
	Amount uint64
}

type Product interface {
	Settle(entryPrice uint64, netPosition uint64) (*FinancialAmount, error)
}

func New(pp interface{}) (Product, error) {
	if pp == nil {
		return nil, ErrNilProduct
	}
	switch p := pp.(type) {
	case *types.Instrument_Future:
		return newFuture(p.Future)
	default:
		return nil, ErrUnimplementedProduct
	}
}
