package market

import (
	"errors"
	"fmt"

	"proto"
)

type OrderBook struct {
	name            string
	buy             *Side
	sell            *Side
	lastTradedPrice uint64
	orders          map[string]*OrderEntry
}

type AddOrderResult struct {
	OrderId string
	Trades *pb.TradeSet
}

func NewAddOrderResult(orderId string, trades *[]Trade) *AddOrderResult {
	tradeSet := make([]*pb.Trade, len(*trades))
	for _, t := range *trades {
		tradeSet = append(tradeSet, t.toMessage())
	}
	return &AddOrderResult{
		OrderId: orderId,
		Trades: &pb.TradeSet{Trades: tradeSet},
	}
}

// Create an order book with a given name
func NewBook(name string) *OrderBook {
	book := &OrderBook{name: name, orders: make(map[string]*OrderEntry)}
	buy, sell := makeSide(pb.Side_Buy, book), makeSide(pb.Side_Sell, book)
	book.buy = buy
	book.buy.other = sell
	book.sell = sell
	book.sell.other = buy
	return book
}

// Add an order and attempt to uncross the book, returns a TradeSet protobufs message object
func (b *OrderBook) AddOrder(order *pb.Order) (*AddOrderResult, error) {

	// Check order is for the correct market
	if order.Market != b.name {
		return nil, errors.New(fmt.Sprintf(
			"Market ID mismatch\norder.Market: %v\nbook.ID: %v",
			order.Market,
			b.name))
	}

	orderEntry := b.fromMessage(order)
	var trades *[]Trade
	if order.Side == pb.Side_Buy {
		trades = b.buy.addOrder(orderEntry)
	} else { // side == Sell
		trades = b.sell.addOrder(orderEntry)
	}

	return NewAddOrderResult(orderEntry.id, trades), nil
}

func (b *OrderBook) GetName() string {
	return b.name
}

func (b *OrderBook) GetMarketData() *pb.MarketData {
	return &pb.MarketData{
		BestBid:         b.buy.bestPrice(),
		BestOffer:       b.sell.bestPrice(),
		LastTradedPrice: b.lastTradedPrice,
	}
}

func (b *OrderBook) RemoveOrder(id string) bool {
	if order, exists := b.orders[id]; exists {
		return order.remove()
	} else {
		return false
	}
}

//
//func (b *OrderBook) GetBook() (buy, sell []*OrderEntry) {
//
//}
