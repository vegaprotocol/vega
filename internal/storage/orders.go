package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// Order is a package internal data struct that implements the OrderStore interface.
type Order struct {
	Config

	cfgMu           sync.Mutex
	log             *logging.Logger
	badger          *badgerStore
	subscribers     map[uint64]chan<- []types.Order
	subscriberId    uint64
	buffer          []types.Order
	depth           map[string]*Depth
	mu              sync.Mutex
	onCriticalError func()
}

// NewOrders is used to initialise and create a OrderStore, this implementation is currently
// using the badger k-v persistent storage engine under the hood. The caller will specify a dir to
// use as the storage location on disk for any stored files via Config.
func NewOrders(log *logging.Logger, c Config, onCriticalError func()) (*Order, error) {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	err := InitStoreDirectory(c.OrdersDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "error on init badger database for orders storage")
	}
	db, err := badger.Open(getOptionsFromConfig(c.Orders, c.OrdersDirPath, log))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for orders storage")
	}
	bs := badgerStore{db: db}
	return &Order{
		log:             log,
		Config:          c,
		badger:          &bs,
		depth:           map[string]*Depth{},
		subscribers:     map[uint64]chan<- []types.Order{},
		buffer:          []types.Order{},
		onCriticalError: onCriticalError,
	}, nil
}

func (os *Order) ReloadConf(cfg Config) {
	os.log.Info("reloading configuration")
	if os.log.GetLevel() != cfg.Level.Get() {
		os.log.Info("updating log level",
			logging.String("old", os.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		os.log.SetLevel(cfg.Level.Get())
	}

	os.cfgMu.Lock()
	os.Config = cfg
	os.cfgMu.Unlock()
}

// Subscribe to a channel of new or updated orders. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (os *Order) Subscribe(orders chan<- []types.Order) uint64 {
	os.mu.Lock()
	defer os.mu.Unlock()

	os.subscriberId = os.subscriberId + 1
	os.subscribers[os.subscriberId] = orders

	os.log.Debug("Orders subscriber added in order store",
		logging.Uint64("subscriber-id", os.subscriberId))

	return os.subscriberId
}

// Unsubscribe from an orders channel. Provide the subscriber id you wish to stop receiving new events for.
func (os *Order) Unsubscribe(id uint64) error {
	os.mu.Lock()
	defer os.mu.Unlock()

	if len(os.subscribers) == 0 {
		os.log.Debug("Un-subscribe called in order store, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := os.subscribers[id]; exists {
		delete(os.subscribers, id)
		os.log.Debug("Un-subscribe called in order store, subscriber removed",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	return errors.New(fmt.Sprintf("Orders subscriber does not exist with id: %d", id))
}

// Post adds an order to the badger store, adds
// to queue the operation to be committed later.
func (os *Order) Post(order types.Order) error {
	// validate an order book (depth of market) exists for order market
	if exists := os.depth[order.MarketID]; exists == nil {
		os.depth[order.MarketID] = NewMarketDepth(order.MarketID)
	}
	// with badger we always buffer for future batch insert via Commit()
	os.addToBuffer(order)
	return nil
}

// Put updates an order in the badger store, adds
// to queue the operation to be committed later.
func (os *Order) Put(order types.Order) error {
	os.addToBuffer(order)
	return nil
}

// Commit saves any operations that are queued to badger store, and includes all updates.
// It will also call notify() to push updated data to any subscribers.
func (os *Order) Commit() error {
	if len(os.buffer) == 0 {
		return nil
	}

	os.mu.Lock()
	items := os.buffer
	os.buffer = make([]types.Order, 0)
	os.mu.Unlock()

	err := os.writeBatch(items)
	if err != nil {
		os.log.Error(
			"unable to write batch in order badger store",
			logging.Error(err),
		)
		os.onCriticalError()
		return err
	}
	err = os.notify(items)
	if err != nil {
		return err
	}
	return nil
}

// Close our connection to the badger database
// ensuring errors will be returned up the stack.
func (os *Order) Close() error {
	return os.badger.db.Close()
}

// GetByMarket retrieves all orders for a given Market. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (os *Order) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) ([]*types.Order, error) {
	var err error
	result := make([]*types.Order, 0, int(limit))

	os.cfgMu.Lock()
	ctx, cancel := context.WithTimeout(ctx, os.Config.Timeout.Duration)
	os.cfgMu.Unlock()
	defer cancel()

	txn := os.badger.readTransaction()
	defer txn.Discard()

	it := os.badger.getIterator(txn, descending)
	defer it.Close()

	deadline, _ := ctx.Deadline()
	marketPrefix, validForPrefix := os.badger.marketPrefix(market, descending)
	orderBuf := []byte{}
	openOnly := open != nil && *open
	for it.Seek(marketPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		select {
		case <-ctx.Done():
			if deadline.Before(time.Now()) {
				return nil, ErrTimeoutReached
			}
			return nil, nil
		default:
			if orderBuf, err = it.Item().ValueCopy(orderBuf); err != nil {
				return nil, err
			}
			var order types.Order
			if err := proto.Unmarshal(orderBuf, &order); err != nil {
				os.log.Error("Failed to unmarshal order value from badger in order store (getByMarket)",
					logging.Error(err),
					logging.String("badger-key", string(it.Item().Key())),
					logging.String("raw-bytes", string(orderBuf)))

				return nil, err
			}
			if !openOnly || (order.Remaining == 0 || order.Status != types.Order_Active) {
				if skip != 0 {
					skip--
					continue
				}
				result = append(result, &order)
				if limit != 0 && len(result) == cap(result) {
					return result, nil
				}
			}
		}
	}

	return result, nil
}

// GetByMarketAndId retrieves an order for a given Market and id, any errors will be returned immediately.
func (os *Order) GetByMarketAndId(ctx context.Context, market string, id string) (*types.Order, error) {
	var order types.Order

	txn := os.badger.readTransaction()
	defer txn.Discard()

	marketKey := os.badger.orderMarketKey(market, id)
	item, err := txn.Get(marketKey)
	if err != nil {
		return nil, err
	}
	orderBuf, _ := item.ValueCopy(nil)
	if err := proto.Unmarshal(orderBuf, &order); err != nil {
		os.log.Error("Failed to unmarshal order value from badger in order store (getByMarketAndId)",
			logging.Error(err),
			logging.String("badger-key", string(item.Key())),
			logging.String("raw-bytes", string(orderBuf)))
		return nil, err
	}
	return &order, nil
}

// GetByParty retrieves orders for a given party. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (os *Order) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) ([]*types.Order, error) {
	var err error
	openOnly := open != nil && *open
	result := make([]*types.Order, 0, int(limit))

	ctx, cancel := context.WithTimeout(ctx, os.Config.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()

	txn := os.badger.readTransaction()
	defer txn.Discard()

	it := os.badger.getIterator(txn, descending)
	defer it.Close()

	partyPrefix, validForPrefix := os.badger.partyPrefix(party, descending)
	marketKey, orderBuf := []byte{}, []byte{}
	for it.Seek(partyPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
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
			orderItem, err := txn.Get(marketKey)
			if err != nil {
				os.log.Error("Order with key does not exist in order store (getByParty)",
					logging.String("badger-key", string(marketKey)),
					logging.Error(err))

				return nil, err
			}
			if orderBuf, err = orderItem.ValueCopy(orderBuf); err != nil {
				return nil, err
			}
			var order types.Order
			if err := proto.Unmarshal(orderBuf, &order); err != nil {
				os.log.Error("Failed to unmarshal order value from badger in order store (getByParty)",
					logging.Error(err),
					logging.String("badger-key", string(marketKey)),
					logging.String("raw-bytes", string(orderBuf)))
				return nil, err
			}
			if !openOnly || (order.Remaining == 0 || order.Status != types.Order_Active) {
				if skip != 0 {
					skip--
					continue
				}
				result = append(result, &order)
				if limit != 0 && len(result) == cap(result) {
					return result, nil
				}
			}
		}
	}
	return result, nil
}

// GetByPartyAndId retrieves a trade for a given Party and id, any errors will be returned immediately.
func (os *Order) GetByPartyAndId(ctx context.Context, party string, id string) (*types.Order, error) {
	var order types.Order

	err := os.badger.db.View(func(txn *badger.Txn) error {
		partyKey := os.badger.orderPartyKey(party, id)
		marketKeyItem, err := txn.Get(partyKey)
		if err != nil {
			return err
		}
		marketKey, err := marketKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		orderItem, err := txn.Get(marketKey)
		if err != nil {
			return err
		}
		orderBuf, err := orderItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			os.log.Error("Failed to unmarshal order value from badger in order store (getByPartyAndId)",
				logging.Error(err),
				logging.String("badger-key", string(marketKey)),
				logging.String("raw-bytes", string(orderBuf)))
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// GetByReference retrieves an order for a given reference, any errors will be returned immediately.
func (os *Order) GetByReference(ctx context.Context, ref string) (*types.Order, error) {
	var order types.Order

	err := os.badger.db.View(func(txn *badger.Txn) error {
		refKey := os.badger.orderReferenceKey(ref)
		marketKeyItem, err := txn.Get(refKey)
		if err != nil {
			return err
		}
		marketKey, err := marketKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		orderItem, err := txn.Get(marketKey)
		if err != nil {
			return err
		}
		orderBuf, err := orderItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			os.log.Error("Failed to unmarshal order value from badger in order store (getByPartyAndId)",
				logging.Error(err),
				logging.String("badger-key", string(refKey)),
				logging.String("raw-bytes", string(orderBuf)))
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// GetMarketDepth calculates and returns order book/depth of market for a given market.
func (os *Order) GetMarketDepth(ctx context.Context, market string) (*types.MarketDepth, error) {

	// validate
	depth, ok := os.depth[market]
	if !ok || depth == nil {
		// When a market is new with no orders there will not be any market depth/order book
		// so we do not need to try and calculate the depth cumulative volumes etc
		return &types.MarketDepth{
			MarketID: market,
			Buy:      []*types.PriceLevel{},
			Sell:     []*types.PriceLevel{},
		}, nil
	}

	// load from store
	buy := depth.BuySide()
	sell := depth.SellSide()

	buyPtr := make([]*types.PriceLevel, 0, len(buy))
	sellPtr := make([]*types.PriceLevel, 0, len(sell))

	ctx, cancel := context.WithTimeout(ctx, os.Config.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()
	// 2 routines, each can push one error on here, so buffer to avoid deadlock
	errCh := make(chan error, 2)
	wg := sync.WaitGroup{}
	wg.Add(2)

	// recalculate accumulated volume, concurrently rather than sequentially
	// make the most of the time we have
	// --- buy side ---
	go func() {
		defer wg.Done()
		var cumulativeVolume uint64
		for i, b := range buy {
			select {
			case <-ctx.Done():
				if deadline.Before(time.Now()) {
					errCh <- ErrTimeoutReached
				}
				return
			default:
				// keep running total
				cumulativeVolume += b.Volume
				buy[i].CumulativeVolume = cumulativeVolume
				buyPtr = append(buyPtr, &buy[i].PriceLevel)
			}
		}
	}()
	// --- sell side ---
	go func() {
		defer wg.Done()
		var cumulativeVolume uint64
		for i, s := range sell {
			select {
			case <-ctx.Done():
				if deadline.Before(time.Now()) {
					errCh <- ErrTimeoutReached
				}
				return
			default:
				// keep running total
				cumulativeVolume += s.Volume
				sell[i].CumulativeVolume = cumulativeVolume
				sellPtr = append(sellPtr, &sell[i].PriceLevel)
			}
		}
	}()
	wg.Wait()
	close(errCh)
	// the second error is the same, they're both ctx.Err()
	for err := range errCh {
		return nil, err
	}

	// return new re-calculated market depth for each side of order book
	return &types.MarketDepth{
		MarketID: market,
		Buy:      buyPtr,
		Sell:     sellPtr,
	}, nil
}

// add an order to the write-batch/notify buffer.
func (os *Order) addToBuffer(o types.Order) {
	os.mu.Lock()
	os.buffer = append(os.buffer, o)
	os.mu.Unlock()
}

// notify any subscribers of order updates.
func (os *Order) notify(items []types.Order) error {
	if len(items) == 0 {
		return nil
	}

	if os.subscribers == nil || len(os.subscribers) == 0 {
		os.log.Debug("No subscribers connected in order store")
		return nil
	}

	var ok bool
	for id, sub := range os.subscribers {
		select {
		case sub <- items:
			ok = true
			break
		default:
			ok = false
		}
		if ok {
			os.log.Debug("Orders channel updated for subscriber successfully",
				logging.Uint64("id", id))
		} else {
			os.log.Debug("Orders channel could not be updated for subscriber",
				logging.Uint64("id", id))
		}
	}
	return nil
}

func (os *Order) orderBatchToMap(batch []types.Order) (map[string][]byte, error) {
	results := make(map[string][]byte)
	for _, order := range batch {
		orderBuf, err := proto.Marshal(&order)
		if err != nil {
			return nil, err
		}
		marketKey := os.badger.orderMarketKey(order.MarketID, order.Id)
		idKey := os.badger.orderIdKey(order.Id)
		refKey := os.badger.orderReferenceKey(order.Reference)
		partyKey := os.badger.orderPartyKey(order.PartyID, order.Id)
		results[string(marketKey)] = orderBuf
		results[string(idKey)] = marketKey
		results[string(partyKey)] = marketKey
		results[string(refKey)] = marketKey
	}
	return results, nil
}

// writeBatch flushes a batch of orders (create/update) to the underlying badger store.
func (os *Order) writeBatch(batch []types.Order) error {
	kv, err := os.orderBatchToMap(batch)
	if err != nil {
		os.log.Error("Failed to marshal orders before writing batch",
			logging.Error(err))
		return err
	}

	b, err := os.badger.writeBatch(kv)
	if err != nil {
		if b == 0 {
			os.log.Warn("Failed to insert order batch; No records were committed, atomicity maintained",
				logging.Error(err))
			// TODO: Retry, in some circumstances.
		} else {
			os.log.Error("Failed to insert order batch; Some records were committed, atomicity lost",
				logging.Error(err))
			// TODO: Mark block dirty, panic node.
		}
		return err
	}

	// Depth of market updater
	for idx := range batch {
		os.depth[batch[idx].MarketID].Update(batch[idx])
	}

	return nil
}
