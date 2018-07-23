package matching

import (
	"vega/log"
	"vega/msg"
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
	log.Infof("\n%s\n", msg)
	log.Infof("------------------------------------------------------------\n")
	log.Infof("                        BUY SIDE                            \n")
	for _, priceLevel := range b.buy.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print()
		}
	}
	log.Infof("------------------------------------------------------------\n")
	log.Infof("                        SELL SIDE                           \n")
	for _, priceLevel := range b.sell.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print()
		}
	}
	log.Infof("------------------------------------------------------------\n")

}
