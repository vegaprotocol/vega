package matching

import (
	"fmt"
	"vega/api/endpoints/sse"
	"vega/proto"
)

type OrderBook struct {
	name            string
	buy             *OrderBookSide
	sell            *OrderBookSide
	lastTradedPrice uint64
	orders          map[string]*OrderEntry
	config          Config
	latestTimestamp uint64
	sse             sse.Server
}

// Create an order book with a given name
func NewBook(name string, orderLookup map[string]*OrderEntry, config Config) *OrderBook {
	book := &OrderBook{
		name:   name,
		orders: orderLookup,
		config: config,
	}
	buy, sell := makeSide(msg.Side_Buy, book), makeSide(msg.Side_Sell, book)
	book.buy = buy
	book.buy.other = sell
	book.sell = sell
	book.sell.other = buy
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
	orderEntry := orderEntryFromMessage(orderMessage)
	trades := b.sideFor(orderMessage).addOrder(orderEntry)
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

func (b *OrderBook) sideFor(orderMessage *msg.Order) *OrderBookSide {
	if orderMessage.Side == msg.Side_Buy {
		return b.buy
	} else { // side == Sell
		return b.sell
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

func (b *OrderBook) RemoveOrder(id string) *msg.Order {
	if order, exists := b.orders[id]; exists {
		return order.remove().order
	} else {
		return nil
	}
}

//
//func (b *OrderBook) GetBook() (buy, sell []*OrderEntry) {
//
//}
