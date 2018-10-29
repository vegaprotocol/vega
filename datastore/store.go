package datastore

import (
	"vega/msg"
	"vega/filters"
)

type TradeStore interface {

	Subscribe(trades chan<- []msg.Trade) uint64
	Unsubscribe(id uint64) error
	Notify() error

	// GetByMarket retrieves trades for a given market.
	GetByMarket(market string, params *filters.TradeQueryFilters) ([]msg.Trade, error)
	// Get retrieves a trade for a given id.
	GetByMarketAndId(market string, id string) (msg.Trade, error)
	// GetByParty retrieves trades for a given party.
	GetByParty(party string, params *filters.TradeQueryFilters) ([]msg.Trade, error)
	// Get retrieves a trade for a given id.
	GetByPartyAndId(party string, id string) (msg.Trade, error)
	// Post creates a new trade in the store.
	Post(r msg.Trade) error
	// Put updates an existing trade in the store.
	Put(r msg.Trade) error
	// Removes a trade from the store.
	Delete(r msg.Trade) error
	// Aggregates trades into candles
	GetCandles(market string, sinceBlock, currentBlock, interval uint64) (msg.Candles, error)
	// Aggregate trades into a single candle from currentBlock for interval
	GetCandle(market string, sinceBlock, currentBlock uint64) (*msg.Candle, error)

	// Returns current market price
	GetMarkPrice(market string) (uint64, error)
	// Returns map of market name to market buckets
	GetTradesBySideBuckets(party string) map[string]*MarketBucket
}

type OrderStore interface {

	Subscribe(orders chan<- []msg.Order) uint64
	Unsubscribe(id uint64) error
	Notify() error

	Close()

	// GetByMarket retrieves all orders for a given market.
	GetByMarket(market string, filters *filters.OrderQueryFilters) ([]*msg.Order, error)
	// Get retrieves an order for a given market and id.
	GetByMarketAndId(market string, id string) (*msg.Order, error)
	// GetByParty retrieves trades for a given party.
	GetByParty(party string, filters *filters.OrderQueryFilters) ([]*msg.Order, error)
	// Get retrieves a trade for a given id.
	GetByPartyAndId(party string, id string) (*msg.Order, error)
	// Post creates a new order in the store.
	Post(r *msg.Order) error
	PostBatch(batch []*msg.Order) error
	// Put updates an existing order in the store.
	Put(r *msg.Order) error
	// Removes an order from the store.
	Delete(r *msg.Order) error
	// Returns Order Book Depth for a market
	GetMarketDepth(market string) (*msg.MarketDepth, error)
	// Returns Order by reference number
	GetByPartyAndReference(party string, reference string) (*msg.Order, error)
}

type PartyStore interface {
	Post(party string) error
	Put(party string) error
	Delete(party string) error
	GetAllParties() ([]string, error)
}

//type StoreProvider interface {
//	Init(markets, parties []string)
//	TradeStore() TradeStore
//	OrderStore() OrderStore
//	PartyStore() PartyStore
//}
//
//type MemoryStoreProvider struct {
//	memStore   MemStore
//	tradeStore TradeStore
//	orderStore OrderStore
//	partyStore PartyStore
//}

//func (m *MemoryStoreProvider) Init(markets, parties []string) {
//	m.memStore = NewMemStore(markets, parties)
//	m.tradeStore = NewTradeStore(&m.memStore)
//	m.orderStore = NewOrderStore(&m.memStore)
//	m.partyStore = NewPartyStore(&m.memStore)
//}

//func (m *MemoryStoreProvider) TradeStore() TradeStore {
//	return m.tradeStore
//}
//
//func (m *MemoryStoreProvider) OrderStore() OrderStore {
//	return m.orderStore
//}
//
//func (m *MemoryStoreProvider) PartyStore() PartyStore {
//	return m.partyStore
//}