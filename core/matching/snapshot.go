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
				PeggedOrderIDs:  b.GetActivePeggedOrderIDs(),
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

	if len(mb.PeggedOrderIDs) != 0 {
		// the pegged orders will be added in an arbitrary order during b.add() above
		// which is all we can do if we've upgraded from older versions. If we have peggedOrder IDs
		// in the snapshot then we clear them and re-add in the snapshot order
		// (which will be the order they were added to the book)
		b.peggedOrders.Clear()
		for _, pid := range mb.PeggedOrderIDs {
			b.peggedOrders.Add(pid)
		}
	}

	if b.auction {
		b.indicativePriceAndVolume = NewIndicativePriceAndVolume(b.log, b.buy, b.sell, b.marketID)
	}

	return nil, nil
}

// RestoreWithMarketPriceFactor takes the given market price factor and updates all the OriginalPrices
// in the orders accordingly.
func (b *OrderBook) RestoreWithMarketPriceFactor(priceFactor num.Decimal) {
	for _, o := range b.ordersByID {
		if o.Price.IsZero() {
			continue
		}
		o.OriginalPrice, _ = num.UintFromDecimal(o.Price.ToDecimal().Div(priceFactor))
	}
}
