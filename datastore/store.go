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

type CandleStore interface {

	Close()
	Subscribe(market string, iT *InternalTransport) uint64
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
