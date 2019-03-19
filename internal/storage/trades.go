package storage

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/internal/filtering"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type TradeStore interface {
	Subscribe(trades chan<- []types.Trade) uint64
	Unsubscribe(id uint64) error

	// Post adds a trade to the store, adds
	// to queue the operation to be committed later.
	Post(trade *types.Trade) error

	// Commit typically saves any operations that are queued to underlying storage,
	// if supported by underlying storage implementation.
	Commit() error

	// Close can be called to clean up and close any storage
	// connections held by the underlying storage mechanism.
	Close() error

	// GetByMarket retrieves trades for a given market.
	GetByMarket(ctx context.Context, market string, params *filtering.TradeQueryFilters) ([]*types.Trade, error)
	// GetByMarketAndId retrieves a trade for a given market and id.
	GetByMarketAndId(ctx context.Context, market string, id string) (*types.Trade, error)
	// GetByParty retrieves trades for a given party (buyer or seller).
	GetByParty(ctx context.Context, party string, params *filtering.TradeQueryFilters) ([]*types.Trade, error)
	// GetByPartyAndId retrieves a trade for a given party (buyer or seller) and id.
	GetByPartyAndId(ctx context.Context, party string, id string) (*types.Trade, error)
	// GetByOrderId retrieves trades relating to the given order id - buy order Id or sell order Id.
	GetByOrderId(ctx context.Context, orderId string, params *filtering.TradeQueryFilters) ([]*types.Trade, error)
	// GetMarkPrice returns the current market price.
	GetMarkPrice(ctx context.Context, market string) (uint64, error)

	// GetTradesBySideBuckets retrieves a map of market name to market buckets.
	GetTradesBySideBuckets(ctx context.Context, party string) map[string]*MarketBucket

	// GetLastTrade returns the last trade created on the vega network.
	GetLastTrade(ctx context.Context) *types.Trade
}

// badgerTradeStore is a package internal data struct that implements the TradeStore interface.
type badgerTradeStore struct {
	*Config
	badger       *badgerStore
	subscribers  map[uint64]chan<- []types.Trade
	subscriberId uint64
	buffer       []types.Trade
	lastTrade    types.Trade
	mu           sync.Mutex
}

// NewTradeStore is used to initialise and create a TradeStore, this implementation is currently
// using the badger k-v persistent storage engine under the hood. The caller will specify a dir to
// use as the storage location on disk for any stored files via Config.
func NewTradeStore(c *Config) (TradeStore, error) {
	err := InitStoreDirectory(c.TradeStoreDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "error on init badger database for trades storage")
	}
	db, err := badger.Open(customBadgerOptions(c.TradeStoreDirPath, c.GetLogger()))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for trades storage")
	}
	bs := badgerStore{db: db}
	return &badgerTradeStore{
		Config:      c,
		badger:      &bs,
		buffer:      make([]types.Trade, 0),
		subscribers: make(map[uint64]chan<- []types.Trade),
	}, nil
}

// Subscribe to a channel of new or updated trades. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (ts *badgerTradeStore) Subscribe(trades chan<- []types.Trade) uint64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.subscriberId = ts.subscriberId + 1
	ts.subscribers[ts.subscriberId] = trades

	ts.log.Debug("Trades subscriber added in order store",
		logging.Uint64("subscriber-id", ts.subscriberId))

	return ts.subscriberId
}

// Unsubscribe from an trades channel. Provide the subscriber id you wish to stop receiving new events for.
func (ts *badgerTradeStore) Unsubscribe(id uint64) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if len(ts.subscribers) == 0 {
		ts.log.Debug("Un-subscribe called in trade store, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := ts.subscribers[id]; exists {
		delete(ts.subscribers, id)
		ts.log.Debug("Un-subscribe called in trade store, subscriber removed",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	return errors.New(fmt.Sprintf("Trades subscriber does not exist with id: %d", id))
}

// Post adds an trade to the badger store, adds
// to queue the operation to be committed later.
func (ts *badgerTradeStore) Post(trade *types.Trade) error {
	// with badger we always buffer for future batch insert via Commit()
	ts.addToBuffer(*trade)
	ts.lastTrade = *trade
	return nil
}

// Commit saves any operations that are queued to badger store, and includes all updates.
// It will also call notify() to push updated data to any subscribers.
func (ts *badgerTradeStore) Commit() error {
	if len(ts.buffer) == 0 {
		return nil
	}

	ts.mu.Lock()
	items := ts.buffer
	ts.buffer = make([]types.Trade, 0)
	ts.mu.Unlock()

	err := ts.writeBatch(items)
	if err != nil {
		return err
	}
	err = ts.notify(items)
	if err != nil {
		return err
	}
	return nil
}

// GetByMarket retrieves trades for a given market. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (ts *badgerTradeStore) GetByMarket(ctx context.Context, market string, queryFilters *filtering.TradeQueryFilters) ([]*types.Trade, error) {
	var result []*types.Trade

	if queryFilters == nil {
		queryFilters = &filtering.TradeQueryFilters{}
	}

	txn := ts.badger.readTransaction()
	defer txn.Discard()

	filter := TradeFilter{queryFilter: queryFilters}
	descending := filter.queryFilter.HasLast()
	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	marketPrefix, validForPrefix := ts.badger.marketPrefix(market, descending)
	for it.Seek(marketPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			item := it.Item()
			tradeBuf, _ := item.ValueCopy(nil)
			var trade types.Trade
			if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
				ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByMarket)",
					logging.Error(err),
					logging.String("badger-key", string(item.Key())),
					logging.String("raw-bytes", string(tradeBuf)))

				return nil, err
			}
			if filter.apply(&trade) {
				result = append(result, &trade)
			}
			if filter.isFull() {
				break
			}
		}
	}
	return result, nil
}

// GetByMarketAndId retrieves a trade for a given market and id, any errors will be returned immediately.
func (ts *badgerTradeStore) GetByMarketAndId(ctx context.Context, market string, Id string) (*types.Trade, error) {
	var trade types.Trade

	txn := ts.badger.readTransaction()
	defer txn.Discard()

	marketKey := ts.badger.tradeMarketKey(market, Id)
	item, err := txn.Get(marketKey)
	if err != nil {
		return nil, err
	}
	tradeBuf, _ := item.ValueCopy(nil)
	if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
		ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByMarketAndId)",
			logging.Error(err),
			logging.String("badger-key", string(item.Key())),
			logging.String("raw-bytes", string(tradeBuf)))

		return nil, err
	}
	return &trade, err
}

// GetByParty retrieves trades for a given party. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (ts *badgerTradeStore) GetByParty(ctx context.Context, party string, queryFilters *filtering.TradeQueryFilters) ([]*types.Trade, error) {
	var result []*types.Trade

	if queryFilters == nil {
		queryFilters = &filtering.TradeQueryFilters{}
	}

	txn := ts.badger.readTransaction()
	defer txn.Discard()

	filter := TradeFilter{queryFilter: queryFilters}
	descending := filter.queryFilter.HasLast()
	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	partyPrefix, validForPrefix := ts.badger.partyPrefix(party, descending)
	for it.Seek(partyPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			marketKeyItem := it.Item()
			marketKey, _ := marketKeyItem.ValueCopy(nil)
			tradeItem, err := txn.Get(marketKey)
			if err != nil {
				ts.log.Error("Trade with key does not exist in trade store (getByParty)",
					logging.String("badger-key", string(marketKey)),
					logging.Error(err))

				return nil, err
			}
			tradeBuf, _ := tradeItem.ValueCopy(nil)
			var trade types.Trade
			if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
				ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByParty)",
					logging.Error(err),
					logging.String("badger-key", string(marketKey)),
					logging.String("raw-bytes", string(tradeBuf)))

				return nil, err
			}
			if filter.apply(&trade) {
				result = append(result, &trade)
			}
			if filter.isFull() {
				break
			}
		}
	}

	return result, nil
}

// GetByPartyAndId retrieves a trade for a given party and id.
func (ts *badgerTradeStore) GetByPartyAndId(ctx context.Context, party string, Id string) (*types.Trade, error) {
	var trade types.Trade
	err := ts.badger.db.View(func(txn *badger.Txn) error {
		partyKey := ts.badger.tradePartyKey(party, Id)
		marketKeyItem, err := txn.Get(partyKey)
		if err != nil {
			return err
		}
		marketKey, err := marketKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		tradeItem, err := txn.Get(marketKey)
		if err != nil {
			return err
		}

		tradeBuf, err := tradeItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
			ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByPartyAndId)",
				logging.Error(err),
				logging.String("badger-key", string(marketKey)),
				logging.String("raw-bytes", string(tradeBuf)))

			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &trade, nil
}

// GetByOrderId retrieves trades relating to the given order id - buy order Id or sell order Id.
// Provide optional query filters to refine the data set further (if required), any errors will be returned immediately.
func (ts *badgerTradeStore) GetByOrderId(ctx context.Context, orderId string, queryFilters *filtering.TradeQueryFilters) ([]*types.Trade, error) {
	var result []*types.Trade

	txn := ts.badger.readTransaction()
	defer txn.Discard()

	filter := TradeFilter{queryFilter: queryFilters}
	descending := filter.queryFilter.HasLast()
	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	orderPrefix, validForPrefix := ts.badger.orderPrefix(orderId, descending)
	for it.Seek(orderPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			marketKeyItem := it.Item()
			marketKey, _ := marketKeyItem.ValueCopy(nil)
			tradeItem, err := txn.Get(marketKey)
			if err != nil {
				ts.log.Error("Trade with key does not exist in trade store (getByOrderId)",
					logging.String("badger-key", string(marketKey)),
					logging.Error(err))

				return nil, err
			}
			tradeBuf, _ := tradeItem.ValueCopy(nil)
			var trade types.Trade
			if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
				ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByOrderId)",
					logging.Error(err),
					logging.String("badger-key", string(marketKey)),
					logging.String("raw-bytes", string(tradeBuf)))

				return nil, err
			}
			if filter.apply(&trade) {
				result = append(result, &trade)
			}
			if filter.isFull() {
				break
			}
		}
	}

	return result, nil
}


// GetMarkPrice returns the current market price, for a requested market.
func (ts *badgerTradeStore) GetMarkPrice(ctx context.Context, market string) (uint64, error) {

	// We just need the very latest trade price
	f := &filtering.TradeQueryFilters{}
	l := uint64(1)
	f.Last = &l

	recentTrade, err := ts.GetByMarket(ctx, market, f)
	if err != nil {
		return 0, err
	}

	if len(recentTrade) == 0 {
		return 0, errors.New("no trades available when getting market price")
	}

	return recentTrade[0].Price, nil
}


// GetLastTrade returns the last trade created on the vega network.
func (ts *badgerTradeStore) GetLastTrade(ctx context.Context) *types.Trade {
	return &ts.lastTrade
}

// Close our connection to the badger database
// ensuring errors will be returned up the stack.
func (ts *badgerTradeStore) Close() error {
	return ts.badger.db.Close()
}

// add a trade to the write-batch/notify buffer.
func (ts *badgerTradeStore) addToBuffer(t types.Trade) {
	ts.mu.Lock()
	ts.buffer = append(ts.buffer, t)
	ts.mu.Unlock()
}

// notify any subscribers of trade updates.
func (ts *badgerTradeStore) notify(items []types.Trade) error {
	if len(items) == 0 {
		return nil
	}
	if len(ts.subscribers) == 0 {
		ts.log.Debug("No subscribers connected in trade store")
		return nil
	}

	var ok bool
	for id, sub := range ts.subscribers {
		select {
		case sub <- items:
			ok = true
			break
		default:
			ok = false
		}
		if ok {
			ts.log.Debug("Trades channel updated for subscriber successfully",
				logging.Uint64("id", id))
		} else {
			ts.log.Debug("Trades channel could not be updated for subscriber",
				logging.Uint64("id", id))
		}
	}
	return nil
}

func (ts *badgerTradeStore) tradeBatchToMap(batch []types.Trade) (map[string][]byte, error) {
	results := make(map[string][]byte)
	for _, trade := range batch {
		tradeBuf, err := proto.Marshal(&trade)
		if err != nil {
			return nil, err
		}
		// Market Index
		marketKey := ts.badger.tradeMarketKey(trade.Market, trade.Id)
		// Trade Id index
		idKey := ts.badger.tradeIdKey(trade.Id)
		// Party indexes (buyer and seller as parties)
		buyerPartyKey := ts.badger.tradePartyKey(trade.Buyer, trade.Id)
		sellerPartyKey := ts.badger.tradePartyKey(trade.Seller, trade.Id)
		// OrderId indexes (relate to both buy and sell orders)
		buyOrderKey := ts.badger.tradeOrderIdKey(trade.BuyOrder, trade.Id)
		sellOrderKey := ts.badger.tradeOrderIdKey(trade.SellOrder, trade.Id)

		results[string(marketKey)] = tradeBuf
		results[string(idKey)] = marketKey
		results[string(buyerPartyKey)] = marketKey
		results[string(sellerPartyKey)] = marketKey
		results[string(buyOrderKey)] = marketKey
		results[string(sellOrderKey)] = marketKey
	}
	return results, nil
}

// writeBatch flushes a batch of trades to the underlying badger store.
func (ts *badgerTradeStore) writeBatch(batch []types.Trade) error {
	kv, err := ts.tradeBatchToMap(batch)
	if err != nil {
		ts.log.Error("Failed to marshal trades before writing batch",
			logging.Error(err))
		return err
	}

	b, err := ts.badger.writeBatch(kv)
	if err != nil {
		if b == 0 {
			ts.log.Warn("Failed to insert trade batch; No records were committed, atomicity maintained",
				logging.Error(err))
			// TODO: Retry, in some circumstances.
		} else {
			ts.log.Error("Failed to insert trade batch; Some records were committed, atomicity lost",
				logging.Error(err))
			// TODO: Mark block dirty, panic node.
		}
		return err
	}

	return nil
}

// TradeFilter is the trade specific filter query data holder. It includes the raw filters
// and helper methods that are used internally to apply and track filter state.
type TradeFilter struct {
	queryFilter *filtering.TradeQueryFilters
	skipped     uint64
	found       uint64
}

func (f *TradeFilter) apply(trade *types.Trade) (include bool) {
	if f.queryFilter.First == nil && f.queryFilter.Last == nil && f.queryFilter.Skip == nil {
		include = true
	} else {
		if f.queryFilter.HasFirst() && f.found < *f.queryFilter.First {
			include = true
		}
		if f.queryFilter.HasLast() && f.found < *f.queryFilter.Last {
			include = true
		}
		if f.queryFilter.HasSkip() && f.skipped < *f.queryFilter.Skip {
			f.skipped++
			return false
		}
	}
	if !applyTradeFilters(trade, f.queryFilter) {
		return false
	}

	// if item passes the filter, increment the found counter
	if include {
		f.found++
	}
	return include
}

func (f *TradeFilter) isFull() bool {
	if f.queryFilter.HasLast() && f.found == *f.queryFilter.Last {
		return true
	}
	if f.queryFilter.HasFirst() && f.found == *f.queryFilter.First {
		return true
	}
	return false
}
