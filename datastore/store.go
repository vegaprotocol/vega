package datastore

import (
	"vega/msg"
	"vega/filters"
)

type TradeStore interface {

	Subscribe(trades chan<- []msg.Trade) uint64
	Unsubscribe(id uint64) error
	Notify() error

	Close()

	// GetByMarket retrieves trades for a given market.
	GetByMarket(market string, params *filters.TradeQueryFilters) ([]*msg.Trade, error)
	// Get retrieves a trade for a given id.
	GetByMarketAndId(market string, id string) (*msg.Trade, error)
	// GetByParty retrieves trades for a given party.
	GetByParty(party string, params *filters.TradeQueryFilters) ([]*msg.Trade, error)
	// Get retrieves a trade for a given id.
	GetByPartyAndId(party string, id string) (*msg.Trade, error)
	// Post creates a new trade in the store.
	Post(trade *msg.Trade) error
	// Removes a trade from the store.
	Delete(trade *msg.Trade) error
	// Aggregates trades into candles
	GetCandles(market string, sinceBlock, currentBlock, interval uint64) ([]*msg.Candle, error)
	// Aggregate trades into a single candle from currentBlock for interval
	GetCandle(market string, sinceBlock, currentBlock uint64) (*msg.Candle, error)
	// Returns current market price
	GetMarkPrice(market string) (uint64, error)
	// Returns map of market name to market buckets
	GetTradesBySideBuckets(party string) map[string]*MarketBucket

	// Trades relating to the given orderId for a particular market
	//GetByMarketAndOrderId(market string, orderId string) ([]Trade, error)
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
	Post(order *msg.Order) error
	PostBatch(batch []*msg.Order) error
	// Put updates an existing order in the store.
	Put(order *msg.Order) error
	// Removes an order from the store.
	Delete(order *msg.Order) error
	// Returns Order Book Depth for a market
	GetMarketDepth(market string) (*msg.MarketDepth, error)
}

type CandleStore interface {

	Subscribe(internalTransport map[msg.Interval]chan msg.Candle) uint64
	Unsubscribe(id uint64) error
	Notify() error
	QueueEvent(candle msg.Candle, interval msg.Interval) error

	Close()

	GetCandles(market string, sinceTimestamp uint64, interval msg.Interval) []*msg.Candle
	GenerateCandles(trade *msg.Trade) error
	GenerateEmptyCandles(market string, timestamp uint64) error
}


//type PartyStore interface {
//	Post(party string) error
//	Put(party string) error
//	Delete(party string) error
//	GetAllParties() ([]string, error)
//}
