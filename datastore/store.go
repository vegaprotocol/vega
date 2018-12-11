package datastore

import (
	"vega/msg"
	"vega/filters"
)

type TradeStore interface {
	// Close database
	Close()
	Subscribe(trades chan<- []msg.Trade) uint64
	Unsubscribe(id uint64) error
	// Notifies all subscribers with buffer content
	Notify(items []msg.Trade) error
	// Makes copy of internal buffer and calls Notify and PostBatch, cleans internal buffer
	Commit() error
	// Post adds trade to the internal buffer
	Post(trade *msg.Trade) error
	// PostBatch inserts all trades from the batch to database
	PostBatch(batch []msg.Trade) error
	// Removes a trade from the store.
	Delete(trade *msg.Trade) error


	// GetByMarket retrieves trades for a given Market.
	GetByMarket(market string, params *filters.TradeQueryFilters) ([]*msg.Trade, error)
	// Get retrieves a trade for a given id.
	GetByMarketAndId(market string, id string) (*msg.Trade, error)
	// GetByParty retrieves trades for a given party.
	GetByParty(party string, params *filters.TradeQueryFilters) ([]*msg.Trade, error)
	// Get retrieves a trade for a given id.
	GetByPartyAndId(party string, id string) (*msg.Trade, error)
	// Returns current Market price
	GetMarkPrice(market string) (uint64, error)
	// Returns map of Market name to Market buckets
	GetTradesBySideBuckets(party string) map[string]*MarketBucket

	// Trades relating to the given orderId for a particular market
	GetByMarketAndOrderId(market string, orderId string) ([]*msg.Trade, error)
}

type OrderStore interface {
	// Close database
	Close()
	Subscribe(orders chan<- []msg.Order) uint64
	Unsubscribe(id uint64) error
	// Notifies all subscribers with buffer content
	Notify(items []msg.Order) error
	// Makes copy of internal buffer and calls Notify and PostBatch, cleans internal buffer
	Commit() error
	// Post adds trade to the internal buffer
	Post(order *msg.Order) error
	// PostBatch inserts all trades from the batch to database
	PostBatch(batch []msg.Order) error
	// Put updates an existing order in the store, either it is in buffer in database
	Put(order *msg.Order) error
	// Removes an order from the store.
	Delete(order *msg.Order) error


	// GetByMarket retrieves all orders for a given Market.
	GetByMarket(market string, filters *filters.OrderQueryFilters) ([]*msg.Order, error)
	// Get retrieves an order for a given Market and id.
	GetByMarketAndId(market string, id string) (*msg.Order, error)
	// GetByParty retrieves trades for a given party.
	GetByParty(party string, filters *filters.OrderQueryFilters) ([]*msg.Order, error)
	// Get retrieves a trade for a given id.
	GetByPartyAndId(party string, id string) (*msg.Order, error)
	// Returns Order Book Depth for a market
	GetMarketDepth(market string) (*msg.MarketDepth, error)
}

type CandleStore interface {

	Close()
	Subscribe(iT *InternalTransport) uint64
	Unsubscribe(id uint64) error
	Notify() error
	//QueueEvent(candle msg.Candle, interval msg.Interval) error

	StartNewBuffer(market string, timestamp uint64)
	AddTradeToBuffer(market string, trade msg.Trade) error
	GenerateCandlesFromBuffer(market string) error

	GetCandles(market string, sinceTimestamp uint64, interval msg.Interval) []*msg.Candle
}


//type PartyStore interface {
//	Post(party string) error
//	Put(party string) error
//	Delete(party string) error
//	GetAllParties() ([]string, error)
//}
