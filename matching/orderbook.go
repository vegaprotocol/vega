package matching

import (
	"fmt"

	"vega/log"
	"vega/msg"
)

type OrderBook struct {
	name            string
	buy             *OrderBookSide
	sell            *OrderBookSide
	lastTradedPrice uint64
	config          *Config
	latestTimestamp uint64

	expiringOrders []*msg.Order  // keep a list of all expiring trades, these will be in timestamp ascending order.
}

// Create an order book with a given name
func NewBook(name string, config *Config) *OrderBook {
	return &OrderBook{
		name:   name,
		buy:    &OrderBookSide{prorataMode: config.ProrataMode},
		sell:   &OrderBookSide{prorataMode: config.ProrataMode},
		config: config,
		expiringOrders: make([]*msg.Order, 0),
	}
}

// Cancel an order that is active on an order book. Market and Order ID are validated, however the order must match
// the order on the book with respect to side etc. The caller will typically validate this by using a store, we should
// not trust that the external world can provide these values reliably.
func (b *OrderBook) CancelOrder(order *msg.Order) (*msg.OrderCancellation, msg.OrderError) {
	// Validate Market
	if order.Market != b.name {
		log.Errorf(fmt.Sprintf(
			"Market ID mismatch\norderMessage.Market: %v\nbook.ID: %v",
			order.Market,
			b.name))
		return nil, msg.OrderError_INVALID_MARKET_ID
	}
	// Validate Order ID must be present
	if order.Id == "" || len(order.Id) < 4 {
		return nil, msg.OrderError_INVALID_ORDER_ID
	}

	if order.Side == msg.Side_Buy {
		if err := b.buy.RemoveOrder(order); err != nil {
			log.Errorf("Error removing (buy side): ", err)
			return nil, msg.OrderError_ORDER_REMOVAL_FAILURE
		}
	} else {
		if err := b.sell.RemoveOrder(order); err != nil {
			log.Errorf("Error removing (sell side): ", err)
			return nil, msg.OrderError_ORDER_REMOVAL_FAILURE
		}
	}

	// Important, mark the order as cancelled (and no longer active)
	order.Status = msg.Order_Cancelled

	result := &msg.OrderCancellation{
		Order: order,
	}
	return result, msg.OrderError_NONE
}

func (b *OrderBook) AmendOrder(order *msg.Order) msg.OrderError {
	if err := b.validateOrder(order); err != msg.OrderError_NONE {
		return err
	}

	if order.Side == msg.Side_Buy {
		if err := b.buy.amendOrder(order); err != msg.OrderError_NONE {
			log.Errorf("Error amending (buy side): ", err)
			return err
		}
	} else {
		if err := b.sell.amendOrder(order); err != msg.OrderError_NONE {
			log.Errorf("Error amending (sell side): ", err)
			return err
		}
	}

	return msg.OrderError_NONE
}

// Add an order and attempt to uncross the book, returns a TradeSet protobufs message object
func (b *OrderBook) AddOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	if err := b.validateOrder(order); err != msg.OrderError_NONE {
		return nil, err
	}

	if order.Timestamp > b.latestTimestamp {
		b.latestTimestamp = order.Timestamp
	}

	if b.config.LogPriceLevels {
		b.PrintState("Entry state:")
	}

	// uncross with opposite
	trades, impactedOrders, lastTradedPrice := b.getOppositeSide(order.Side).uncross(order)

	if lastTradedPrice != 0 {
		b.lastTradedPrice = lastTradedPrice
	}

	// if state of the book changed show state
	if b.config.LogPriceLevels && len(trades) != 0 {
		b.PrintState("After uncross state:")
	}

	// if order is persistent type add to order book to the correct side
	if (order.Type == msg.Order_GTC || order.Type == msg.Order_GTT) && order.Remaining > 0 {

		// GTT orders need to be added to the expiring orders table, these orders will be removed when expired.
		if order.Type == msg.Order_GTT {
			b.expiringOrders = append(b.expiringOrders, order)
		}

		b.getSide(order.Side).addOrder(order, order.Side)

		if b.config.LogPriceLevels {
			b.PrintState("After addOrder state:")
		}
	}

	// update order statuses based on the order types
	if order.Type == msg.Order_FOK {
		if order.Remaining == order.Size {
			order.Status = msg.Order_Stopped
		} else {
			order.Status = msg.Order_Filled
		}
	}

	if order.Type == msg.Order_ENE {
		if order.Remaining == order.Size {
			order.Status = msg.Order_Stopped
		} else {
			order.Status = msg.Order_Filled
		}
	}

	for idx, _ := range impactedOrders {
		if impactedOrders[idx].Remaining == 0 {
			impactedOrders[idx].Status = msg.Order_Filled
		}
	}

	orderConfirmation := makeResponse(order, trades, impactedOrders)
	return orderConfirmation, msg.OrderError_NONE
}

func (b *OrderBook) RemoveOrder(order *msg.Order) error {
	err := b.getSide(order.Side).RemoveOrder(order)
	return err
}

// RemoveExpiredOrders removes any GTT orders that will expire on or before the expiration timestamp (epoch+nano).
// expirationTimestamp must be of the format unix epoch seconds with nanoseconds e.g. 1544010789803472469.
// RemoveExpiredOrders returns a slice of Orders that were removed, internally it will remove the orders from the
// matching engine price levels. The returned orders will have an Order_Expired status, ready to update in stores.
func (b *OrderBook) RemoveExpiredOrders(expirationTimestamp uint64) []*msg.Order {
	var expiredOrders []*msg.Order
	var pendingOrders []*msg.Order

	// linear scan of our expiring orders, prune any that have expired
	for _, or := range b.expiringOrders {
		if or.ExpirationTimestamp <= expirationTimestamp {
			b.RemoveOrder(or)                              // order is removed from the book
			or.Status = msg.Order_Expired                  // order is marked as expired for storage
			expiredOrders = append(expiredOrders, or)
		} else {
			pendingOrders = append(pendingOrders, or)      // order is pending expiry (future)
		}
	}

	log.Debugf("Matching: Removed %d orders that expired, %d remaining on book", len(expiredOrders), len(pendingOrders))
	
	// update our list of GTT orders pending expiry, ready for next run.
	b.expiringOrders = nil
	b.expiringOrders = pendingOrders
	return expiredOrders
}

func (b OrderBook) getSide(orderSide msg.Side) *OrderBookSide {
	if orderSide == msg.Side_Buy {
		return b.buy
	} else {
		return b.sell
	}
}

func (b *OrderBook) getOppositeSide(orderSide msg.Side) *OrderBookSide {
	if orderSide == msg.Side_Buy {
		return b.sell
	} else {
		return b.buy
	}
}

func makeResponse(order *msg.Order, trades []*msg.Trade, impactedOrders []*msg.Order) *msg.OrderConfirmation {
	confirm := msg.OrderConfirmationPool.Get().(*msg.OrderConfirmation)
	confirm.Order = order
	confirm.PassiveOrdersAffected = impactedOrders
	confirm.Trades = trades
	return confirm
}

func (b *OrderBook) PrintState(msg string) {
	log.Debugf("\n%s\n", msg)
	log.Debugf("------------------------------------------------------------\n")
	log.Debugf("                        BUY SIDE                            \n")
	for _, priceLevel := range b.buy.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print()
		}
	}
	log.Debugf("------------------------------------------------------------\n")
	log.Debugf("                        SELL SIDE                           \n")
	for _, priceLevel := range b.sell.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print()
		}
	}
	log.Debugf("------------------------------------------------------------\n")

}
