package datastore

import (
	"fmt"
	"vega/proto"
)

type TradeStore interface {
	// GetAll retrieves a trades for a given market.
	// If market == "" it will return trades for all markets in the store.
	// If party == "" it will return trades for all parties.
	GetAll(market string, params GetParams) ([]Trade, error)
	// Get retrieves a trade for a given id.
	Get(market string, id string) (Trade, error)
	// GetByOrderId retrieves all trades for a given order id.
	GetByOrderId(market string, orderId string, params GetParams) ([]Trade, error)
	// Post creates a new trade in the store.
	Post(r Trade) error
	// Put updates an existing trade in the store.
	Put(r Trade) error
	// Removes a trade from the store.
	Delete(r Trade) error
}

type OrderStore interface {
	// GetAll retrieves all orders for a given market.
	// If market == "" it will return orders for all markets in the store.
	// If party == "" it will return orders for all parties.
	GetAll(market string, party string, params GetParams) ([]Order, error)
	// Get retrieves an order for a given market and id.
	Get(market string, id string) (Order, error)
	// Post creates a new order in the store.
	Post(r Order) error
	// Put updates an existing order in the store.
	Put(r Order) error
	// Removes an order from the store.
	Delete(r Order) error
}

type StoreProvider interface {
	Init(markets []string, orderChan <-chan msg.Order, tradeChan <-chan msg.Trade)
	TradeStore() TradeStore
	OrderStore() OrderStore
}

type MemoryStoreProvider struct {
	memStore   MemStore
	tradeStore TradeStore
	orderStore OrderStore
	tradeChan  <-chan msg.Trade
	orderChan  <-chan msg.Order
}

func (m *MemoryStoreProvider) Init(markets []string, orderChan <-chan msg.Order, tradeChan <-chan msg.Trade) {
	m.memStore = NewMemStore(markets)
	m.tradeStore = NewTradeStore(&m.memStore)
	m.orderStore = NewOrderStore(&m.memStore)
	m.tradeChan = tradeChan
	m.orderChan = orderChan

	go m.listenForOrders()
	go m.listenForTrades()
}

func (m *MemoryStoreProvider) TradeStore() TradeStore {
	return m.tradeStore
}

func (m *MemoryStoreProvider) OrderStore() OrderStore {
	return m.orderStore
}

func (m *MemoryStoreProvider) listenForOrders() {
	for orderMsg := range m.orderChan {
		m.processOrderMessage(orderMsg)
	}
}

// processOrderMessage takes an incoming order msg protobuf and logs/updates the stores.
func (m *MemoryStoreProvider) processOrderMessage(orderMsg msg.Order) {
	o := NewOrderFromProtoMessage(orderMsg)

	m.orderStore.Put(*o)

	fmt.Printf("Added order of size %d, price %d", o.Size, o.Price)
	fmt.Println("---")
}

func (m *MemoryStoreProvider) listenForTrades() {
	for tradeMsg := range m.tradeChan {

		t := NewTradeFromProtoMessage(tradeMsg, "")

		m.tradeStore.Put(*t)

		fmt.Printf("Added trade of size %d, price %d", t.Size, t.Price)
		fmt.Println("---")

	}

}
