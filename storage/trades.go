package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger/v2"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	// ErrTradeWithUnspecifiedType is used when Trade.Type == types.Trade_TYPE_UNSPECIFIED.
	ErrTradeWithUnspecifiedType = errors.New("trade has unspecified type")
)

// Trade is a package internal data struct that implements the TradeStore interface.
type Trade struct {
	Config

	mu              sync.Mutex
	log             *logging.Logger
	badger          *badgerStore
	subscribers     map[uint64]chan<- []types.Trade
	subscriberID    uint64
	onCriticalError func()
}

// NewTrades is used to initialise and create a TradeStore, this implementation is currently
// using the badger k-v persistent storage engine under the hood. The caller will specify a dir to
// use as the storage location on disk for any stored files via Config.
func NewTrades(log *logging.Logger, c Config, onCriticalError func()) (*Trade, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	err := InitStoreDirectory(c.TradesDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "error on init badger database for trades storage")
	}
	db, err := badger.Open(getOptionsFromConfig(c.Trades, c.TradesDirPath, log))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for trades storage")
	}
	bs := badgerStore{db: db}
	return &Trade{
		log:             log,
		Config:          c,
		badger:          &bs,
		subscribers:     make(map[uint64]chan<- []types.Trade),
		onCriticalError: onCriticalError,
	}, nil
}

// ReloadConf update the internal configuration of the trade
func (ts *Trade) ReloadConf(cfg Config) {
	ts.log.Info("reloading configuration")
	if ts.log.GetLevel() != cfg.Level.Get() {
		ts.log.Info("updating log level",
			logging.String("old", ts.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		ts.log.SetLevel(cfg.Level.Get())
	}

	// only Timeout is really use in here
	ts.mu.Lock()
	ts.Config = cfg
	ts.mu.Unlock()
}

// Subscribe to a channel of new or updated trades. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (ts *Trade) Subscribe(trades chan<- []types.Trade) uint64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.subscriberID++
	ts.subscribers[ts.subscriberID] = trades

	ts.log.Debug("Trades subscriber added in order store",
		logging.Uint64("subscriber-id", ts.subscriberID))

	return ts.subscriberID
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

	return fmt.Errorf("subscriber to Trades store does not exist with id: %d", id)
}

// GetByMarket retrieves trades for a given market. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (ts *Trade) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) ([]*types.Trade, error) {
	// get results cap
	var (
		err error
	)
	//TODO: (WG 05/11/2019): Bug: Setting limit to maximum value of uint64 results in l=-1
	result := make([]*types.Trade, 0, int(limit))

	ctx, cancel := context.WithTimeout(ctx, ts.Config.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()

	txn := ts.badger.readTransaction()
	defer txn.Discard()

	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	marketPrefix, validForPrefix := ts.badger.marketPrefix(market, descending)
	tradeBuf := []byte{}
	for it.Seek(marketPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			if deadline.Before(time.Now()) {
				return nil, ErrTimeoutReached
			}
			return nil, nil
		default:
			if tradeBuf, err = it.Item().ValueCopy(tradeBuf); err != nil {
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

// GetByMarketAndID retrieves a trade for a given market and id, any errors will be returned immediately.
func (ts *Trade) GetByMarketAndID(ctx context.Context, market string, id string) (trade *types.Trade, err error) {
	txn := ts.badger.readTransaction()
	defer txn.Discard()

	marketKey := ts.badger.tradeMarketKey(market, id)
	item, err := txn.Get(marketKey)
	if err != nil {
		return
	}
	tradeBuf, _ := item.ValueCopy(nil)
	trade = &types.Trade{}
	if err = proto.Unmarshal(tradeBuf, trade); err != nil {
		trade = nil
		ts.log.Error("Failed to unmarshal trade value from badger in trade store (getByMarketAndId)",
			logging.Error(err),
			logging.String("badger-key", string(item.Key())),
			logging.String("raw-bytes", string(tradeBuf)))
	}
	return
}

// GetByParty retrieves trades for a given party. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (ts *Trade) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, market *string) ([]*types.Trade, error) {
	var err error
	tmk, tmkLen := ts.getTradeMarketFilter(market)
	result := make([]*types.Trade, 0, int(limit))

	ctx, cancel := context.WithTimeout(ctx, ts.Config.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()

	txn := ts.badger.readTransaction()
	defer txn.Discard()

	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	// reuse these buffers, slices will get reallocated, so if the buffer is big enough
	// next calls won't alloc memory again
	marketKey, tradeBuf := []byte{}, []byte{}
	partyPrefix, validForPrefix := ts.badger.partyPrefix(party, descending)
	for it.Seek(partyPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			if deadline.Before(time.Now()) {
				return nil, ErrTimeoutReached
			}
			return nil, nil
		default:
			// these errors should be logged, means the data is being stored inconsistently
			if marketKey, err = it.Item().ValueCopy(marketKey); err != nil {
				return nil, err
			}
			// we are filtering by market, but the market key doesn't match, stop here, don't waste time reading and unmarshalling the full trade item
			if tmkLen != 0 && string(marketKey[:tmkLen]) != string(tmk) {
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

// GetByPartyAndID retrieves a trade for a given party and id.
func (ts *Trade) GetByPartyAndID(ctx context.Context, party string, id string) (*types.Trade, error) {
	var trade types.Trade
	err := ts.badger.db.View(func(txn *badger.Txn) error {
		partyKey := ts.badger.tradePartyKey(party, id)
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

// GetByOrderID retrieves trades relating to the given order id - buy order Id or sell order Id.
// Provide optional query filters to refine the data set further (if required), any errors will be returned immediately.
func (ts *Trade) GetByOrderID(ctx context.Context, orderID string, skip, limit uint64, descending bool, market *string) ([]*types.Trade, error) {
	var err error
	tmk, tmkLen := ts.getTradeMarketFilter(market)
	result := make([]*types.Trade, 0, int(limit))

	ctx, cancel := context.WithTimeout(ctx, ts.Config.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()

	txn := ts.badger.readTransaction()
	defer txn.Discard()

	it := ts.badger.getIterator(txn, descending)
	defer it.Close()

	orderPrefix, validForPrefix := ts.badger.orderPrefix(orderID, descending)
	marketKey, tradeBuf := []byte{}, []byte{}
	for it.Seek(orderPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			if deadline.Before(time.Now()) {
				return nil, ErrTimeoutReached
			}
			return nil, nil
		default:
			if marketKey, err = it.Item().ValueCopy(marketKey); err != nil {
				return nil, err
			}
			// apply market filter here, avoid getting the trade item + unmarshalling
			if tmkLen != 0 && string(marketKey[:tmkLen]) != string(tmk) {
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
		if trade.Type == types.Trade_TYPE_UNSPECIFIED {
			ts.log.Error("attempting to store a trade with UNSPECIFIED Type (tradeBatchToMap)")
			return nil, ErrTradeWithUnspecifiedType
		}
		tradeBuf, err := proto.Marshal(&trade)
		if err != nil {
			return nil, err
		}
		// Market Index
		marketKey := ts.badger.tradeMarketKey(trade.MarketID, trade.Id)
		// Trade Id index
		idKey := ts.badger.tradeIDKey(trade.Id)
		// Party indexes (buyer and seller as parties)
		buyerPartyKey := ts.badger.tradePartyKey(trade.Buyer, trade.Id)
		sellerPartyKey := ts.badger.tradePartyKey(trade.Seller, trade.Id)
		// OrderId indexes (relate to both buy and sell orders)
		buyOrderKey := ts.badger.tradeOrderIDKey(trade.BuyOrder, trade.Id)
		sellOrderKey := ts.badger.tradeOrderIDKey(trade.SellOrder, trade.Id)

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

// SaveBatch writes the given batch of trades to the underlying badger store and notifies any observers.
func (ts *Trade) SaveBatch(batch []types.Trade) error {
	if len(batch) == 0 {
		// Sanity check, no need to do any processing on an empty batch.
		return nil
	}
	timer := metrics.NewTimeCounter("-", "tradestore", "SaveBatch")

	// write the batch down to the badger kv store, notify observers if successful
	err := ts.writeBatch(batch)
	if err != nil {
		ts.log.Error(
			"unable to write trades batch to badger store",
			logging.Error(err),
		)
		ts.onCriticalError()
	} else {
		err = ts.notify(batch)
	}

	timer.EngineTimeCounterAdd()
	return err
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
