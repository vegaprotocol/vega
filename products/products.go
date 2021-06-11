package products

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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
	Subscribe(context.Context, oracles.OracleSpec, oracles.OnMatchedOracleData) oracles.SubscriptionID
	Unsubscribe(context.Context, oracles.SubscriptionID)
}

// Product is the interface provided by all product in vega
type Product interface {
	Settle(entryPrice *num.Uint, netPosition int64) (*types.FinancialAmount, error)
	Value(markPrice *num.Uint) (*num.Uint, error)
	GetAsset() string
}

// New instance a new product from a Market framework product configuration
func New(ctx context.Context, log *logging.Logger, pp interface{}, oe OracleEngine) (Product, error) {
	if pp == nil {
		return nil, ErrNilProduct
	}
	switch p := pp.(type) {
	case *types.Instrument_Future:
		return newFuture(ctx, log, p.Future, oe)
	default:
		return nil, ErrUnimplementedProduct
	}
}
