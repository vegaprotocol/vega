package products

import (
	types "code.vegaprotocol.io/vega/proto/gen/golang"

	"github.com/pkg/errors"
)

var (
	// ErrNilProduct signals the product passed in the constructor was nil
	ErrNilProduct = errors.New("nil product")
	// ErrUnimplementedProduct signal that the product passed to the
	// constructor was not nil, but the code as no knowledged of it.
	ErrUnimplementedProduct = errors.New("unimplemented product")
)

// Product is the interface provided by all product in vega
type Product interface {
	Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error)
	Value(markPrice uint64) (uint64, error)
	GetAsset() string
}

// New instance a new product from a Market frameword product configuration
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
