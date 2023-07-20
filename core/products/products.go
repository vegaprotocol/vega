// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package products

import (
	"context"
	"errors"

	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var (
	// ErrNilProduct signals the product passed in the constructor was nil.
	ErrNilProduct = errors.New("nil product")
	// ErrUnimplementedProduct signal that the product passed to the
	// constructor was not nil, but the code as no knowledge of it.
	ErrUnimplementedProduct = errors.New("unimplemented product")
)

// OracleEngine ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/oracle_engine_mock.go -package mocks code.vegaprotocol.io/vega/core/products OracleEngine
type OracleEngine interface {
	ListensToSigners(dscommon.Data) bool
	Subscribe(context.Context, spec.Spec, spec.OnMatchedData) (spec.SubscriptionID, spec.Unsubscriber, error)
	Unsubscribe(context.Context, spec.SubscriptionID)
}

// Product is the interface provided by all product in vega.
type Product interface {
	Settle(*num.Uint, *num.Uint, num.Decimal) (amt *types.FinancialAmount, neg bool, err error)
	Value(markPrice *num.Uint) (*num.Uint, error)
	GetAsset() string
	IsTradingTerminated() bool
	ScaleSettlementDataToDecimalPlaces(price *num.Numeric, dp uint32) (*num.Uint, error)
	NotifyOnTradingTerminated(listener func(context.Context, bool))
	NotifyOnSettlementData(listener func(context.Context, *num.Numeric))
	UnsubscribeTradingTerminated(ctx context.Context)
	UnsubscribeSettlementData(ctx context.Context)
	RestoreSettlementData(*num.Numeric)
}

// New instance a new product from a Market framework product configuration.
func New(ctx context.Context, log *logging.Logger, pp interface{}, oe OracleEngine) (Product, error) {
	if pp == nil {
		return nil, ErrNilProduct
	}
	switch p := pp.(type) {
	case *types.InstrumentFuture:
		return NewFuture(ctx, log, p.Future, oe)
	default:
		return nil, ErrUnimplementedProduct
	}
}
