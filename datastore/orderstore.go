package datastore

import (
	"errors"
	"fmt"
	"vega/msg"
	"sync"
	"vega/log"
	"vega/filters"

	"github.com/dgraph-io/badger"
)

// orderStore should implement OrderStore interface.
type orderStore struct {
	badger         *badgerStore
	orderBookDepth MarketDepthManager

	subscribers map[uint64] chan<- []msg.Order
	buffer []msg.Order
	subscriberId uint64
	mu sync.Mutex
}

func NewOrderStore(dir string) OrderStore {

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf(err.Error())
	}
	bs := badgerStore{db: db}
	return &orderStore{badger: &bs, orderBookDepth: NewMarketDepthUpdaterGetter()}
}

func (os *orderStore) Close() {
	os.badger.db.Close()
}

func (m *orderStore) Subscribe(orders chan<- []msg.Order) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil {
		log.Debugf("OrderStore -> Subscribe: Creating subscriber chan map")
		m.subscribers = make(map[uint64] chan<- []msg.Order)
	}

	m.subscriberId = m.subscriberId+1
	m.subscribers[m.subscriberId] = orders
	log.Debugf("OrderStore -> Subscribe: Order subscriber added: %d", m.subscriberId)
	return m.subscriberId
}

func (m *orderStore) Unsubscribe(id uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil || len(m.subscribers) == 0 {
		log.Debugf("OrderStore -> Unsubscribe: No subscribers connected")
		return nil
	}

	if _, exists := m.subscribers[id]; exists {
		delete(m.subscribers, id)
		log.Debugf("OrderStore -> Unsubscribe: Subscriber removed: %v", id)
		return nil
	}
	return errors.New(fmt.Sprintf("OrderStore subscriber does not exist with id: %d", id))
}

func (m *orderStore) Notify() error {

	if m.subscribers == nil || len(m.subscribers) == 0 {
		//log.Debugf("OrderStore -> Notify: No subscribers connected")
		return nil
	}

	if m.buffer == nil || len(m.buffer) == 0 {
		// Only publish when we have items
		log.Debugf("OrderStore -> Notify: No orders in buffer")
		return nil
	}
	
	m.mu.Lock()
	items := m.buffer
	m.buffer = nil
	m.mu.Unlock()

	// iterate over items in buffer and push to observers
	var ok bool
	for id, sub := range m.subscribers {
		select {
		case sub <- items:
			ok = true
			break
		default:
			ok = false
		}
		if ok{
			log.Debugf("Orders state updated")
		} else {
			log.Infof("Orders state could not been updated for subscriber %d", id)
		}
	}
	return nil
}

func (m *orderStore) queueEvent(o msg.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil || len(m.subscribers) == 0 {
		//log.Debugf("OrderStore -> queueEvent: No subscribers connected")
		return nil
	}

	if m.buffer == nil {
		m.buffer = make([]msg.Order, 0)
	}

	log.Debugf("OrderStore -> queueEvent: Adding order to buffer: %+v", o)
	m.buffer = append(m.buffer, o)
	return nil
}

func (os *orderStore) GetByMarket(market string, queryFilters *filters.OrderQueryFilters) ([]*msg.Order, error) {
	var result []*msg.Order
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}

	txn := os.badger.db.NewTransaction(false)
	filter := OrderFilter{queryFilter: queryFilters}
	descending := filter.queryFilter.HasLast()
	it := os.badger.getIterator(txn, descending)
	defer it.Close()
	marketPrefix, validForPrefix := os.badger.marketPrefix(market, descending)
	for it.Seek(marketPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		item := it.Item()
		orderBuf, _ := item.ValueCopy(nil)

		var order msg.Order
		order.XXX_Unmarshal(orderBuf)
		if filter.apply(&order) {
			result = append(result, &order)
		}
	}

	log.Debugf("order fetched %d\n", len(result))

	return result, nil
}

// Get retrieves an order for a given market and id.
func (os *orderStore) GetByMarketAndId(market string, id string) (*msg.Order, error) {
	var order msg.Order
	txn := os.badger.db.NewTransaction(false)
	marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", market, id))
	item, err := txn.Get(marketKey)
	if err != nil {
		return nil, err
	}

	orderBf, _ := item.ValueCopy(nil)
	if err := order.XXX_Unmarshal(orderBf); err != nil {
		return nil, err
	}
	return &order, nil
}

func (os *orderStore) GetByParty(party string, queryFilters *filters.OrderQueryFilters) ([]*msg.Order, error) {
	var result []*msg.Order
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}

	txn := os.badger.db.NewTransaction(false)
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
			// todo return or just log this - check with maks?
			return nil, errors.New(fmt.Sprintf("order with key %s does not exist in badger store", string(marketKey)))
		}

		orderBuf, _ := orderItem.ValueCopy(nil)
		var order msg.Order
		order.XXX_Unmarshal(orderBuf)
		if filter.apply(&order) {
			result = append(result, &order)
		}
	}

	fmt.Printf("order fetched %d\n", len(result))

	return result, nil
}

// Get retrieves an order for a given market and id.
func (m *orderStore) GetByPartyAndId(party string, id string) (*msg.Order, error) {
	var order msg.Order
	err := m.badger.db.View(func(txn *badger.Txn) error {
		partyKey := []byte(fmt.Sprintf("P:%s_ID:%s", party, id))
		marketKeyItem, err := txn.Get(partyKey)
		if err != nil {
			return err
		}
		marketKey, err := marketKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		fmt.Printf("marketKey %s\n", string(marketKey))
		orderItem, err := txn.Get(marketKey)
		if err != nil {
			return err
		}

		orderBuf, err := orderItem.ValueCopy(nil)
		if err != nil {
			fmt.Printf("ORDER %s DOES NOT EXIST\n", string(marketKey))
			return err
		}
		order.XXX_Unmarshal(orderBuf)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (os *orderStore) Post(order *msg.Order) error {

	txn := os.badger.db.NewTransaction(true)
	insertAtomically := func(txn *badger.Txn) error {
		orderBuf, _ := order.XXX_Marshal(nil, true)
		marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id))
		idKey := []byte(fmt.Sprintf("ID:%s", order.Id))
		partyKey := []byte(fmt.Sprintf("P:%s_ID:%s", order.Party, order.Id))
		if err := txn.Set(marketKey, orderBuf); err != nil {
			return err
		}
		if err := txn.Set(idKey, marketKey); err != nil {
			return err
		}
		if err := txn.Set(partyKey, marketKey); err != nil {
			return err
		}
		return	nil
	}

	if err := insertAtomically(txn); err != nil {
		return err
	}

	if err := txn.Commit(); err != nil {
		txn.Discard()
		return err
	}

	if order.Remaining != uint64(0) {
		os.orderBookDepth.updateWithRemaining(order)
	}

	os.queueEvent(*order)
	return nil
}

func (os *orderStore) PostBatch(batch []*msg.Order) error {

	wb := os.badger.db.NewWriteBatch()
	defer wb.Cancel()

	insertBatchAtomically := func() error {
		for idx := range batch{
			orderBuf, _ := batch[idx].XXX_Marshal(nil, true)
			marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", batch[idx].Market, batch[idx].Id))
			idKey := []byte(fmt.Sprintf("ID:%s", batch[idx].Id))
			partyKey := []byte(fmt.Sprintf("P:%s_ID:%s", batch[idx].Party, batch[idx].Id))
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
		wb.Flush()
	} else {
		wb.Cancel()
		// implement retry mechanism
	}

	for idx := range batch {
		// Update orderBookDepth
		if batch[idx].Remaining != uint64(0) {
			os.orderBookDepth.updateWithRemaining(batch[idx])
		}
	}

	for idx := range batch {
		os.queueEvent(*batch[idx])
	}

	return nil
}


// Put updates an existing order in the memory store.
func (os *orderStore) Put(order *msg.Order) error {

	var currentOrder msg.Order
	os.badger.db.View(func(txn *badger.Txn) error {
		partyKey := fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id)
		item, err := txn.Get([]byte(partyKey))
		if err != nil {
			return err
		}
		orderBuf, err := item.ValueCopy(nil)
		currentOrder.XXX_Unmarshal(orderBuf)
		if err != nil {
			return err
		}
		return nil
	})

	err := os.badger.db.Update(func(txn *badger.Txn) error {
		orderBuf, _ := order.XXX_Marshal(nil, true)
		marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id))
		txn.Set(marketKey, orderBuf)
		return	nil
	})

	if err != nil {
		return err
	}

	remainingDelta := currentOrder.Remaining - order.Remaining
	if order.Remaining == uint64(0) || order.Status == msg.Order_Cancelled || order.Status == msg.Order_Expired {
		os.orderBookDepth.removeWithRemaining(order)
	} else {
		os.orderBookDepth.updateWithRemainingDelta(order, remainingDelta)
	}

	os.queueEvent(*order)

	return nil
}

// Delete removes an order from the memory store.
func (os *orderStore) Delete(order *msg.Order) error {

	txn := os.badger.db.NewTransaction(true)
	deleteAtomically := func() error {
		marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id))
		idKey := []byte(fmt.Sprintf("ID:%s", order.Id))
		partyKey := []byte(fmt.Sprintf("P:%s_ID:%s", order.Party, order.Id))
		if err := txn.Delete(marketKey); err != nil {
			return err
		}
		if err := txn.Delete(idKey); err != nil {
			return err
		}
		if err := txn.Delete(partyKey); err != nil {
			return err
		}
		return nil
	}

	if err := deleteAtomically(); err != nil {
		txn.Discard()
		return err
	}

	if err := txn.Commit(); err != nil {
		txn.Discard()
		return err
	}

	os.orderBookDepth.removeWithRemaining(order)

	return nil
}

type OrderFilter struct {
	queryFilter *filters.OrderQueryFilters
	skipped uint64
	found uint64
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
