package products

import (
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrNilProduct           = errors.New("nil product")
	ErrUnimplementedProduct = errors.New("unimplemented product")
)

type Product interface {
	Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error)
	Value(markPrice uint64) (uint64, error)
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
