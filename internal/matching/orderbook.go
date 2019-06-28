package matching

import (
	"fmt"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type OrderBook struct {
	log *logging.Logger
	Config

	cfgMu           *sync.Mutex
	marketID        string
	buy             *OrderBookSide
	sell            *OrderBookSide
	lastTradedPrice uint64
	latestTimestamp int64
	expiringOrders  []types.Order // keep a list of all expiring trades, these will be in timestamp ascending order.
}

// Create an order book with a given name
func NewOrderBook(log *logging.Logger, config Config, marketID string, proRataMode bool) *OrderBook {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &OrderBook{
		log:            log,
		marketID:       marketID,
		cfgMu:          &sync.Mutex{},
		buy:            &OrderBookSide{log: log, proRataMode: proRataMode},
		sell:           &OrderBookSide{log: log, proRataMode: proRataMode},
		Config:         config,
		expiringOrders: make([]types.Order, 0),
	}
}

func (s *OrderBook) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.cfgMu.Lock()
	s.Config = cfg
	s.cfgMu.Unlock()
}

// Cancel an order that is active on an order book. Market and Order ID are validated, however the order must match
// the order on the book with respect to side etc. The caller will typically validate this by using a store, we should
// not trust that the external world can provide these values reliably.
func (b *OrderBook) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	// Validate Market
	if order.MarketID != b.marketID {
		b.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("order-book", b.marketID))
	}

	// Validate Order ID must be present
	if order.Id == "" || len(order.Id) < 4 {
		b.log.Error("Order ID missing or invalid",
			logging.Order(*order),
			logging.String("order-book", b.marketID))

		return nil, types.ErrInvalidOrderID
	}

	if order.Side == types.Side_Buy {
		if err := b.buy.RemoveOrder(order); err != nil {
			b.log.Error("Failed to remove order (buy side)",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))

			return nil, types.ErrOrderRemovalFailure
		}
	} else {
		if err := b.sell.RemoveOrder(order); err != nil {
			b.log.Error("Failed to remove order (sell side)",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))

			return nil, types.ErrOrderRemovalFailure
		}
	}

	// Important to mark the order as cancelled (and no longer active)
	order.Status = types.Order_Cancelled

	result := &types.OrderCancellationConfirmation{
		Order: order,
	}
	return result, nil
}

func (b *OrderBook) AmendOrder(order *types.Order) error {
	if err := b.validateOrder(order); err != nil {
		b.log.Error("Order validation failure",
			logging.Order(*order),
			logging.Error(err),
			logging.String("order-book", b.marketID))

		return err
	}

	if order.Side == types.Side_Buy {
		if err := b.buy.amendOrder(order); err != nil {
			b.log.Error("Failed to amend (buy side)",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))

			return err
		}
	} else {
		if err := b.sell.amendOrder(order); err != nil {
			b.log.Error("Failed to amend (sell side)",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))

			return err
		}
	}

	return nil
}

// Add an order and attempt to uncross the book, returns a TradeSet protobuf message object
func (b *OrderBook) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	if err := b.validateOrder(order); err != nil {
		return nil, err
	}

	if order.CreatedAt > b.latestTimestamp {
		b.latestTimestamp = order.CreatedAt
	}

	if b.LogPriceLevelsDebug {
		b.PrintState("Entry state:")
	}

	// uncross with opposite
	trades, impactedOrders, lastTradedPrice := b.getOppositeSide(order.Side).uncross(order)
	if lastTradedPrice != 0 {
		b.lastTradedPrice = lastTradedPrice
	}

	// if state of the book changed show state
	if b.LogPriceLevelsDebug && len(trades) != 0 {
		b.PrintState("After uncross state:")
	}

	// if order is persistent type add to order book to the correct side
	if (order.Type == types.Order_GTC || order.Type == types.Order_GTT) && order.Remaining > 0 {

		// GTT orders need to be added to the expiring orders table, these orders will be removed when expired.
		if order.Type == types.Order_GTT {
			b.insertExpiringOrder(*order)
		}

		b.getSide(order.Side).addOrder(order, order.Side)

		if b.LogPriceLevelsDebug {
			b.PrintState("After addOrder state:")
		}
	}

	// did we fully fill the originating order?
	if order.Remaining == 0 {
		order.Status = types.Order_Filled
	}

	// update order statuses based on the order types if they didn't trade
	if (order.Type == types.Order_FOK || order.Type == types.Order_ENE) && order.Remaining == order.Size {
		order.Status = types.Order_Stopped
	}

	for idx := range impactedOrders {
		if impactedOrders[idx].Remaining == 0 {
			impactedOrders[idx].Status = types.Order_Filled

			// Ensure any fully filled impacted GTT orders are removed
			// from internal matching engine pending orders list
			if impactedOrders[idx].Type == types.Order_GTT {
				b.removePendingGttOrder(*impactedOrders[idx])
			}
		}
	}

	orderConfirmation := makeResponse(order, trades, impactedOrders)
	return orderConfirmation, nil
}

func (b *OrderBook) DeleteOrder(order *types.Order) error {
	err := b.getSide(order.Side).RemoveOrder(order)
	return err
}

// RemoveExpiredOrders removes any GTT orders that will expire on or before the expiration timestamp (epoch+nano).
// expirationTimestamp must be of the format unix epoch seconds with nanoseconds e.g. 1544010789803472469.
// RemoveExpiredOrders returns a slice of Orders that were removed, internally it will remove the orders from the
// matching engine price levels. The returned orders will have an Order_Expired status, ready to update in stores.
func (b *OrderBook) RemoveExpiredOrders(expirationTimestamp int64) []types.Order {
	// expiring orders are ordered by expiration time.
	// so we'll search for the position where the expirationTimestamp would be in the slice
	// e.g: if our timestamp is 4
	// []int{1, 2, 3, 4, 4, 4, 4, 6, 7, 8}
	// ~~~~~~~~~~~~~~~~~~~~~~~~^
	// Also we add + 1 to the timestamp so it would find the last in the list,
	// if not the previous example would return
	// []int{1, 2, 3, 4, 4, 4, 4, 6, 7, 8}
	// ~~~~~~~~~~~~~~~^
	// by adding + 1 we actuall get everything which is stricly before the expirationTimestamp
	i := sort.Search(len(b.expiringOrders), func(i int) bool { return b.expiringOrders[i].ExpiresAt >= expirationTimestamp+1 })

	// make slice with the right size for the expired orders
	// then copy them
	expiredOrders := make([]types.Order, i)
	copy(expiredOrders, b.expiringOrders[:i])

	// delete the orders now
	for at := range expiredOrders {
		b.DeleteOrder(&expiredOrders[at])
		expiredOrders[at].Status = types.Order_Expired
	}

	// mem move all orders to be expired at the beginning of the slice, so
	// we do not need to reallocate in the future and we'll just reuse the actual slice.
	b.expiringOrders = b.expiringOrders[:copy(b.expiringOrders[0:], b.expiringOrders[i:])]

	if b.LogRemovedOrdersDebug {
		b.log.Debug("Removed expired orders from order book",
			logging.String("order-book", b.marketID),
			logging.Int("expired-orders", len(expiredOrders)),
			logging.Int("remaining-orders", len(b.expiringOrders)))
	}

	return expiredOrders
}

func (b OrderBook) getSide(orderSide types.Side) *OrderBookSide {
	if orderSide == types.Side_Buy {
		return b.buy
	} else {
		return b.sell
	}
}

func (b *OrderBook) getOppositeSide(orderSide types.Side) *OrderBookSide {
	if orderSide == types.Side_Buy {
		return b.sell
	} else {
		return b.buy
	}
}

func (b *OrderBook) insertExpiringOrder(ord types.Order) {
	if len(b.expiringOrders) <= 0 {
		b.expiringOrders = append(b.expiringOrders, ord)
		return
	}

	// first find the position where this should be inserted
	i := sort.Search(len(b.expiringOrders), func(i int) bool { return b.expiringOrders[i].ExpiresAt >= ord.ExpiresAt })

	// append new elem first to make sure we have enough place
	// this would reallocate sufficiently then
	// no risk of this being a empty order, as it's overwritten just next with
	// the slice insert
	b.expiringOrders = append(b.expiringOrders, types.Order{})
	copy(b.expiringOrders[i+1:], b.expiringOrders[i:])
	b.expiringOrders[i] = ord
}

func (b OrderBook) removePendingGttOrder(order types.Order) bool {
	// this will return the index of the first order with an expiry matching the order expiry
	// e.g: []int{1, 2, 3, 4, 4, 4, 5, 6, 7, 8, 9}
	//                     ^ this will return index 3
	i := sort.Search(len(b.expiringOrders), func(i int) bool { return b.expiringOrders[i].ExpiresAt >= order.ExpiresAt })
	if i < len(b.expiringOrders) {
		// orders with the same expiry found, now we need to iterate over the result to find
		// an order with the same expiry and may the order ID
		for i <= len(b.expiringOrders) && b.expiringOrders[i].ExpiresAt == order.ExpiresAt {
			if b.expiringOrders[i].ExpiresAt == order.ExpiresAt {
				// we found our order, let's remove it
				b.expiringOrders = b.expiringOrders[:i+copy(b.expiringOrders[i:], b.expiringOrders[i+1:])]
				return true
			}
		}
	}
	return false
}

func makeResponse(order *types.Order, trades []*types.Trade, impactedOrders []*types.Order) *types.OrderConfirmation {
	return &types.OrderConfirmation{
		Order:                 order,
		PassiveOrdersAffected: impactedOrders,
		Trades:                trades,
	}
}

func (b *OrderBook) PrintState(types string) {
	b.log.Debug(fmt.Sprintf("%s", types))
	b.log.Debug("------------------------------------------------------------")
	b.log.Debug("                        BUY SIDE                            ")
	for _, priceLevel := range b.buy.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print(b.log)
		}
	}
	b.log.Debug("------------------------------------------------------------")
	b.log.Debug("                        SELL SIDE                           ")
	for _, priceLevel := range b.sell.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print(b.log)
		}
	}
	b.log.Debug("------------------------------------------------------------")
}
