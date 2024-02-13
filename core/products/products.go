// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package products

import (
	"context"
	"errors"
	"time"

	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
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
//go:generate go run github.com/golang/mock/mockgen -destination mocks/mock.go -package mocks code.vegaprotocol.io/vega/core/products OracleEngine,Broker
type OracleEngine interface {
	ListensToSigners(dscommon.Data) bool
	Subscribe(context.Context, spec.Spec, spec.OnMatchedData) (spec.SubscriptionID, spec.Unsubscriber, error)
	Unsubscribe(context.Context, spec.SubscriptionID)
}

type Broker interface {
	Send(e events.Event)
	SendBatch(es []events.Event)
}

// Product is the interface provided by all product in vega.
type Product interface {
	Settle(*num.Uint, *num.Uint, num.Decimal) (amt *types.FinancialAmount, neg bool, rounding num.Decimal, err error)
	Value(markPrice *num.Uint) (*num.Uint, error)
	GetAsset() string
	IsTradingTerminated() bool
	ScaleSettlementDataToDecimalPlaces(price *num.Numeric, dp uint32) (*num.Uint, error)
	NotifyOnTradingTerminated(listener func(context.Context, bool))
	NotifyOnSettlementData(listener func(context.Context, *num.Numeric))
	Update(ctx context.Context, pp interface{}, oe OracleEngine) error
	UnsubscribeTradingTerminated(ctx context.Context)
	UnsubscribeSettlementData(ctx context.Context)
	RestoreSettlementData(*num.Numeric)
	UpdateAuctionState(context.Context, bool)

	// tell the product about an internal data-point such as a the current mark-price
	SubmitDataPoint(context.Context, *num.Uint, int64) error

	// snapshot specific
	Serialize() *snapshotpb.Product
	GetMarginIncrease(int64) num.Decimal
	GetData(t int64) *types.ProductData
	GetCurrentPeriod() uint64
}

// TimeService ...
type TimeService interface {
	GetTimeNow() time.Time
}

// New instance a new product from a Market framework product configuration.
func New(ctx context.Context, log *logging.Logger, pp interface{}, marketID string, ts TimeService, oe OracleEngine, broker Broker, assetDP uint32) (Product, error) {
	if pp == nil {
		return nil, ErrNilProduct
	}

	switch p := pp.(type) {
	case *types.InstrumentFuture:
		return NewFuture(ctx, log, p.Future, oe, assetDP)
	case *types.InstrumentPerps:
		return NewPerpetual(ctx, log, p.Perps, marketID, ts, oe, broker, assetDP)
	default:
		return nil, ErrUnimplementedProduct
	}
}
