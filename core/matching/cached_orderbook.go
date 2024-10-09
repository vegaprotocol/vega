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
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

type CachedOrderBook struct {
	*OrderBook
	cache BookCache
}

func NewCachedOrderBook(
	log *logging.Logger, config Config, market string, auction bool, peggedCounterNotify func(int64),
) *CachedOrderBook {
	return &CachedOrderBook{
		OrderBook: NewOrderBook(log, config, market, auction, peggedCounterNotify),
		cache:     NewBookCache(),
	}
}

func (b *CachedOrderBook) SetOffbookSource(obs OffbookSource) {
	b.OrderBook.SetOffbookSource(obs)
}

func (b *CachedOrderBook) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	providers, err := b.OrderBook.LoadState(ctx, payload)
	if err != nil {
		return providers, err
	}

	if b.auction {
		b.cache.Invalidate()
		b.log.Info("restoring orderbook cache for", logging.String("marketID", b.marketID))
		b.GetIndicativePriceAndVolume()
	}

	return providers, err
}

func (b *CachedOrderBook) EnterAuction() []*types.Order {
	b.cache.Invalidate()
	return b.OrderBook.EnterAuction()
}

func (b *CachedOrderBook) LeaveAuction(
	at time.Time,
) ([]*types.OrderConfirmation, []*types.Order, error) {
	b.cache.Invalidate()
	return b.OrderBook.LeaveAuction(at)
}

func (b *CachedOrderBook) CancelAllOrders(
	party string,
) ([]*types.OrderCancellationConfirmation, error) {
	b.cache.Invalidate()
	return b.OrderBook.CancelAllOrders(party)
}

func (b *CachedOrderBook) maybeInvalidateDuringAuction(orderID string) {
	bestBid, errBestBid := b.GetBestBidPrice()
	bestAsk, errBestAsk := b.GetBestAskPrice()
	// if any of side have not best price, let's invalidate
	if errBestBid != nil || errBestAsk != nil {
		b.cache.Invalidate()
		return
	}

	order, ok := b.ordersByID[orderID]
	if !ok {
		b.log.Panic("could not find order in order book", logging.OrderID(orderID))
	}

	// only invalidate cache if it gets in the
	// uncrossing range
	switch order.Side {
	case types.SideBuy:
		if order.Price.GTE(bestAsk) {
			b.cache.Invalidate()
		}
	case types.SideSell:
		if order.Price.LTE(bestBid) {
			b.cache.Invalidate()
		}
	}
}

func (b *CachedOrderBook) maybeInvalidateDuringAuctionNewOrder(order *types.Order) {
	bestBid, errBestBid := b.GetBestBidPrice()
	bestAsk, errBestAsk := b.GetBestAskPrice()
	// if any of side have not best price, let's invalidate
	if errBestBid != nil || errBestAsk != nil {
		b.cache.Invalidate()
		return
	}

	// only invalidate cache if it gets in the
	// uncrossing range
	switch order.Side {
	case types.SideBuy:
		if order.Price.GTE(bestAsk) {
			b.cache.Invalidate()
		}
	case types.SideSell:
		if order.Price.LTE(bestBid) {
			b.cache.Invalidate()
		}
	}
}

func (b *CachedOrderBook) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(order.ID)
	}
	return b.OrderBook.CancelOrder(order)
}

func (b *CachedOrderBook) RemoveOrder(order string) (*types.Order, error) {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(order)
	}
	return b.OrderBook.RemoveOrder(order)
}

func (b *CachedOrderBook) AmendOrder(
	originalOrder, amendedOrder *types.Order,
) error {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(amendedOrder.ID)
	}
	return b.OrderBook.AmendOrder(originalOrder, amendedOrder)
}

func (b *CachedOrderBook) ReplaceOrder(rm, rpl *types.Order) (*types.OrderConfirmation, error) {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(rpl.ID)
	}
	return b.OrderBook.ReplaceOrder(rm, rpl)
}

func (b *CachedOrderBook) SubmitOrder(
	order *types.Order,
) (*types.OrderConfirmation, error) {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuctionNewOrder(order)
	}
	return b.OrderBook.SubmitOrder(order)
}

func (b *CachedOrderBook) DeleteOrder(
	order *types.Order,
) (*types.Order, error) {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(order.ID)
	}
	return b.OrderBook.DeleteOrder(order)
}

func (b *CachedOrderBook) RemoveDistressedOrders(
	parties []events.MarketPosition,
) ([]*types.Order, error) {
	b.cache.Invalidate()
	return b.OrderBook.RemoveDistressedOrders(parties)
}

func (b *CachedOrderBook) GetIndicativePriceAndVolume() (*num.Uint, uint64, types.Side) {
	price, cachedPriceOk := b.cache.GetIndicativePrice()
	volume, cachedVolOk := b.cache.GetIndicativeVolume()
	side, cachedSideOk := b.cache.GetIndicativeUncrossingSide()
	if !cachedPriceOk || !cachedVolOk || !cachedSideOk {
		r := b.OrderBook.GetIndicativePriceAndVolume()
		price, volume, side = r.price, r.volume, r.side

		b.cache.SetIndicativePrice(price.Clone())
		b.cache.SetIndicativeVolume(volume)
		b.cache.SetIndicativeUncrossingSide(side)
	}
	return price, volume, side
}

func (b *CachedOrderBook) GetIndicativePrice() *num.Uint {
	price, ok := b.cache.GetIndicativePrice()

	if !ok {
		price = b.OrderBook.GetIndicativePrice()
		b.cache.SetIndicativePrice(price.Clone())
	}
	return price
}

func (b *CachedOrderBook) UpdateAMM(party string) {
	if !b.auction {
		return
	}

	b.cache.Invalidate()
	b.OrderBook.UpdateAMM(party)
}
