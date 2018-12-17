package datastore

import (
	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"sync"
	"vega/filters"
	"vega/log"
	"vega/msg"
)

type OrderStore interface {
	Subscribe(orders chan<- []msg.Order) uint64
	Unsubscribe(id uint64) error

	// Post adds an order to the store, adds
	// to queue the operation to be committed later.
	Post(order *msg.Order) error

	// Put updates an order in the store, adds
	// to queue the operation to be committed later.
	Put(order *msg.Order) error

	// Commit typically saves any operations that are queued to underlying storage medium,
	// if supported by underlying storage implementation.
	Commit() error

	// Close can be called to clean up and close any storage
	// connections held by the underlying storage mechanism.
	Close() error

	// GetByMarket retrieves all orders for a given Market.
	GetByMarket(market string, filters *filters.OrderQueryFilters) ([]*msg.Order, error)
	// GetByMarketAndId retrieves an order for a given Market and id.
	GetByMarketAndId(market string, id string) (*msg.Order, error)
	// GetByParty retrieves orders for a given party.
	GetByParty(party string, filters *filters.OrderQueryFilters) ([]*msg.Order, error)
	// GetByPartyAndId retrieves an order for a given Party and id.
	GetByPartyAndId(party string, id string) (*msg.Order, error)
	
	// GetMarketDepth calculates and returns order book depth for a given market.
	GetMarketDepth(market string) (*msg.MarketDepth, error)
}

// badgerOrderStore is a package internal data struct that implements the OrderStore interface.
type badgerOrderStore struct {
	badger         *badgerStore
	subscribers    map[uint64]chan<- []msg.Order
	subscriberId   uint64
	orderBookDepth MarketDepthManager
	buffer         []msg.Order
	mu             sync.Mutex
}

// NewOrderStore is used to initialise and create a OrderStore, this implementation is currently
// using the badger k-v persistent storage engine under the hood. The caller will specify a dir to
// use as the storage location on disk for any stored files.
func NewOrderStore(dir string) OrderStore {
	db, err := badger.Open(customBadgerOptions(dir))
	if err != nil {
		log.Fatalf(err.Error())
	}
	bs := badgerStore{db: db}
	return &badgerOrderStore{
		badger:         &bs,
		orderBookDepth: NewMarketDepthUpdaterGetter(),
		subscribers:    make(map[uint64]chan<- []msg.Order),
		buffer:         make([]msg.Order, 0),
	}
}

// Subscribe to a channel of new or updated orders. The subscriber id will be returned as a uint64 value
// and must be retained for future reference and to unsubscribe.
func (os *badgerOrderStore) Subscribe(orders chan<- []msg.Order) uint64 {
	os.mu.Lock()
	defer os.mu.Unlock()

	os.subscriberId = os.subscriberId + 1
	os.subscribers[os.subscriberId] = orders

	log.Debugf("OrderStore -> Subscribe: Order subscriber added: %d", os.subscriberId)
	return os.subscriberId
}

// Unsubscribe from an orders channel. Provide the subscriber id you wish to stop receiving new events for.
func (os *badgerOrderStore) Unsubscribe(id uint64) error {
	os.mu.Lock()
	defer os.mu.Unlock()

	if len(os.subscribers) == 0 {
		log.Debugf("OrderStore -> Unsubscribe: No subscribers connected")
		return nil
	}

	if _, exists := os.subscribers[id]; exists {
		delete(os.subscribers, id)
		log.Debugf("OrderStore -> Unsubscribe: Subscriber removed: %v", id)
		return nil
	}
	return errors.New(fmt.Sprintf("OrderStore subscriber does not exist with id: %d", id))
}

// Post adds an order to the badger store, adds
// to queue the operation to be committed later.
func (os *badgerOrderStore) Post(order *msg.Order) error {
	// with badger we always buffer for future batch insert via Commit()
	os.addToBuffer(*order)
	return nil
}

// Put updates an order in the badger store, adds
// to queue the operation to be committed later.
func (os *badgerOrderStore) Put(order *msg.Order) error {

	var orderBeforeUpdate msg.Order
	var recordExistsInBuffer bool

	for idx := range os.buffer {
		if os.buffer[idx].Id == order.Id {
			// we found an order in our write queue that matches
			// the order being updated, swap for latest data
			orderBeforeUpdate = os.buffer[idx]
			os.buffer[idx] = *order
			recordExistsInBuffer = true
			break
		}
	}

	if !recordExistsInBuffer {
		// we tried to update a record that is not in our buffer, validate it exists
		// with a read transaction lookup and add to write queue if exists
		if order.Status != msg.Order_Cancelled && order.Status != msg.Order_Expired {
			o, err := os.GetByMarketAndId(order.Market, order.Id)
			if err != nil {
				return err
			}
			orderBeforeUpdate = *o
		}
		os.addToBuffer(*order)
	}

	if order.Status == msg.Order_Cancelled || order.Status == msg.Order_Expired {
		os.orderBookDepth.DecreaseByTradedVolume(order, 0)
	} else {
		os.orderBookDepth.DecreaseByTradedVolume(order, orderBeforeUpdate.Remaining - order.Remaining)
	}

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
	os.buffer = make([]msg.Order, 0)
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
func (os *badgerOrderStore) GetByMarket(market string, queryFilters *filters.OrderQueryFilters) ([]*msg.Order, error) {
	var result []*msg.Order
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
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
		var order msg.Order
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			log.Errorf("unmarshal failed %s", err.Error())
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
func (os *badgerOrderStore) GetByMarketAndId(market string, id string) (*msg.Order, error) {
	var order msg.Order

	txn := os.badger.readTransaction()
	defer txn.Discard()

	marketKey := os.badger.orderMarketKey(market, id)
	item, err := txn.Get(marketKey)
	if err != nil {
		return nil, err
	}
	orderBuf, _ := item.ValueCopy(nil)
	if err := proto.Unmarshal(orderBuf, &order); err != nil {
		log.Errorf("Unmarshal failed %s", err.Error())
		return nil, err
	}
	return &order, nil
}

// GetByParty retrieves orders for a given party. Provide optional query filters to
// refine the data set further (if required), any errors will be returned immediately.
func (os *badgerOrderStore) GetByParty(party string, queryFilters *filters.OrderQueryFilters) ([]*msg.Order, error) {
	var result []*msg.Order

	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
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
			log.Errorf("order with key %s does not exist in store", string(marketKey))
			return nil, err
		}
		orderBuf, _ := orderItem.ValueCopy(nil)
		var order msg.Order
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			log.Errorf("unmarshal failed %s", err.Error())
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
func (os *badgerOrderStore) GetByPartyAndId(party string, id string) (*msg.Order, error) {
	var order msg.Order

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
			log.Errorf("unmarshal failed %s", err.Error())
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &order, nil
}

// GetMarketDepth calculates and returns order book depth for a given market.
func (os *badgerOrderStore) GetMarketDepth(market string) (*msg.MarketDepth, error) {
	// load from store
	buy := os.orderBookDepth.getBuySide()
	sell := os.orderBookDepth.getSellSide()

	// recalculate accumulated volume
	// --- buy side ---
	for idx := range buy {
		if idx == 0 {
			buy[idx].CumulativeVolume = buy[idx].Volume
			continue
		}
		buy[idx].CumulativeVolume = buy[idx-1].CumulativeVolume + buy[idx].Volume
	}
	// --- sell side ---
	for idx := range sell {
		if idx == 0 {
			sell[idx].CumulativeVolume = sell[idx].Volume
			continue
		}
		sell[idx].CumulativeVolume = sell[idx-1].CumulativeVolume + sell[idx].Volume
	}

	// return new re-calculated market depth for each side of order book
	orderBookDepth := msg.MarketDepth{Name: market, Buy: buy, Sell: sell}
	return &orderBookDepth, nil
}

// add an order to the write-batch/notify buffer.
func (os *badgerOrderStore) addToBuffer(o msg.Order) {
	os.mu.Lock()
	os.buffer = append(os.buffer, o)
	os.mu.Unlock()
}

// notify any subscribers of order updates.
func (os *badgerOrderStore) notify(items []msg.Order) error {
	if len(items) == 0 {
		return nil
	}

	if os.subscribers == nil || len(os.subscribers) == 0 {
		log.Debugf("OrderStore: No subscribers connected")
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
			log.Debugf("OrderStore: send on channel success for subscriber %d", id)
		} else {
			log.Infof("OrderStore: channel could not been updated for subscriber %d", id)
		}
	}
	return nil
}

// writeBatch flushes a batch of orders (create/update) to the underlying badger store.
func (os *badgerOrderStore) writeBatch(batch []msg.Order) error {

	wb := os.badger.db.NewWriteBatch()
	defer wb.Cancel()

	insertBatchAtomically := func() error {
		for idx := range batch {
			orderBuf, err := proto.Marshal(&batch[idx])
			if err != nil {
				log.Errorf("marshal failed: %s", err.Error())
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
			log.Errorf("failed to flush batch: %s", err)
		}
	} else {
		wb.Cancel()
		// todo: retry mechanism, also handle badger txn too large errors
		log.Errorf("failed to insert order batch atomically, %s", err)
		return nil
	}

	// update order book depth
	for idx := range batch {
		if batch[idx].Remaining != uint64(0) {
			os.orderBookDepth.Add(&batch[idx])
		}
	}

	return nil
}

// OrderFilter is the order specific filter query data holder. It includes the raw filters
// and helper methods that are used internally to apply and track filter state.
type OrderFilter struct {
	queryFilter *filters.OrderQueryFilters
	skipped     uint64
	found       uint64
}

func (f *OrderFilter) apply(order *msg.Order) (include bool) {
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
	return false
}

