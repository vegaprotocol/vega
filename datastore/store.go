package datastore

import (
	"fmt"
	"vega/proto"
)

type TradeStore interface {
	// Get retrieves a trades for a given market.
	GetAll(market string, params GetParams) ([]*Trade, error)
	// Get retrieves a trade for a given id.
	Get(market string, id string) (*Trade, error)
	// GetByOrderId retrieves all trades for a given order id.
	GetByOrderId(market string, orderId string, params GetParams) ([]*Trade, error)
	// Post creates a new trade in the store.
	Post(r *Trade) error
	// Put updates an existing trade in the store
	Put(r *Trade) error
	// Removes a trade from the store.
	Delete(r *Trade) error
}

type OrderStore interface {
	// All retrieves all orders for a given market.
	GetAll(market string, params GetParams) ([]*Order, error)
	// Get retrieves an order for a given market and id.
	Get(market string, id string) (*Order, error)
	// GetByParty retrieves all orders for a given party name.
	//GetByParty(party string, params GetParams) ([]*Order, error)
	// Post creates a new order in the store.
	Post(r *Order) error
	// Put updates an existing order in the store.
	Put(r *Order) error
	// Removes an order from the store.
	Delete(r *Order) error
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
	o := &Order{}
	o = o.FromProtoMessage(orderMsg)

	switch msg.Order_Status(o.Status) {
	case msg.Order_NEW:
		// Audit new order via order audit log
	case msg.Order_FILLED:
		// Audit new order filled ^
	case msg.Order_ACTIVE:
		// Audit new order active ^
	case msg.Order_CANCELLED:
		// Audit new order cancelled ^
	}

	m.orderStore.Put(o)

	fmt.Printf("Added order of size %d, price %d", o.Size, o.Price)
	fmt.Println("---")
}

func (m *MemoryStoreProvider) listenForTrades() {
	for tradeMsg := range m.tradeChan {

		t := &Trade{}
		t = t.FromProtoMessage(tradeMsg, "")

		m.tradeStore.Put(t)

		fmt.Printf("Added trade of size %d, price %d", t.Size, t.Price)
		fmt.Println("---")

	}

}
