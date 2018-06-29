package datastore

import (
	"fmt"
	"context"
	"vega/proto"
)

type TradeStore interface {
	// Get retrieves a trades for a given market.
	All(ctx context.Context, market string) ([]*Trade, error)
	// Get retrieves a trade for a given id.
	Get(ctx context.Context, market string, id string) (*Trade, error)
	// GetByOrderID retrieves all trades for a given order id.
	GetByOrderID(ctx context.Context, market string, orderID string) ([]*Trade, error)
	// Put stores a trade.
	Put(ctx context.Context, r *Trade) error
	// Removes a trade from the store.
	Delete(ctx context.Context, r *Trade) error
}

type OrderStore interface {
	// All retrieves all orders for a given market
	All(ctx context.Context, market string) ([]*Order, error)
	// Get retrieves an order for a given market and id.
	Get(ctx context.Context, market string, id string) (*Order, error)
	// FindByParty retrieves all order for a given party name.
	//FindByParty(party string) ([]*Order, error)
	// Put stores a trade.
	Put(ctx context.Context, r *Order) error
	// Removes a trade from the store.
	Delete(ctx context.Context, r *Order) error
}

type StorageProvider interface {
	Init (markets []string, orderChan <-chan msg.Order, tradeChan <-chan msg.Trade)
	TradeStore() TradeStore
	OrderStore() OrderStore
}

type MemoryStorageProvider struct {
	memStore MemStore
	tradeStore TradeStore
	orderStore OrderStore
	tradeChan <-chan msg.Trade
	orderChan <-chan msg.Order
}

func (m *MemoryStorageProvider) Init (markets []string, orderChan <-chan msg.Order, tradeChan <-chan msg.Trade) {
	m.memStore = NewMemStore(markets)
	m.tradeStore = NewTradeStore(&m.memStore)
	m.orderStore = NewOrderStore(&m.memStore)
	m.tradeChan = tradeChan
	m.orderChan = orderChan

	go m.listenForOrders()
	go m.listenForTrades()
}

func (m *MemoryStorageProvider) TradeStore() TradeStore {
	return m.tradeStore
}

func (m *MemoryStorageProvider) OrderStore() OrderStore {
	return m.orderStore
}

func (m *MemoryStorageProvider) listenForOrders() {
	for orderMsg := range m.orderChan {
		m.processOrderMessage(orderMsg)
	}
}

// processOrderMessage takes an incoming order msg protobuf and logs/updates the stores.
func (m *MemoryStorageProvider) processOrderMessage(orderMsg msg.Order) {
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

	// todo how to pass context in order chans?
	ctx := context.Background()
	m.orderStore.Put(ctx, o)

	fmt.Printf("Added order of size %d, price %d", o.Size, o.Price)
	fmt.Println("---")
}



func (m *MemoryStorageProvider) listenForTrades() {
	for tradeMsg := range m.tradeChan {

		t := &Trade{}
		t = t.FromProtoMessage(tradeMsg, "")

		// todo how to pass context in order chans?
		ctx := context.Background()
		m.tradeStore.Put(ctx, t)

		fmt.Printf("Added trade of size %d, price %d", t.Size, t.Price)
		fmt.Println("---")

	}

}


