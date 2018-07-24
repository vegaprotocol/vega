package matching

import (
	"fmt"
	"vega/msg"

	"log"
)

type OrderBook struct {
	name            string
	buy             *OrderBookSide
	sell            *OrderBookSide
	lastTradedPrice uint64
	config          Config
	latestTimestamp uint64
}

// Create an order book with a given name
func NewBook(name string, config Config) *OrderBook {
	return &OrderBook{
		name:   name,
		buy:    &OrderBookSide{},
		sell:   &OrderBookSide{},
		config: config,
	}
}

// Cancel an order that is active on an orderbook. Market and Order ID are validated, however the order must match
// the order on the book with respect to side etc. The caller will typically validate this by using a store, we should
// not trust that the external world can provide these values reliably.
func (b *OrderBook) CancelOrder(order *msg.Order) (*msg.OrderCancellation, msg.OrderError) {
	// todo(cdm): an improvement here would be to add lookup validation in the matching engine, order pointer slice idx
	// Validate Market
	if order.Market != b.name {
		log.Println(fmt.Sprintf(
			"Market ID mismatch\norderMessage.Market: %v\nbook.ID: %v",
			order.Market,
			b.name))
		return nil, msg.OrderError_INVALID_MARKET_ID
	}
	// Validate Order ID must be present
	if order.Id == "" || len(order.Id) < 4 {
		return nil, msg.OrderError_INVALID_ORDER_ID
	}
	var err error
	if order.Side == msg.Side_Buy {
		err = b.buy.RemoveOrder(order)
	} else {
		err = b.sell.RemoveOrder(order)
	}
	if err != nil {
		// todo(cdm): structured logging here as this shouldnt happen often and we'd love to know if it does (MR!)
		return nil, msg.OrderError_ORDER_REMOVAL_FAILURE
	}

	// Important, mark the order as cancelled (and no longer active)
	order.Status = msg.Order_Cancelled

	result := &msg.OrderCancellation{
		Order: order,
	}
	return result, msg.OrderError_NONE
}

// Add an order and attempt to uncross the book, returns a TradeSet protobufs message object
func (b *OrderBook) AddOrder(order *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	if err := b.validateOrder(order); err != msg.OrderError_NONE {
		return nil, err
	}

	if order.Timestamp > b.latestTimestamp {
		b.latestTimestamp = order.Timestamp
	}

	b.PrintState("Entry state:")

	// uncross with opposite
	trades, impactedOrders, lastTradedPrice := b.getOppositeSide(order.Side).uncross(order)

	if lastTradedPrice != 0 {
		b.lastTradedPrice = lastTradedPrice
	}

	// if state of the book changed show state
	if len(trades) != 0 {
		b.PrintState("After uncross state:")
	}

	// if order is persistent type add to order book to the correct side
	if (order.Type == msg.Order_GTC || order.Type == msg.Order_GTT) && order.Remaining > 0 {
		b.getSide(order.Side).addOrder(order, order.Side)

		b.PrintState("After addOrder state:")
	}

	orderConfirmation := makeResponse(order, trades, impactedOrders)
	return orderConfirmation, msg.OrderError_NONE
}

func (b *OrderBook) RemoveOrder(order *msg.Order) error {
	err := b.getSide(order.Side).RemoveOrder(order)
	return err
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
	fmt.Println()
	fmt.Println(msg)
	fmt.Println("------------------------------------------------------------")
	fmt.Println("                        BUY SIDE                            ")
	for _, priceLevel := range b.buy.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print()
		}
	}
	fmt.Println("------------------------------------------------------------")
	fmt.Println("                        SELL SIDE                           ")
	for _, priceLevel := range b.sell.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print()
		}
	}
	fmt.Println("------------------------------------------------------------")

}
