package matching

import (
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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
	}
}

func (b *CachedOrderBook) EnterAuction() ([]*types.Order, error) {
	b.cache.Invalidate()
	return b.OrderBook.EnterAuction()
}

func (b *CachedOrderBook) LeaveAuction(
	at time.Time) ([]*types.OrderConfirmation, []*types.Order, error) {
	b.cache.Invalidate()
	return b.OrderBook.LeaveAuction(at)
}

func (b *CachedOrderBook) CancelAllOrders(
	party string) ([]*types.OrderCancellationConfirmation, error) {
	b.cache.Invalidate()
	return b.OrderBook.CancelAllOrders(party)
}

func (b *CachedOrderBook) maybeInvalidateDuringAuction(order *types.Order) {
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
	case types.Side_SIDE_BUY:
		if order.Price.GTE(bestAsk) {
			b.cache.Invalidate()
		}
	case types.Side_SIDE_SELL:
		if order.Price.LTE(bestBid) {
			b.cache.Invalidate()
		}
	}
}

func (b *CachedOrderBook) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(order)
	}
	return b.OrderBook.CancelOrder(order)
}

func (b *CachedOrderBook) RemoveOrder(order *types.Order) error {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(order)
	}
	return b.OrderBook.RemoveOrder(order)
}

func (b *CachedOrderBook) AmendOrder(
	originalOrder, amendedOrder *types.Order) error {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(amendedOrder)
	}
	return b.OrderBook.AmendOrder(originalOrder, amendedOrder)
}

func (b *CachedOrderBook) SubmitOrder(
	order *types.Order) (*types.OrderConfirmation, error) {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(order)
	}
	return b.OrderBook.SubmitOrder(order)
}

func (b *CachedOrderBook) DeleteOrder(
	order *types.Order) (*types.Order, error) {
	if !b.InAuction() {
		b.cache.Invalidate()
	} else {
		b.maybeInvalidateDuringAuction(order)
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
