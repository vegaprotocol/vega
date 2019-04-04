package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// Trade is a package internal data struct that implements the TradeStore interface.
type Trade struct {
	*Config
	badger          *badgerStore
	subscribers     map[uint64]chan<- []types.Trade
	subscriberId    uint64
	buffer          []types.Trade
	mu              sync.Mutex
	onCriticalError func()
}

// NewTrades is used to initialise and create a TradeStore, this implementation is currently
// using the badger k-v persistent storage engine under the hood. The caller will specify a dir to
// use as the storage location on disk for any stored files via Config.
func NewTrades(c *Config, onCriticalError func()) (*Trade, error) {
	err := InitStoreDirectory(c.TradeStoreDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "error on init badger database for trades storage")
	}
	db, err := badger.Open(customBadgerOptions(c.TradeStoreDirPath, c.GetLogger()))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for trades storage")
	}
	bs := badgerStore{db: db}
	return &Trade{
		Config:          c,
		badger:          &bs,
		buffer:          make([]types.Trade, 0),
		subscribers:     make(map[uint64]chan<- []types.Trade),
		onCriticalError: onCriticalError,
	}, nil
}

// Subscribe to a channel of new or updated trades. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (ts *Trade) Subscribe(trades chan<- []types.Trade) uint64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.subscriberId = ts.subscriberId + 1
	ts.subscribers[ts.subscriberId] = trades

	ts.log.Debug("Trades subscriber added in order store",
		logging.Uint64("subscriber-id", ts.subscriberId))

	return ts.subscriberId
}

// Unsubscribe from an trades channel. Provide the subscriber id you wish to stop receiving new events for.
func (ts *Trade) Unsubscribe(id uint64) error {
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
func (ts *Trade) Post(trade *types.Trade) error {
	// with badger we always buffer for future batch insert via Commit()
	ts.addToBuffer(*trade)
	return nil
}

// Commit saves any operations that are queued to badger store, and includes all updates.
// It will also call notify() to push updated data to any subscribers.
func (ts *Trade) Commit() error {
	if len(ts.buffer) == 0 {
		return nil
	}

	ts.mu.Lock()
	items := ts.buffer
	ts.buffer = []types.Trade{}
	ts.mu.Unlock()

	err := ts.writeBatch(items)
	if err != nil {
		ts.log.Error(
			"badger store error on write",
			logging.Error(err),
		)
		ts.onCriticalError()
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
func (ts *Trade) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) ([]*types.Trade, error) {
	// get results cap
	var (
		err error
	)
	result := make([]*types.Trade, 0, int(limit))

	txn := ts.badger.readTransaction()
	defer txn.Discard()
	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	ctx, cancel := context.WithTimeout(ctx, ts.Config.Timeout*time.Second)
	defer cancel()
	marketPrefix, validForPrefix := ts.badger.marketPrefix(market, descending)
	tradeBuf := []byte{}
	for it.Seek(marketPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if tradeBuf, err = it.Item().ValueCopy(tradeBuf); err != nil {
				// @TODO log this error
				return nil, err
			}
			var trade types.Trade
			if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
				ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByMarket)",
					logging.Error(err),
					logging.String("badger-key", string(it.Item().Key())),
					logging.String("raw-bytes", string(tradeBuf)))

				return nil, err
			}
			if skip != 0 {
				skip--
				continue
			}
			result = append(result, &trade)
			if limit != 0 && len(result) == cap(result) {
				return result, nil
			}
		}
	}
	return result, nil
}

// GetByMarketAndId retrieves a trade for a given market and id, any errors will be returned immediately.
func (ts *Trade) GetByMarketAndId(ctx context.Context, market string, Id string) (*types.Trade, error) {
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
func (ts *Trade) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, market *string) ([]*types.Trade, error) {
	// get results cap
	var (
		err error
	)
	tmk, kLen := ts.getTradeMarketFilter(market)
	result := make([]*types.Trade, 0, int(limit))

	txn := ts.badger.readTransaction()
	defer txn.Discard()

	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	ctx, cancel := context.WithTimeout(ctx, ts.Config.Timeout*time.Second)
	defer cancel()
	// reuse these buffers, slices will get reallocated, so if the buffer is big enough
	// next calls won't alloc memory again
	marketKey, tradeBuf := []byte{}, []byte{}
	partyPrefix, validForPrefix := ts.badger.partyPrefix(party, descending)
	for it.Seek(partyPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// these errors should be logged, means the data is being stored inconsistently
			if marketKey, err = it.Item().ValueCopy(marketKey); err != nil {
				return nil, err
			}
			// we are filtering by market, but the market key doesn't match, stop here, don't waste time reading and unmarshalling the full trade item
			if kLen != 0 && string(marketKey[:kLen]) != string(tmk) {
				continue
			}
			tradeItem, err := txn.Get(marketKey)
			if err != nil {
				ts.log.Error("Trade with key does not exist in trade store (getByParty)",
					logging.String("badger-key", string(marketKey)),
					logging.Error(err))

				return nil, err
			}
			// these errors should be logged, means the data is being stored inconsistently
			if tradeBuf, err = tradeItem.ValueCopy(tradeBuf); err != nil {
				return nil, err
			}
			var trade types.Trade
			if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
				ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByParty)",
					logging.Error(err),
					logging.String("badger-key", string(marketKey)),
					logging.String("raw-bytes", string(tradeBuf)))

				return nil, err
			}
			// skip matches if needed
			if skip != 0 {
				skip--
				continue
			}
			result = append(result, &trade)
			if limit != 0 && len(result) == cap(result) {
				return result, nil
			}
		}
	}
	return result, nil
}

// GetByPartyAndId retrieves a trade for a given party and id.
func (ts *Trade) GetByPartyAndId(ctx context.Context, party string, Id string) (*types.Trade, error) {
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
func (ts *Trade) GetByOrderId(ctx context.Context, orderID string, skip, limit uint64, descending bool, market *string) ([]*types.Trade, error) {
	var (
		err error
	)
	tmk, kLen := ts.getTradeMarketFilter(market)
	result := make([]*types.Trade, 0, int(limit))
	txn := ts.badger.readTransaction()
	defer txn.Discard()

	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	orderPrefix, validForPrefix := ts.badger.orderPrefix(orderID, descending)

	ctx, cancel := context.WithTimeout(ctx, ts.Config.Timeout*time.Second)
	defer cancel()
	marketKey, tradeBuf := []byte{}, []byte{}
	for it.Seek(orderPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			if marketKey, err = it.Item().ValueCopy(marketKey); err != nil {
				return nil, err
			}
			// apply market filter here, avoid getting the trade item + unmarshalling
			if kLen != 0 && string(marketKey[:kLen]) != string(tmk) {
				continue
			}
			tradeItem, err := txn.Get(marketKey)
			if err != nil {
				ts.log.Error("Trade with key does not exist in trade store (getByOrderId)",
					logging.String("badger-key", string(marketKey)),
					logging.Error(err))

				return nil, err
			}
			if tradeBuf, err = tradeItem.ValueCopy(tradeBuf); err != nil {
				return nil, err
			}
			var trade types.Trade
			if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
				ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByOrderId)",
					logging.Error(err),
					logging.String("badger-key", string(marketKey)),
					logging.String("raw-bytes", string(tradeBuf)))

				return nil, err
			}
			if skip != 0 {
				skip--
				continue
			}
			result = append(result, &trade)
			if limit != 0 && len(result) == int(limit) {
				return result, nil
			}
		}
	}
	return result, nil
}

// GetMarkPrice returns the current market price, for a requested market.
func (ts *Trade) GetMarkPrice(ctx context.Context, market string) (uint64, error) {
	recentTrade, err := ts.GetByMarket(ctx, market, 0, 1, true)
	if err != nil {
		return 0, err
	}

	if len(recentTrade) == 0 {
		return 0, errors.New("no trades available when getting market price")
	}

	return recentTrade[0].Price, nil
}

// Close our connection to the badger database
// ensuring errors will be returned up the stack.
func (ts *Trade) Close() error {
	return ts.badger.db.Close()
}

// add a trade to the write-batch/notify buffer.
func (ts *Trade) addToBuffer(t types.Trade) {
	ts.mu.Lock()
	ts.buffer = append(ts.buffer, t)
	ts.mu.Unlock()
}

// notify any subscribers of trade updates.
func (ts *Trade) notify(items []types.Trade) error {
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

func (ts *Trade) tradeBatchToMap(batch []types.Trade) (map[string][]byte, error) {
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
func (ts *Trade) writeBatch(batch []types.Trade) error {
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

func (ts *Trade) getTradeMarketFilter(market *string) ([]byte, int) {
	if market == nil {
		return nil, 0
	}
	// create fake/partial key, and split at the _ID:foobar suffix
	parts := strings.Split(
		string(ts.badger.tradeMarketKey(*market, "")),
		"_ID",
	)
	// cast the partial key to []byte in case there's some UTF-8 weirdness
	tmk := []byte(parts[0])
	// return partial key + its length
	return tmk, len(tmk)
}
