package storage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger/v2"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrOrderNotFoundForMarketAndID   = errors.New("order not found for market and id")
	ErrOrderDoesNotExistForReference = errors.New("order does not exist for reference")
	ErrOrderDoesNotExistForID        = errors.New("order does not exist for ID")
)

// Order is a package internal data struct that implements the OrderStore interface.
type Order struct {
	Config

	mu              sync.Mutex
	log             *logging.Logger
	badger          *badgerStore
	batchCountForGC int32
	subscribers     map[uint64]chan<- []types.Order
	subscriberID    uint64
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
		subscribers:     map[uint64]chan<- []types.Order{},
		onCriticalError: onCriticalError,
	}, nil
}

// ReloadConf reloads the config, watches for a changed loglevel.
func (os *Order) ReloadConf(cfg Config) {
	os.log.Info("reloading configuration")
	if os.log.GetLevel() != cfg.Level.Get() {
		os.log.Info("updating log level",
			logging.String("old", os.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		os.log.SetLevel(cfg.Level.Get())
	}

	os.mu.Lock()
	os.Config = cfg
	os.mu.Unlock()
}

// Subscribe to a channel of new or updated orders. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (os *Order) Subscribe(orders chan<- []types.Order) uint64 {
	os.mu.Lock()
	defer os.mu.Unlock()

	os.subscriberID = os.subscriberID + 1
	os.subscribers[os.subscriberID] = orders

	os.log.Debug("Orders subscriber added in order store",
		logging.Uint64("subscriber-id", os.subscriberID))

	return os.subscriberID
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

	return fmt.Errorf("subscriber to Orders does not exist with id: %d", id)
}

// Close our connection to the badger database
// ensuring errors will be returned up the stack.
func (os *Order) Close() error {
	return os.badger.db.Close()
}

// GetByMarket retrieves all orders for a given Market. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (os *Order) GetByMarket(ctx context.Context, market string, skip,
	limit uint64, descending bool) ([]*types.Order, error) {

	marketPrefix, validForPrefix := os.badger.marketPrefix(market, descending)
	return os.getOrdersIndirectly(ctx, marketPrefix, validForPrefix, skip, limit, descending, nil)
}

// GetByMarketAndID retrieves an order for a given Market and id, any errors will be returned immediately.
func (os *Order) GetByMarketAndID(ctx context.Context, market string, id string) (*types.Order, error) {
	var order types.Order

	err := os.badger.db.View(func(txn *badger.Txn) error {
		marketKey := os.badger.orderMarketKey(market, id)
		primaryKeyItem, err := txn.Get(marketKey)
		if err != nil {
			return err
		}
		primaryKey, err := primaryKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		orderItem, err := txn.Get(primaryKey)
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
				logging.String("badger-key", string(primaryKey)),
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

// GetByParty retrieves orders for a given party. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (os *Order) GetByParty(ctx context.Context, party string, skip uint64,
	limit uint64, descending bool) ([]*types.Order, error) {

	partyPrefix, validForPrefix := os.badger.partyPrefix(party, descending)
	return os.getOrdersIndirectly(ctx, partyPrefix, validForPrefix, skip, limit, descending, nil)
}

// GetByPartyAndID retrieves a trade for a given Party and id, any errors will be returned immediately.
func (os *Order) GetByPartyAndID(ctx context.Context, party string, id string) (*types.Order, error) {
	var order types.Order

	err := os.badger.db.View(func(txn *badger.Txn) error {
		partyKey := os.badger.orderPartyKey(party, id)
		primaryKeyItem, err := txn.Get(partyKey)
		if err != nil {
			return err
		}
		primaryKey, err := primaryKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		orderItem, err := txn.Get(primaryKey)
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
				logging.String("badger-key", string(primaryKey)),
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
		primaryKeyItem, err := txn.Get(refKey)
		if err != nil {
			return err
		}

		primaryKey, err := primaryKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		orderItem, err := txn.Get(primaryKey)
		if err != nil {
			return err
		}
		orderBuf, err := orderItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(orderBuf, &order)
		if err != nil {
			os.log.Error("Failed to unmarshal order value from badger in order store (GetByReference)",
				logging.Error(err),
				logging.String("badger-key", string(refKey)),
				logging.String("raw-bytes", string(orderBuf)))
			return err
		}
		return nil
	})

	if err == badger.ErrKeyNotFound {
		return nil, ErrOrderDoesNotExistForReference
	} else if err != nil {
		return nil, err
	}

	return &order, nil
}

// GetByOrderID retrieves an order for a given orderID, any errors will be returned immediately.
func (os *Order) GetByOrderID(ctx context.Context, id string, version *uint64) (*types.Order, error) {
	var order types.Order
	err := os.badger.db.View(func(txn *badger.Txn) error {

		var primaryKey []byte
		if version == nil {
			idKey := os.badger.orderIDKey(id)
			primaryKeyItem, err := txn.Get(idKey)
			if err != nil {
				return err
			}
			primaryKey, err = primaryKeyItem.ValueCopy(primaryKey)
			if err != nil {
				return err
			}
		} else {
			primaryKey = os.badger.orderIDVersionKey(id, *version)
		}

		orderItem, err := txn.Get(primaryKey)
		if err != nil {
			return err
		}
		orderBuf, err := orderItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(orderBuf, &order)
		if err != nil {
			os.log.Error("Failed to unmarshal order value from badger in order store (GetByOrderId)",
				logging.Error(err),
				logging.String("badger-id-key", string(primaryKey)),
				logging.String("raw-bytes", string(orderBuf)))
			return err
		}
		return nil
	})

	if err == badger.ErrKeyNotFound {
		return nil, ErrOrderDoesNotExistForID
	} else if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetAllVersionsByOrderID returns available versions of the specified order
func (os *Order) GetAllVersionsByOrderID(
	ctx context.Context,
	id string,
	skip uint64,
	limit uint64,
	descending bool,
) ([]*types.Order, error) {

	verionsPrefix, validForPrefix := os.badger.orderIDVersionPrefix(id, descending)
	return os.getOrdersDirectly(ctx, verionsPrefix, validForPrefix, skip, limit, descending, nil)
}

type orderFilter = func(orderEntry *types.Order) bool

// getOrdersIndirectly loads a collection of orders based on keyPrefix
// the function assumes that the orders' primary key value is stored under keyPrefix
func (os *Order) getOrdersIndirectly(
	ctx context.Context,
	keyPrefix, validForPrefix []byte,
	skip, limit uint64,
	descending bool,
	filterOut *orderFilter,
) ([]*types.Order, error) {

	var err error
	result := make([]*types.Order, 0, int(limit))

	ctx, cancel := context.WithTimeout(ctx, os.Config.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()

	txn := os.badger.readTransaction()
	defer txn.Discard()

	it := os.badger.getIterator(txn, descending)
	defer it.Close()

	primaryIndex, orderBuf := []byte{}, []byte{}
	for it.Seek(keyPrefix); it.ValidForPrefix(validForPrefix); it.Next() {

		select {
		case <-ctx.Done():
			if deadline.Before(time.Now()) {
				return nil, ErrTimeoutReached
			}
			return nil, nil
		default:
			if primaryIndex, err = it.Item().ValueCopy(primaryIndex); err != nil {
				return nil, err
			}
			orderItem, err := txn.Get(primaryIndex)
			if err != nil {
				os.log.Error("Order with key does not exist in order store (getOrdersIndirectly)",
					logging.String("badger-key", string(primaryIndex)),
					logging.Error(err))

				return nil, err
			}
			if orderBuf, err = orderItem.ValueCopy(orderBuf); err != nil {
				return nil, err
			}
			var order types.Order
			if err := proto.Unmarshal(orderBuf, &order); err != nil {
				os.log.Error("Failed to unmarshal order value from badger in order store (getOrdersIndirectly)",
					logging.Error(err),
					logging.String("badger-key", string(primaryIndex)),
					logging.String("raw-bytes", string(orderBuf)))
				return nil, err
			}

			if filterOut != nil && (*filterOut)(&order) {
				continue
			}
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
	return result, nil
}

// getOrdersDirectly loads a collection of orders based on the orders' primary key prefix
// function assumes that primary key is defined in `orderIDVersionKey`
func (os *Order) getOrdersDirectly(
	ctx context.Context,
	primaryKeyPrefix, validForPrefix []byte,
	skip, limit uint64,
	descending bool,
	filterOut *orderFilter,
) ([]*types.Order, error) {

	var err error
	result := make([]*types.Order, 0, int(limit))

	ctx, cancel := context.WithTimeout(ctx, os.Config.Timeout.Duration)
	defer cancel()
	deadline, _ := ctx.Deadline()

	txn := os.badger.readTransaction()
	defer txn.Discard()

	it := os.badger.getIterator(txn, descending)
	defer it.Close()

	orderBuf := []byte{}
	for it.Seek(primaryKeyPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
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
			if filterOut != nil && (*filterOut)(&order) {
				continue
			}

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

	return result, nil
}

// notify sends order updates to all subscribers.
func (os *Order) notify(items []types.Order) error {
	if len(items) == 0 {
		return nil
	}

	os.mu.Lock()
	if os.subscribers == nil || len(os.subscribers) == 0 {
		os.mu.Unlock()
		os.log.Debug("No subscribers connected in order store")
		return nil
	}

	var ok bool
	for id, sub := range os.subscribers {
		select {
		case sub <- items:
			ok = true
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
	os.mu.Unlock()
	return nil
}

func (os *Order) orderBatchToMap(batch []types.Order) (map[string][]byte, error) {
	results := make(map[string][]byte)
	for _, order := range batch {
		orderBuf, err := proto.Marshal(&order)
		if err != nil {
			return nil, err
		}
		idVersionKey := os.badger.orderIDVersionKey(order.Id, order.Version)
		marketKey := os.badger.orderMarketKey(order.MarketID, order.Id)
		idKey := os.badger.orderIDKey(order.Id)
		refKey := os.badger.orderReferenceKey(order.Reference)
		partyKey := os.badger.orderPartyKey(order.PartyID, order.Id)

		results[string(idVersionKey)] = orderBuf
		results[string(marketKey)] = idVersionKey
		results[string(idKey)] = idVersionKey
		results[string(partyKey)] = idVersionKey
		results[string(refKey)] = idVersionKey
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
	return nil
}

// SaveBatch writes the given batch of orders to the underlying badger store and notifies any observers.
func (os *Order) SaveBatch(batch []types.Order) error {
	if len(batch) == 0 {
		// Sanity check, no need to do any processing on an empty batch.
		return nil
	}
	timer := metrics.NewTimeCounter("-", "orderstore", "SaveBatch")

	// write the batch down to the badger kv store, notify observers if successful
	err := os.writeBatch(batch)
	if err != nil {
		os.log.Error(
			"unable to write orders batch to badger store",
			logging.Error(err),
		)
		os.onCriticalError()
	} else {
		err = os.notify(batch)
	}

	// Using a batch counter ties the clean up to the average
	// expected size of a batch of account updates, not just time.
	atomic.AddInt32(&os.batchCountForGC, 1)
	if atomic.LoadInt32(&os.batchCountForGC) >= maxBatchesUntilValueLogGC {
		go func() {
			os.log.Info("Orders store value log garbage collection",
				logging.Int32("attempt", atomic.LoadInt32(&os.batchCountForGC)-maxBatchesUntilValueLogGC))

			gcErr := os.badger.GarbageCollectValueLog()
			if gcErr != nil {
				os.log.Error("Unexpected problem running valueLogGC on orders store",
					logging.Error(gcErr))
			} else {
				atomic.StoreInt32(&os.batchCountForGC, 0)
			}
		}()
	}

	timer.EngineTimeCounterAdd()
	return err
}
