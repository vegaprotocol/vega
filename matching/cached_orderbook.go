package matching

import (
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
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

func (b *CachedOrderBook) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	b.cache.Invalidate()
	return b.OrderBook.CancelOrder(order)
}

func (b *CachedOrderBook) RemoveOrder(order *types.Order) error {
	b.cache.Invalidate()
	return b.OrderBook.RemoveOrder(order)
}

func (b *CachedOrderBook) AmendOrder(
	originalOrder, amendedOrder *types.Order) error {
	b.cache.Invalidate()
	return b.OrderBook.AmendOrder(originalOrder, amendedOrder)
}

func (b *CachedOrderBook) SubmitOrder(
	order *types.Order) (*types.OrderConfirmation, error) {
	b.cache.Invalidate()
	return b.OrderBook.SubmitOrder(order)
}

func (b *CachedOrderBook) DeleteOrder(
	order *types.Order) (*types.Order, error) {
	// invalidate all caches
	b.cache.Invalidate()
	return b.OrderBook.DeleteOrder(order)
}

func (b *CachedOrderBook) RemoveDistressedOrders(
	parties []events.MarketPosition,
) ([]*types.Order, error) {
	b.cache.Invalidate()
	return b.OrderBook.RemoveDistressedOrders(parties)
}

func (b *CachedOrderBook) GetIndicativePriceAndVolume() (uint64, uint64, types.Side) {
	price, cachedPriceOk := b.cache.GetIndicativePrice()
	volume, cachedVolOk := b.cache.GetIndicativeVolume()
	side, cachedSideOk := b.cache.GetIndicativeUncrossingSide()
	if !cachedPriceOk || !cachedVolOk || !cachedSideOk {
		price, volume, side = b.OrderBook.GetIndicativePriceAndVolume()
		b.cache.SetIndicativePrice(price)
		b.cache.SetIndicativeVolume(volume)
		b.cache.SetIndicativeUncrossingSide(side)
	}
	return price, volume, side
}

func (b *CachedOrderBook) GetIndicativePrice() uint64 {
	price, ok := b.cache.GetIndicativePrice()

	if !ok {
		price = b.OrderBook.GetIndicativePrice()
		b.cache.SetIndicativePrice(price)
	}
	return price
}
