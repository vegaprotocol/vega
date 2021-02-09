package products

import (
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrNilProduct signals the product passed in the constructor was nil
	ErrNilProduct = errors.New("nil product")
	// ErrUnimplementedProduct signal that the product passed to the
	// constructor was not nil, but the code as no knowledge of it.
	ErrUnimplementedProduct = errors.New("unimplemented product")
)

// OracleEngine ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_engine_mock.go -package mocks code.vegaprotocol.io/vega/products OracleEngine
type OracleEngine interface {
	Subscribe(spec oracles.OracleSpec, cb oracles.OnMatchedOracleData) oracles.SubscriptionID
	Unsubscribe(oracles.SubscriptionID)
}

// Product is the interface provided by all product in vega
type Product interface {
	Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error)
	Value(markPrice uint64) (uint64, error)
	GetAsset() string
}

// New instance a new product from a Market framework product configuration
func New(log *logging.Logger, pp interface{}, oe OracleEngine) (Product, error) {
	if pp == nil {
		return nil, ErrNilProduct
	}
	switch p := pp.(type) {
	case *types.Instrument_Future:
		return newFuture(log, p.Future, oe)
	default:
		return nil, ErrUnimplementedProduct
	}
}
