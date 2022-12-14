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

package matching

import (
	"context"
	"log"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
)

func (b *OrderBook) StopSnapshots() {
	b.log.Debug("market has been cleared, stopping snapshot production", logging.String("marketid", b.marketID))
	b.stopped = true
}

func (b *OrderBook) Keys() []string {
	return []string{b.snapshot.Key()}
}

func (b *OrderBook) Stopped() bool {
	return b.stopped
}

func (b OrderBook) Namespace() types.SnapshotNamespace {
	return types.MatchingSnapshot
}

func (b *OrderBook) GetState(key string) ([]byte, []types.StateProvider, error) {
	if key != b.snapshot.Key() {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	if b.stopped {
		return nil, nil, nil
	}

	// Copy all the state into a domain object
	payload := b.buildPayload()

	s, err := proto.Marshal(payload.IntoProto())
	return s, nil, err
}

func (b *OrderBook) buildPayload() *types.Payload {
	return &types.Payload{
		Data: &types.PayloadMatchingBook{
			MatchingBook: &types.MatchingBook{
				MarketID:        b.marketID,
				Buy:             b.copyOrders(b.buy),
				Sell:            b.copyOrders(b.sell),
				LastTradedPrice: b.lastTradedPrice,
				Auction:         b.auction,
				BatchID:         b.batchID,
			},
		},
	}
}

func (b *OrderBook) copyOrders(obs *OrderBookSide) []*types.Order {
	orders := make([]*types.Order, 0)
	pricelevels := obs.getLevels()
	for _, pl := range pricelevels {
		for _, order := range pl.orders {
			orders = append(orders, order.Clone())
		}
	}
	return orders
}

func (b *OrderBook) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if b.Namespace() != payload.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	var mb *types.MatchingBook

	switch pl := payload.Data.(type) {
	case *types.PayloadMatchingBook:
		mb = pl.MatchingBook
	default:
		return nil, types.ErrUnknownSnapshotType
	}

	// Check we have an empty book here or else we should panic
	if len(b.buy.levels) > 0 || len(b.sell.levels) > 0 {
		log.Panic("orderbook is not empty so we should not be loading snapshot state")
	}

	b.marketID = mb.MarketID
	b.batchID = mb.BatchID
	b.auction = mb.Auction
	b.lastTradedPrice = mb.LastTradedPrice

	for _, o := range mb.Buy {
		b.buy.addOrder(o)
		b.add(o)
	}

	for _, o := range mb.Sell {
		b.sell.addOrder(o)
		b.add(o)
	}

	// If we are in an auction we need to build the IP&V structure
	if b.auction {
		b.indicativePriceAndVolume = NewIndicativePriceAndVolume(b.log, b.buy, b.sell)
	}
	return nil, nil
}

// RestoreWithMarketPriceFactor takes the given market price factor and updates all the OriginalPrices
// in the orders accordingly.
func (b *OrderBook) RestoreWithMarketPriceFactor(priceFactor *num.Uint) {
	for _, o := range b.ordersByID {
		if o.Price.IsZero() {
			continue
		}
		o.OriginalPrice = o.Price.Clone()
		o.OriginalPrice.Div(o.Price, priceFactor)
	}
}
