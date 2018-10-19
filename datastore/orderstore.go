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
	//store *MemStore

	persistentStore *badger.DB
	orderBookDepth MarketDepthUpdater

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
	return &orderStore{persistentStore: db, orderBookDepth: NewMarketDepthUpdater()}
}

func (os *orderStore) Close() {
	os.persistentStore.Close()
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
		log.Debugf("OrderStore -> Notify: No subscribers connected")
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
		log.Debugf("OrderStore -> queueEvent: No subscribers connected")
		return nil
	}

	if m.buffer == nil {
		m.buffer = make([]msg.Order, 0)
	}

	log.Debugf("OrderStore -> queueEvent: Adding order to buffer: %+v", o)
	m.buffer = append(m.buffer, o)
	return nil
}

func (m *orderStore) GetByMarket(market string, queryFilters *filters.OrderQueryFilters) ([]*msg.Order, error) {
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}

	var (
		result []*msg.Order
		tempOrder msg.Order
	)

	m.persistentStore.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		marketPrefix := []byte(fmt.Sprintf("M:%s_", market))
		filter := Filter{queryFilters, 0}

		for it.Seek(marketPrefix); it.ValidForPrefix(marketPrefix); it.Next() {
			item := it.Item()
			orderBuf, _ := item.ValueCopy(nil)
			tempOrder.XXX_Unmarshal(orderBuf)
			if filter.apply(&tempOrder) {
				// allocate memory and append pointer
				var order msg.Order
				order = tempOrder
				result = append(result, &order)
			}
		}
		return nil
	})

	return result, nil
}

// Get retrieves an order for a given market and id.
func (m *orderStore) GetByMarketAndId(market string, id string) (*msg.Order, error) {
	var order *msg.Order
	err := m.persistentStore.View(func(txn *badger.Txn) error {
		marketKey := fmt.Sprintf("M:%s_ID:%s", market, id)
		item, err := txn.Get([]byte(marketKey))
		if err != nil {
			return err
		}

		orderBuf, _ := item.ValueCopy(nil)
		order.XXX_Unmarshal(orderBuf)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return order, nil
}

func (m *orderStore) GetByParty(party string, queryFilters *filters.OrderQueryFilters) ([]*msg.Order, error) {
	//if !m.partyExists(party) {
	//	return nil, NotFoundError{fmt.Errorf("could not find party %s", party)}
	//}
	//if queryFilters == nil {
	//	queryFilters = &filters.OrderQueryFilters{}
	//}
	//
	//return m.filterResults(m.store.parties[party].ordersByTimestamp, queryFilters)
	return nil, nil
}

// Get retrieves an order for a given market and id.
func (m *orderStore) GetByPartyAndId(party string, id string) (*msg.Order, error) {
	var order *msg.Order
	err := m.persistentStore.View(func(txn *badger.Txn) error {
		partyKey := fmt.Sprintf("P:%s_ID:%s", party, id)
		item, err := txn.Get([]byte(partyKey))
		if err != nil {
			return err
		}
		marketKey, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		item, err = txn.Get(marketKey)
		if err != nil {
			return err
		}

		orderBuf, _ := item.ValueCopy(nil)
		order.XXX_Unmarshal(orderBuf)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return order, nil
}

func (m *orderStore) GetByPartyAndReference(party string, reference string) (*msg.Order, error) {
	//if exists := m.partyExists(party); !exists {
	//	return Order{}, fmt.Errorf("could not find party %s", party)
	//}
	//
	//var at = -1
	//for idx, order := range m.store.parties[party].ordersByTimestamp {
	//	if order.order.Reference == reference {
	//		at = idx
	//		break
	//	}
	//}
	//
	//if at == -1 {
	//	return Order{}, NotFoundError{fmt.Errorf("could not find reference %s", reference)}
	//}
	//return m.store.parties[party].ordersByTimestamp[at].order, nil
	return nil, nil
}

func (os *orderStore) Post(order *msg.Order) error {

	insertAtomically := func() error {
		txn := os.persistentStore.NewTransaction(true)
		orderBuf, _ := order.XXX_Marshal(nil, true)
		marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id))
		idKey := []byte(fmt.Sprintf("ID:%s", order.Id))
		partyKey := []byte(fmt.Sprintf("P:%s_ID:%s", order.Party, order.Id))
		txn.Set(marketKey, orderBuf)
		txn.Set(idKey, marketKey)
		txn.Set(partyKey, marketKey)
		return	nil
	}

	if err := insertAtomically(); err != nil {
		return err
	}

	// Update orderBookDepth
	if order.Remaining != uint64(0) {
		os.orderBookDepth.updateWithRemaining(&order)
	}

	os.queueEvent(*order)
	return nil
}

func (os *orderStore) PostBatch(batch []*msg.Order) error {


	wb := os.persistentStore.NewWriteBatch()
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
			os.orderBookDepth.updateWithRemaining(&batch[idx])
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
	os.persistentStore.View(func(txn *badger.Txn) error {
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

	err := os.persistentStore.Update(func(txn *badger.Txn) error {
		orderBuf, _ := order.XXX_Marshal(nil, true)
		marketKey := fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id)
		txn.Set([]byte(marketKey), orderBuf)
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

	deleteAtomically := func() error {
		txn := os.persistentStore.NewTransaction(true)
		marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id))
		idKey := []byte(fmt.Sprintf("ID:%s", order.Id))
		partyKey := []byte(fmt.Sprintf("P:%s_ID:%s", order.Party, order.Id))
		if err := txn.Delete(marketKey); err != nil {
			txn.Discard()
			return err
		}
		if err := txn.Delete(idKey); err != nil {
			txn.Discard()
			return err
		}
		if err := txn.Delete(partyKey); err != nil {
			txn.Discard()
			return err
		}
		if err := txn.Commit(); err != nil {
			txn.Discard()
			return err
		}
		return nil
	}

	if err := deleteAtomically(); err != nil {
		return err
	}

	os.orderBookDepth.removeWithRemaining(&order)

	return nil
}

type Filter struct {
	queryFilter *filters.OrderQueryFilters
	Q uint64
}

func (f Filter) apply(order *msg.Order) (include bool) {
	if f.queryFilter.First != nil && *f.queryFilter.First > 0 && f.Q < *f.queryFilter.First {
		include = true
	}

	if !applyOrderFilters2(*order, f.queryFilter) {
		include = false
	}

	f.Q++
	return include
}


func (m *orderStore) filterResults3(input []msg.Order, queryFilters *filters.OrderQueryFilters) (output []msg.Order, error error) {
	var pos, skipped uint64

	// Last == descending by timestamp
	// First == ascending by timestamp
	// Skip == offset by value, then first/last depending on direction

	if queryFilters.First != nil && *queryFilters.First > 0 {
		// If first is set we iterate ascending
		for i := 0; i < len(input); i++ {
			if pos == *queryFilters.First {
				break
			}
			if applyOrderFilters2(input[i], queryFilters) {
				if queryFilters.Skip != nil && *queryFilters.Skip > 0 && skipped < *queryFilters.Skip {
					skipped++
					continue
				}
				output = append(output, input[i])
				pos++
			}
		}
	} else {
		// default is descending 'last' n items
		for i := len(input) - 1; i >= 0; i-- {
			if queryFilters.Last != nil && *queryFilters.Last > 0 && pos == *queryFilters.Last {
				break
			}
			if applyOrderFilters2(input[i], queryFilters) {
				if queryFilters.Skip != nil && *queryFilters.Skip > 0 && skipped < *queryFilters.Skip {
					skipped++
					continue
				}
				output = append(output, input[i])
				pos++
			}
		}
	}

	return output, nil
}