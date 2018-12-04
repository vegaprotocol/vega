package datastore

import (
	"errors"
	"fmt"
	"sync"
	"vega/filters"
	"vega/log"
	"vega/msg"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
)

// orderStore should implement OrderStore interface.
type orderStore struct {
	badger         *badgerStore
	orderBookDepth MarketDepthManager
	buffer       []msg.Order

	subscribers  map[uint64]chan<- []msg.Order
	subscriberId uint64

	mu           sync.Mutex
}

func NewOrderStore(dir string) OrderStore {
	db, err := badger.Open(customBadgerOptions(dir))
	if err != nil {
		log.Fatalf(err.Error())
	}
	bs := badgerStore{db: db}
	return &orderStore{badger: &bs, orderBookDepth: NewMarketDepthUpdaterGetter(),
		buffer: make([]msg.Order, 0), subscribers: make(map[uint64]chan<- []msg.Order)}
}

func (os *orderStore) Close() {
	os.badger.db.Close()
}

func (m *orderStore) Subscribe(orders chan<- []msg.Order) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subscriberId = m.subscriberId + 1
	m.subscribers[m.subscriberId] = orders

	log.Debugf("OrderStore -> Subscribe: Order subscriber added: %d", m.subscriberId)
	return m.subscriberId
}

func (m *orderStore) Unsubscribe(id uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.subscribers) == 0 {
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

func (m *orderStore) Commit() error {
	if len(m.buffer) == 0 {
		// Only commit when we have items
		log.Debugf("OrderStore -> Commit: Buffer empty")
		return nil
	}

	m.mu.Lock()
	items := m.buffer
	m.buffer = make([]msg.Order, 0)
	m.mu.Unlock()

	err := m.PostBatch(items)
	if err != nil {
		return err
	}
	err = m.Notify(items)
	if err != nil {
		return err
	}
	return nil
}

func (m *orderStore) Notify(items []msg.Order) error {
	if m.subscribers == nil || len(m.subscribers) == 0 {
		log.Debugf("OrderStore -> Notify: No subscribers connected")
		return nil
	}
	var ok bool
	for id, sub := range m.subscribers {
		select {
		case sub <- items:
			ok = true
			break
		default:
			ok = false
		}
		if ok {
			log.Debugf("OrderStore -> send on channel success for subscriber %d", id)
		} else {
			log.Infof("OrderStore -> channel could not been updated for subscriber %d", id)
		}
	}
	return nil
}

func (m *orderStore) addToBuffer(o msg.Order) {
	m.mu.Lock()
	m.buffer = append(m.buffer, o)
	m.mu.Unlock()

	log.Debugf("OrderStore -> addToBuffer: Adding order to buffer: %+v", o)
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
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			log.Errorf("Unmarshal failed %s", err.Error())
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

// Get retrieves an order for a given Market and id.
func (os *orderStore) GetByMarketAndId(market string, Id string) (*msg.Order, error) {
	var order msg.Order
	txn := os.badger.db.NewTransaction(false)
	marketKey := os.badger.orderMarketKey(market, Id)
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
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			log.Errorf("Unmarshal faixxxxxled %s", err.Error())
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

// Get retrieves an order for a given Market and id.
func (os *orderStore) GetByPartyAndId(party string, Id string) (*msg.Order, error) {
	var order msg.Order
	err := os.badger.db.View(func(txn *badger.Txn) error {
		partyKey := os.badger.orderPartyKey(party, Id)
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
			log.Errorf("ORDER %s DOES NOT EXIST\n", string(marketKey))
			return err
		}
		if err := proto.Unmarshal(orderBuf, &order); err != nil {
			log.Errorf("Unmarshal failed %s", err.Error())
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (os *orderStore) Post(order *msg.Order) error {
	os.addToBuffer(*order)
	return nil
}

func (os *orderStore) PostBatch(batch []msg.Order) error {

	wb := os.badger.db.NewWriteBatch()
	defer wb.Cancel()

	insertBatchAtomically := func() error {
		for idx := range batch {
			orderBuf, err := proto.Marshal(&batch[idx])
			if err != nil {
				log.Errorf("Marshal failed %s", err.Error())
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
			log.Errorf("failed to flush batch %+v \n", err)
		}
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

	return nil
}

// Put updates an existing order in the memory store.
func (os *orderStore) Put(order *msg.Order) error {
	var currentOrder msg.Order
	var recordExistsInBuffer bool

	for idx := range os.buffer {
		if os.buffer[idx].Id == order.Id {
			currentOrder = os.buffer[idx]
			os.buffer[idx] = *order
			recordExistsInBuffer = true
		}
	}

	if !recordExistsInBuffer {
		err := os.badger.db.View(func(txn *badger.Txn) error {
			marketKey := os.badger.orderMarketKey(order.Market, order.Id)
			orderItem, err := txn.Get(marketKey)
			if err != nil {
				return err
			}
			orderBuf, err := orderItem.ValueCopy(nil)
			if err != nil {
				log.Errorf("ORDER %s DOES NOT EXIST\n", string(marketKey))
				return err
			}
			if err := proto.Unmarshal(orderBuf, &currentOrder); err != nil {
				log.Errorf("Unmarshal failed %s", err.Error())
				return err
			}
			return nil
		})
		if err != nil {
			log.Errorf("Failed to fetch current order %s\n", err.Error())
			return err
		}

		err = os.badger.db.Update(func(txn *badger.Txn) error {
			orderBuf, err := proto.Marshal(order)
			if err != nil {
				return err
			}
			marketKey := os.badger.orderMarketKey(order.Market, order.Id)
			txn.Set(marketKey, orderBuf)
			return nil
		})
		if err != nil {
			log.Errorf("Failed to update current order %s\n", err.Error())
		}
	}

	remainingDelta := currentOrder.Remaining - order.Remaining
	if order.Remaining == uint64(0) || order.Status == msg.Order_Cancelled || order.Status == msg.Order_Expired {
		os.orderBookDepth.removeWithRemaining(order)
	} else {
		os.orderBookDepth.updateWithRemainingDelta(order, remainingDelta)
	}

	os.addToBuffer(*order)
	return nil
}

// Delete removes an order from the memory store.
func (os *orderStore) Delete(order *msg.Order) error {

	txn := os.badger.db.NewTransaction(true)
	deleteAtomically := func() error {
		marketKey := os.badger.orderMarketKey(order.Market, order.Id)
		idKey := os.badger.orderIdKey(order.Id)
		partyKey := os.badger.orderPartyKey(order.Party, order.Id)
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

func (m *orderStore) GetMarketDepth(market string) (*msg.MarketDepth, error) {

	// get from store, recalculate accumulated volume and respond
	buy := m.orderBookDepth.getBuySide()
	sell := m.orderBookDepth.getSellSide()

	// recalculate accumulated volume
	for idx := range buy {
		if idx == 0 {
			buy[idx].CumulativeVolume = buy[idx].Volume
			continue
		}
		buy[idx].CumulativeVolume = buy[idx-1].CumulativeVolume + buy[idx].Volume
	}

	for idx := range m.orderBookDepth.getSellSide() {
		if idx == 0 {
			sell[idx].CumulativeVolume = sell[idx].Volume
			continue
		}
		sell[idx].CumulativeVolume = sell[idx-1].CumulativeVolume + sell[idx].Volume
	}

	orderBookDepth := msg.MarketDepth{Name: market, Buy: buy, Sell: sell}

	return &orderBookDepth, nil
}

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
