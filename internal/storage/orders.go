package storage

import (
	"fmt"
	"sync"

	"vega/internal/filtering"
	"vega/internal/logging"
	types "vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type OrderStore interface {
	Subscribe(orders chan<- []types.Order) uint64
	Unsubscribe(id uint64) error

	// Post adds an order to the store, adds
	// to queue the operation to be committed later.
	Post(order types.Order) error

	// Put updates an order in the store, adds
	// to queue the operation to be committed later.
	Put(order types.Order) error

	// Commit typically saves any operations that are queued to underlying storage medium,
	// if supported by underlying storage implementation.
	Commit() error

	// Close can be called to clean up and close any storage
	// connections held by the underlying storage mechanism.
	Close() error

	// GetByMarket retrieves all orders for a given Market.
	GetByMarket(market string, filters *filtering.OrderQueryFilters) ([]*types.Order, error)
	// GetByMarketAndId retrieves an order for a given Market and id.
	GetByMarketAndId(market string, id string) (*types.Order, error)
	// GetByParty retrieves orders for a given party.
	GetByParty(party string, filters *filtering.OrderQueryFilters) ([]*types.Order, error)
	// GetByPartyAndId retrieves an order for a given Party and id.
	GetByPartyAndId(party string, id string) (*types.Order, error)

	// GetMarketDepth calculates and returns depth of market for a given market.
	GetMarketDepth(market string) (*types.MarketDepth, error)
}

// badgerOrderStore is a package internal data struct that implements the OrderStore interface.
type badgerOrderStore struct {
	*Config
	badger       *badgerStore
	subscribers  map[uint64]chan<- []types.Order
	subscriberId uint64
	buffer       []types.Order
	mu           sync.Mutex
	depth        map[string]MarketDepth
}

// NewOrderStore is used to initialise and create a OrderStore, this implementation is currently
// using the badger k-v persistent storage engine under the hood. The caller will specify a dir to
// use as the storage location on disk for any stored files via Config.
func NewOrderStore(c *Config) (OrderStore, error) {
	err := InitStoreDirectory(c.OrderStoreDirPath)
	if err != nil {
		return nil, errors.Wrap(err, "error on init badger database for orders storage")
	}
	db, err := badger.Open(customBadgerOptions(c.OrderStoreDirPath, c.GetLogger()))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for orders storage")
	}
	bs := badgerStore{db: db}
	return &badgerOrderStore{
		Config:      c,
		badger:      &bs,
		depth:       make(map[string]MarketDepth, 0),
		subscribers: make(map[uint64]chan<- []types.Order),
		buffer:      make([]types.Order, 0),
	}, nil
}

// Subscribe to a channel of new or updated orders. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (os *badgerOrderStore) Subscribe(orders chan<- []types.Order) uint64 {
	os.mu.Lock()
	defer os.mu.Unlock()

	os.subscriberId = os.subscriberId + 1
	os.subscribers[os.subscriberId] = orders

	os.log.Debug("Orders subscriber added in order store",
		logging.Uint64("subscriber-id", os.subscriberId))

	return os.subscriberId
}

// Unsubscribe from an orders channel. Provide the subscriber id you wish to stop receiving new events for.
func (os *badgerOrderStore) Unsubscribe(id uint64) error {
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
func (os *badgerOrderStore) Post(order types.Order) error {
	// validate an order book (depth of market) exists for order market
	if exists := os.depth[order.Market]; exists == nil {
		os.depth[order.Market] = NewMarketDepth(order.Market)
	}
	// with badger we always buffer for future batch insert via Commit()
	os.addToBuffer(order)
	return nil
}

// Put updates an order in the badger store, adds
// to queue the operation to be committed later.
func (os *badgerOrderStore) Put(order types.Order) error {
	os.addToBuffer(order)
	return nil
}

// Commit saves any operations that are queued to badger store, and includes all updates.
// It will also call notify() to push updated data to any subscribers.
func (os *badgerOrderStore) Commit() error {
	if len(os.buffer) == 0 {
		return nil
	}

	os.mu.Lock()
	items := os.buffer
	os.buffer = make([]types.Order, 0)
	os.mu.Unlock()

	err := os.writeBatch(items)
	if err != nil {
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
func (os *badgerOrderStore) Close() error {
	return os.badger.db.Close()
}

// GetByMarket retrieves all orders for a given Market. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (os *badgerOrderStore) GetByMarket(market string, queryFilters *filtering.OrderQueryFilters) ([]*types.Order, error) {
	var result []*types.Order
	if queryFilters == nil {
		queryFilters = &filtering.OrderQueryFilters{}
	}

	txn := os.badger.readTransaction()
	defer txn.Discard()

	filter := OrderFilter{queryFilter: queryFilters}
	descending := filter.queryFilter.HasLast()
	it := os.badger.getIterator(txn, descending)
	defer it.Close()

	marketPrefix, validForPrefix := os.badger.marketPrefix(market, descending)
	for it.Seek(marketPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		item := it.Item()
		orderBuf, _ := item.ValueCopy(nil)
		var order types.Order
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			os.log.Error("Failed to unmarshal order value from badger in order store (getByMarket)",
				logging.Error(err),
				logging.String("badger-key", string(item.Key())),
				logging.String("raw-bytes", string(orderBuf)))

			return nil, err
		}
		if filter.apply(&order) {
			result = append(result, &order)
		}
		if filter.isFull() {
			break
		}
	}

	return result, nil
}

// GetByMarketAndId retrieves an order for a given Market and id, any errors will be returned immediately.
func (os *badgerOrderStore) GetByMarketAndId(market string, id string) (*types.Order, error) {
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
func (os *badgerOrderStore) GetByParty(party string, queryFilters *filtering.OrderQueryFilters) ([]*types.Order, error) {
	var result []*types.Order

	if queryFilters == nil {
		queryFilters = &filtering.OrderQueryFilters{}
	}

	txn := os.badger.readTransaction()
	defer txn.Discard()

	filter := OrderFilter{queryFilter: queryFilters}
	descending := filter.queryFilter.HasLast()
	it := os.badger.getIterator(txn, descending)
	defer it.Close()

	partyPrefix, validForPrefix := os.badger.partyPrefix(party, descending)
	for it.Seek(partyPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		marketKeyItem := it.Item()
		marketKey, _ := marketKeyItem.ValueCopy(nil)
		orderItem, err := txn.Get(marketKey)
		if err != nil {
			os.log.Error("Order with key does not exist in order store (getByParty)",
				logging.String("badger-key", string(marketKey)),
				logging.Error(err))

			return nil, err
		}
		orderBuf, _ := orderItem.ValueCopy(nil)
		var order types.Order
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			os.log.Error("Failed to unmarshal order value from badger in order store (getByParty)",
				logging.Error(err),
				logging.String("badger-key", string(marketKey)),
				logging.String("raw-bytes", string(orderBuf)))
			return nil, err
		}
		if filter.apply(&order) {
			result = append(result, &order)
		}
		if filter.isFull() {
			break
		}
	}
	return result, nil
}

// GetByPartyAndId retrieves a trade for a given Party and id, any errors will be returned immediately.
func (os *badgerOrderStore) GetByPartyAndId(party string, id string) (*types.Order, error) {
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

// GetMarketDepth calculates and returns order book/depth of market for a given market.
func (os *badgerOrderStore) GetMarketDepth(market string) (*types.MarketDepth, error) {

	// validate
	if exists := os.depth[market]; exists == nil {
		return nil, errors.New(fmt.Sprintf("market depth for %s does not exist", market))
	}

	// load from store
	buy := os.depth[market].BuySide()
	sell := os.depth[market].SellSide()

	var buyPtr []*types.PriceLevel
	var sellPtr []*types.PriceLevel

	// recalculate accumulated volume
	// --- buy side ---
	for idx := range buy {
		if idx == 0 {
			buy[idx].CumulativeVolume = buy[idx].Volume

			buyPtr = append(buyPtr, &buy[idx].PriceLevel)
			continue
		}
		buy[idx].CumulativeVolume = buy[idx-1].CumulativeVolume + buy[idx].Volume
		buyPtr = append(buyPtr, &buy[idx].PriceLevel)
	}
	// --- sell side ---
	for idx := range sell {
		if idx == 0 {
			sell[idx].CumulativeVolume = sell[idx].Volume
			sellPtr = append(sellPtr, &sell[idx].PriceLevel)
			continue
		}
		sell[idx].CumulativeVolume = sell[idx-1].CumulativeVolume + sell[idx].Volume
		sellPtr = append(sellPtr, &sell[idx].PriceLevel)
	}

	// return new re-calculated market depth for each side of order book
	orderBook := &types.MarketDepth{Name: market, Buy: buyPtr, Sell: sellPtr}
	return orderBook, nil
}

// add an order to the write-batch/notify buffer.
func (os *badgerOrderStore) addToBuffer(o types.Order) {
	os.mu.Lock()
	os.buffer = append(os.buffer, o)
	os.mu.Unlock()
}

// notify any subscribers of order updates.
func (os *badgerOrderStore) notify(items []types.Order) error {
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

// writeBatch flushes a batch of orders (create/update) to the underlying badger store.
func (os *badgerOrderStore) writeBatch(batch []types.Order) error {

	wb := os.badger.db.NewWriteBatch()
	defer wb.Cancel()

	insertBatchAtomically := func() error {
		for idx := range batch {
			orderBuf, err := proto.Marshal(&batch[idx])
			if err != nil {
				os.log.Error("Failed to marshal order value to badger in order store (writeBatch)",
					logging.Error(err),
					logging.Order(batch[idx]))
			}
			marketKey := os.badger.orderMarketKey(batch[idx].Market, batch[idx].Id)
			idKey := os.badger.orderIdKey(batch[idx].Id)
			partyKey := os.badger.orderPartyKey(batch[idx].Party, batch[idx].Id)
			if err := wb.Set(marketKey, orderBuf, 0); err != nil {
				return err
			}
			if err := wb.Set(idKey, marketKey, 0); err != nil {
				return err
			}
			if err := wb.Set(partyKey, marketKey, 0); err != nil {
				return err
			}
		}
		return nil
	}
	if err := insertBatchAtomically(); err == nil {
		if err := wb.Flush(); err != nil {

			// todo: can we handle flush errors in a similar way to below?
			os.log.Error("Failed to flush batch of orders when calling writeBatch in badger order store",
				logging.Error(err))
		}
	} else {
		wb.Cancel()

		// todo: retry mechanism, also handle badger txn too large errors
		os.log.Error("Failed to insert order batch atomically when calling writeBatch in badger order store",
			logging.Error(err))

		return nil
	}

	// Depth of market updater
	for idx := range batch {
		os.depth[batch[idx].Market].Update(batch[idx])
	}

	return nil
}

// OrderFilter is the order specific filter query data holder. It includes the raw filters
// and helper methods that are used internally to apply and track filter state.
type OrderFilter struct {
	queryFilter *filtering.OrderQueryFilters
	skipped     uint64
	found       uint64
}

func (f *OrderFilter) apply(order *types.Order) (include bool) {
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

	if !applyOrderFilters(order, f.queryFilter) {
		return false
	}

	// if item passes the filter, increment the found queue
	if include {
		f.found++
	}
	return include
}

func (f *OrderFilter) isFull() bool {
	if f.queryFilter.HasLast() && f.found == *f.queryFilter.Last {
		return true
	}
	if f.queryFilter.HasFirst() && f.found == *f.queryFilter.First {
		return true
	}
	return false
}
