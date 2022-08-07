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
	log *logging.Logger, config Config, market string, auction bool,
) *CachedOrderBook {
	return &CachedOrderBook{
		OrderBook: NewOrderBook(log, config, market, auction),
		cache:     NewBookCache(),
	}
}

func (b *CachedOrderBook) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	providers, err := b.OrderBook.LoadState(ctx, payload)
	if err != nil {
		return providers, err
	}

	// when a market is restored we call `GetMarketData` which fills this cache based on an unrestored orderbook,
	// now we have restored we need to recalculate.
	b.log.Info("restoring orderbook cache for", logging.String("marketID", b.marketID))
	b.cache.Invalidate()
	b.GetIndicativePriceAndVolume()
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
	bestAsk, errBestAsk := b.GetBestBidPrice()
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
	bestAsk, errBestAsk := b.GetBestBidPrice()
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
		price, volume, side = b.OrderBook.GetIndicativePriceAndVolume()
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
