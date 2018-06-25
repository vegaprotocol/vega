package matching

import (
	"fmt"
	"vega/proto"
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
	book := &OrderBook{
		name:   name,
		config: config,
	}
	book.buy = newSide(msg.Side_Buy)
	book.sell = newSide(msg.Side_Sell)
	return book
}

// Add an order and attempt to uncross the book, returns a TradeSet protobufs message object
func (b *OrderBook) AddOrder(orderMessage *msg.Order) (*msg.OrderConfirmation, msg.OrderError) {
	if err := b.validateOrder(orderMessage); err != msg.OrderError_NONE {
		return nil, err
	}
	if orderMessage.Timestamp > b.latestTimestamp {
		b.latestTimestamp = orderMessage.Timestamp
	}

	o := &OrderEntry{
		Side: orderMessage.Side,
		order:   orderMessage,
		persist: orderMessage.Type == msg.Order_GTC || orderMessage.Type == msg.Order_GTT,
		dispatchChannels: b.config.OrderChans,
	}
	o.order.Id = o.Digest()

	// uncross with opposite
	trades, lastTradedPrice := b.getOppositeSide(orderMessage.Side).cross(o)
	if lastTradedPrice != 0 {
		b.lastTradedPrice = lastTradedPrice
	}

	for _, t := range *trades {
		for _, c := range b.config.TradeChans {
			c <- *t.toMessage()
		}
	}

	// if persist add to tradebook to the right side
	if o.persist && o.order.Remaining > 0 {
		b.getSide(orderMessage.Side).addOrder(o)
	}

	orderConfirmation := MakeResponse(orderMessage, trades)
	printSlice(*trades)
	if len(*trades) == 0 {
		for _, c := range b.config.OrderChans {
			c <- *orderMessage
		}
	}
	return orderConfirmation, msg.OrderError_NONE
}

func printSlice(s []Trade) {
	fmt.Printf("len=%d cap=%d\n", len(s), cap(s))
}

func (b *OrderBook) getSide(orderSide msg.Side) *OrderBookSide {
	if orderSide == msg.Side_Buy {
		return b.buy
	} else { // side == Sell
		return b.sell
	}
}

func (b *OrderBook) getOppositeSide(orderSide msg.Side) *OrderBookSide {
	if orderSide == msg.Side_Buy {
		return b.sell
	} else { // side == Sell
		return b.buy
	}
}

func (b *OrderBook) GetName() string {
	return b.name
}

func (b *OrderBook) GetMarketData() *msg.MarketData {
	return &msg.MarketData{
		BestBid:         b.buy.bestPrice(),
		BestOffer:       b.sell.bestPrice(),
		LastTradedPrice: b.lastTradedPrice,
	}
}

func (b *OrderBook) GetMarketDepth() *msg.MarketDepth {
	return &msg.MarketDepth{
		BuyOrderCount:   b.buy.getOrderCount(),
		SellOrderCount:  b.sell.getOrderCount(),
		BuyOrderVolume:  b.buy.getTotalVolume(),
		SellOrderVolume: b.sell.getTotalVolume(),
		BuyPriceLevels:  uint64(b.buy.getNumberOfPriceLevels()),
		SellPriceLevels: uint64(b.sell.getNumberOfPriceLevels()),
	}
}

func (b *OrderBook) RemoveOrder(orderEntry *OrderEntry) {
	b.getSide(orderEntry.Side).RemoveOrder(orderEntry)
}
