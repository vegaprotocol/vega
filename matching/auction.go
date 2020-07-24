package matching

import (
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type AuctionBook struct {
	buy, sell *OrderBookSide
	expiring  *ExpiringOrders
	byID      map[string]*types.Order
	levels    map[uint64]*auctionPriceLevel
	highest   *auctionPriceLevel
}

type auctionPriceLevel struct {
	trades        []*types.Trade
	price, volume uint64
	orders        map[string]*types.Order
}

func newAuctionBook(log *logging.Logger) *AuctionBook {
	return &AuctionBook{
		buy:      &OrderBookSide{log: log},
		sell:     &OrderBookSide{log: log},
		expiring: NewExpiringOrders(),
		byID:     map[string]*types.Order{},
		levels:   map[uint64]*auctionPriceLevel{},
		highest:  &auctionPriceLevel{}, // empty -> volume 0
	}
}

func (b *AuctionBook) applyOrder(order *types.Order) error {
	side, add := b.buy, b.sell
	if order.Side == types.Side_SIDE_BUY {
		side, add = add, side
	}
	uncross := *order
	cpy := &uncross
	trades, impactedOrders, _, err := side.uncross(cpy)
	// wash trade is rejected
	if err != nil && err == ErrWashTrade {
		return err
	}
	if cpy.Remaining == 0 {
		cpy.Status = types.Order_STATUS_FILLED
	} else if isPersistent(cpy) {
		add.addOrder(cpy, cpy.Side)
		if order.TimeInForce == types.Order_TIF_GTT {
			b.expiring.Insert(*cpy)
		}
	}

	for _, o := range impactedOrders {
		if o.Remaining == 0 {
			o.Status = types.Order_STATUS_FILLED
			if o.TimeInForce == types.Order_TIF_GTT {
				b.expiring.RemoveOrder(*o)
			}
			delete(b.byID, o.Id)
		}
	}

	// pointer to the original, unaltered order
	// so we can uncross at a later point (remove/amend)
	if cpy.Status == types.Order_STATUS_ACTIVE {
		b.byID[order.Id] = order
	}
	lvl := b.getLevel(cpy.Price)
	// this order is new add now:
	lvl.orders[cpy.Id] = cpy
	lvl.appendTrades(trades)
	lvl.appendOrders(impactedOrders)
	if b.highest != lvl && b.highest.volume < lvl.volume {
		b.highest = lvl
	}
	return nil
}

func (b *AuctionBook) amendOrder(original, amended *types.Order) {
	side := b.buy
	if original.Side == types.Side_SIDE_SELL {
		side = b.sell
	}
	_ = side.amendOrder(amended)
	if original.ExpiresAt != amended.ExpiresAt ||
		original.TimeInForce != amended.TimeInForce {
		b.expiring.RemoveOrder(*original)
		if amended.TimeInForce == types.Order_TIF_GTT {
			b.expiring.Insert(*amended)
		}
	}
}

// remove one or more orders (cancel orders + order expiry)
func (b *AuctionBook) removeOrders(buy, sell *OrderBookSide, orders ...types.Order) {
	// remove the original book
	for _, o := range orders {
		delete(b.byID, o.Id)
		if o.TimeInForce == types.Order_TIF_GTT {
			b.expiring.RemoveOrder(o)
		}
	}
	b.levels = make(map[uint64]*auctionPriceLevel, len(b.levels)) // we'll have to re-uncross the whole thing
	// copy over the un-uncrossed buy and sell sides (deep copies)
	b.buy, b.sell = buy, sell
	// now uncross all orders again
	byID := b.byID
	b.byID = make(map[string]*types.Order, len(byID)) // clear out the original map
	for _, o := range byID {
		_ = b.applyOrder(o)
	}
}

func (b *AuctionBook) getLevel(price uint64) *auctionPriceLevel {
	if l, ok := b.levels[price]; ok {
		return l
	}
	l := &auctionPriceLevel{
		price:  price,
		trades: []*types.Trade{},
		orders: map[string]*types.Order{},
	}
	b.levels[price] = l
	return l
}

// this returns the level we want to uncross
func (b *AuctionBook) getMarketLevel() *auctionPriceLevel {
	return b.highest
}

func (l *auctionPriceLevel) appendTrades(trades []*types.Trade) {
	l.trades = append(l.trades, trades...)
	for _, t := range trades {
		l.volume += t.Size
	}
}

func (l *auctionPriceLevel) appendOrders(orders []*types.Order) {
	for _, o := range orders {
		if _, ok := l.orders[o.Id]; !ok {
			l.orders[o.Id] = o
		}
	}
}
