package products

import (
	"context"
	"errors"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/oracles"
	"code.vegaprotocol.io/data-node/types"
	"code.vegaprotocol.io/data-node/types/num"
)

var (
	// ErrNilProduct signals the product passed in the constructor was nil
	ErrNilProduct = errors.New("nil product")
	// ErrUnimplementedProduct signal that the product passed to the
	// constructor was not nil, but the code as no knowledge of it.
	ErrUnimplementedProduct = errors.New("unimplemented product")
)

// TODO - remove after all core functionality is remove from data node.
// This is here only to be able to compile data node.
type OnMatchedOracleData func(ctx context.Context, data oracles.OracleData) error
type OracleSpecPredicate func(spec oracles.OracleSpec) (bool, error)
type SubscriptionID uint64

// OracleEngine ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_engine_mock.go -package mocks code.vegaprotocol.io/data-node/products OracleEngine
type OracleEngine interface {
	Subscribe(context.Context, oracles.OracleSpec, OnMatchedOracleData) SubscriptionID
	Unsubscribe(context.Context, SubscriptionID)
}

// Product is the interface provided by all product in vega
type Product interface {
	Settle(entryPrice *num.Uint, netPosition int64) (*types.FinancialAmount, error)
	Value(markPrice *num.Uint) (*num.Uint, bool, error)
	GetAsset() string
	IsTradingTerminated() bool
	SettlementPrice() (*num.Uint, error)
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
