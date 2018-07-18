package datastore

import "vega/proto"

type TradeStore interface {
	// GetByMarket retrieves trades for a given market.
	GetByMarket(market string, params GetParams) ([]Trade, error)
	// Get retrieves a trade for a given id.
	GetByMarketAndId(market string, id string) (Trade, error)
	// GetByParty retrieves trades for a given party.
	GetByParty(party string, params GetParams) ([]Trade, error)
	// Get retrieves a trade for a given id.
	GetByPartyAndId(party string, id string) (Trade, error)
	// Post creates a new trade in the store.
	Post(r Trade) error
	// Put updates an existing trade in the store.
	Put(r Trade) error
	// Removes a trade from the store.
	Delete(r Trade) error
	// Aggregates trades into candles
	GetCandles(market string, sinceBlock, currentBlock, interval uint64) (msg.Candles, error)
}

type OrderStore interface {
	// GetByMarket retrieves all orders for a given market.
	GetByMarket(market string, params GetParams) ([]Order, error)
	// Get retrieves an order for a given market and id.
	GetByMarketAndId(market string, id string) (Order, error)
	// GetByParty retrieves trades for a given party.
	GetByParty(party string, params GetParams) ([]Order, error)
	// Get retrieves a trade for a given id.
	GetByPartyAndId(party string, id string) (Order, error)
	// Post creates a new order in the store.
	Post(r Order) error
	// Put updates an existing order in the store.
	Put(r Order) error
	// Removes an order from the store.
	Delete(r Order) error
	// Returns all the markets
	GetMarkets() ([]string, error)
}

type StoreProvider interface {
	Init(markets, parties []string)
	TradeStore() TradeStore
	OrderStore() OrderStore
}

type MemoryStoreProvider struct {
	memStore   MemStore
	tradeStore TradeStore
	orderStore OrderStore
}

func (m *MemoryStoreProvider) Init(markets, parties []string) {
	m.memStore = NewMemStore(markets, parties)
	m.tradeStore = NewTradeStore(&m.memStore)
	m.orderStore = NewOrderStore(&m.memStore)
}

func (m *MemoryStoreProvider) TradeStore() TradeStore {
	return m.tradeStore
}

func (m *MemoryStoreProvider) OrderStore() OrderStore {
	return m.orderStore
}