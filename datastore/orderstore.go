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
	store *MemStore

	persistentStore *badger.DB
	orderBookDepth MarketDepthUpdater

	subscribers map[uint64] chan<- []Order
	buffer []Order
	subscriberId uint64
	mu sync.Mutex
}

// NewOrderStore initialises a new OrderStore backed by a MemStore.
func NewOrderStore(ms *MemStore) OrderStore {
	return &orderStore{store: ms, persistentStore: nil, orderBookDepth: NewMarketDepthUpdater()}
}

func NewOrderStoreP(ms *MemStore, dir string) OrderStore {
	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	//fmt.Println("ex ", opts.Dir)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return &orderStore{store: ms, persistentStore: db, orderBookDepth: NewMarketDepthUpdater()}
}

func (os *orderStore) Close() {
	os.persistentStore.Close()
}

func (m *orderStore) Subscribe(orders chan<- []Order) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil {
		log.Debugf("OrderStore -> Subscribe: Creating subscriber chan map")
		m.subscribers = make(map[uint64] chan<- []Order)
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

func (m *orderStore) queueEvent(o Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil || len(m.subscribers) == 0 {
		log.Debugf("OrderStore -> queueEvent: No subscribers connected")
		return nil
	}

	if m.buffer == nil {
		m.buffer = make([]Order, 0)
	}

	log.Debugf("OrderStore -> queueEvent: Adding order to buffer: %+v", o)
	m.buffer = append(m.buffer, o)
	return nil
}

func (m *orderStore) GetByMarket(market string, queryFilters *filters.OrderQueryFilters) ([]Order, error) {
	if err := m.marketExists(market); err != nil {
		return nil, err
	}
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}

	return m.filterResults(m.store.markets[market].ordersByTimestamp, queryFilters)
}

func (m *orderStore) GetByMarket2(market string, queryFilters *filters.OrderQueryFilters) ([]Order, error) {

	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}

	var orderBuffers, out []msg.Order
	var tempOrder msg.Order
	m.persistentStore.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		marketPrefix := []byte(fmt.Sprintf("M:%s_", market))
		for it.Seek(marketPrefix); it.ValidForPrefix(marketPrefix); it.Next() {
			item := it.Item()
			//k := item.Key()
			//err := item.Value(func(v []byte) error {
			//	//fmt.Printf("key=%s, value=%s\n", k, v)
			//	return nil
			//})
			//if err != nil {
			//	return err
			//}

			orderBuf, _ := item.ValueCopy(nil)
			tempOrder.XXX_Unmarshal(orderBuf)
			out, _ = m.filterResults3([]msg.Order{tempOrder}, queryFilters)

			orderBuffers = append(orderBuffers, out...)
		}
		return nil
	})



	var result []Order
	for _, order := range orderBuffers {
		result = append(result, *NewOrderFromProtoMessage(&order))
	}
	//return m.filterResults2(result, queryFilters)
	return result, nil
}

// Get retrieves an order for a given market and id.
func (m *orderStore) GetByMarketAndId(market string, id string) (Order, error) {
	if err := m.marketExists(market); err != nil {
		return Order{}, err
	}
	_, ok := m.store.markets[market].orders[id]
	if !ok {
		return Order{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}

	var orderBuf []byte
	m.persistentStore.View(func(txn *badger.Txn) error {
		marketKey := fmt.Sprintf("M:%s_ID:%s", market, id)
		item, err := txn.Get([]byte(marketKey))
		if err != nil {
			return err
		}

		orderBuf, _ = item.ValueCopy(nil)
		return nil
	})
	var order msg.Order
	order.XXX_Unmarshal(orderBuf)

	//return v.order, nil
	return *NewOrderFromProtoMessage(&order), nil
}

func (m *orderStore) GetByParty(party string, queryFilters *filters.OrderQueryFilters) ([]Order, error) {
	if !m.partyExists(party) {
		return nil, NotFoundError{fmt.Errorf("could not find party %s", party)}
	}
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}

	return m.filterResults(m.store.parties[party].ordersByTimestamp, queryFilters)
}

// Get retrieves an order for a given market and id.
func (m *orderStore) GetByPartyAndId(party string, id string) (Order, error) {
	if !m.partyExists(party) {
		return Order{}, NotFoundError{fmt.Errorf("could not find party %s", party)}
	}
	
	var at = -1
	for idx, order := range m.store.parties[party].ordersByTimestamp {
		if order.order.Id == id {
			at = idx
			break
		}
	}

	if at == -1 {
		return Order{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}


	var orderBuf []byte
	m.persistentStore.View(func(txn *badger.Txn) error {
		partyKey := fmt.Sprintf("P:%s_ID:%s", party, id)
		item, err := txn.Get([]byte(partyKey))
		if err != nil {
			return err
		}
		marketKey, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}

		fmt.Println("fetched item ", string(marketKey))

		item, err = txn.Get(marketKey)
		if err != nil {
			return err
		}

		orderBuf, _ = item.ValueCopy(nil)
		fmt.Println("fetched item 2 ", string(orderBuf))
		return nil
	})
	var order msg.Order
	order.XXX_Unmarshal(orderBuf)
	fmt.Printf("order %+v", order)

	return *NewOrderFromProtoMessage(&order), nil
	//return m.store.parties[party].ordersByTimestamp[at].order, nil
}

func (m *orderStore) GetByPartyAndReference(party string, reference string) (Order, error) {
	if exists := m.partyExists(party); !exists {
		return Order{}, fmt.Errorf("could not find party %s", party)
	}

	var at = -1
	for idx, order := range m.store.parties[party].ordersByTimestamp {
		if order.order.Reference == reference {
			at = idx
			break
		}
	}

	if at == -1 {
		return Order{}, NotFoundError{fmt.Errorf("could not find reference %s", reference)}
	}
	return m.store.parties[party].ordersByTimestamp[at].order, nil
}

// Post creates a new order in the memory store.
func (os *orderStore) Post(order Order) error {
	if err := os.validate(&order); err != nil {
		return err
	}

	// Order cannot already exist in the store
	if _, exists := os.store.markets[order.Market].orders[order.Id]; exists {
		return fmt.Errorf("order exists in memstore: %s", order.Id)
	}

	// Party 'name' is added on the fly to the parties store
	if !os.partyExists(order.Party) {
		os.newMemParty(order.Party)
		log.Debugf("new party added to store: %v", order.Party)
	}

	newOrder := &memOrder{
		trades: make([]*memTrade, 0),
		order:  order,
	}

	// Insert new order struct into lookup hash table
	os.store.markets[order.Market].orders[order.Id] = newOrder

	// Insert new order into slice of orders ordered by timestamp
	os.store.markets[order.Market].ordersByTimestamp = append(os.store.markets[order.Market].ordersByTimestamp, newOrder)

	// Insert new order into Party map of slices of orders
	os.store.parties[order.Party].ordersByTimestamp = append(os.store.parties[order.Party].ordersByTimestamp, newOrder)

	// Update orderBookDepth
	if newOrder.order.Remaining != uint64(0) {
		os.store.markets[order.Market].marketDepth.updateWithRemaining(&order)
	}

	//os.queueEvent(order)
	return nil
}

func (os *orderStore) PostP(order Order) error {
	if err := os.validate(&order); err != nil {
		return err
	}

	os.persistentStore.Update(func(txn *badger.Txn) error {
		orderBuf, _ := order.XXX_Marshal(nil, true)
		marketKey := fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id)
		idKey := fmt.Sprintf("ID:%s", order.Id)
		partyKey := fmt.Sprintf("P:%s_ID:%s", order.Party, order.Id)
		txn.Set([]byte(marketKey), orderBuf)
		txn.Set([]byte(idKey), []byte(marketKey))
		txn.Set([]byte(partyKey), []byte(marketKey))


		//fmt.Println("saving: ", string(orderBuf))
		//fmt.Println("marketKey: ", marketKey)
		//fmt.Println("partyKey: ", partyKey)
		//fmt.Println("idKey: ", idKey)
		return	nil
	})

	// Update orderBookDepth
	if order.Remaining != uint64(0) {
		os.orderBookDepth.updateWithRemaining(&order)
	}

	//os.queueEvent(order)
	return nil
}

func (os *orderStore) PostBatch(batch []Order) error {
	for idx := range batch {
		if err := os.validate(&batch[idx]); err != nil {
			return err
		}
	}

	wb := os.persistentStore.NewWriteBatch()
	defer wb.Cancel()

	preloadBatch := func() error {
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
	if err := preloadBatch(); err == nil {
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


	//os.queueEvent(order)
	return nil
}


// Put updates an existing order in the memory store.
func (m *orderStore) Put(order Order) error {
	if err := m.validate(&order); err != nil {
		return err
	}

	if !m.partyExists(order.Party) {
		return NotFoundError{fmt.Errorf("could not find party %s", order.Party)}
	}

	if _, exists := m.store.markets[order.Market].orders[order.Id]; !exists {
		return NotFoundError{fmt.Errorf("order not found in memstore: %s", order.Id)}
	}

	remainingDelta := m.store.markets[order.Market].orders[order.Id].order.Remaining - order.Remaining
	m.store.markets[order.Market].orders[order.Id].order = order

	if order.Remaining == uint64(0) || order.Status == msg.Order_Cancelled || order.Status == msg.Order_Expired {
		m.store.markets[order.Market].marketDepth.removeWithRemaining(&order)
	} else {
		m.store.markets[order.Market].marketDepth.updateWithRemainingDelta(&order, remainingDelta)
	}

	m.queueEvent(order)


	m.persistentStore.Update(func(txn *badger.Txn) error {
		orderBuf, _ := order.XXX_Marshal(nil, true)
		marketKey := fmt.Sprintf("M:%s_ID:%s", order.Market, order.Id)
		txn.Set([]byte(marketKey), orderBuf)

		fmt.Println("updating: ", string(orderBuf))
		fmt.Println("marketKey: ", marketKey)
		return	nil
	})

	return nil
}

// Delete removes an order from the memory store.
func (m *orderStore) Delete(order Order) error {
	if err := m.validate(&order); err != nil {
		return err
	}

	if !m.partyExists(order.Party) {
		return NotFoundError{fmt.Errorf("could not find party %s", order.Party)}
	}

	// Remove from orders map
	delete(m.store.markets[order.Market].orders, order.Id)

	// Remove from MARKET ordersByTimestamp
	var pos uint64
	for idx, v := range m.store.markets[order.Market].ordersByTimestamp {
		if v.order.Id == order.Id {
			pos = uint64(idx)
			break
		}
	}
	m.store.markets[order.Market].ordersByTimestamp =
		append(m.store.markets[order.Market].ordersByTimestamp[:pos], m.store.markets[order.Market].ordersByTimestamp[pos+1:]...)

	// Remove from PARTIES ordersByTimestamp
	pos = 0
	for idx, v := range m.store.parties[order.Party].ordersByTimestamp {
		if v.order.Id == order.Id {
			pos = uint64(idx)
			break
		}
	}
	m.store.parties[order.Party].ordersByTimestamp =
		append(m.store.parties[order.Party].ordersByTimestamp[:pos], m.store.parties[order.Party].ordersByTimestamp[pos+1:]...)
	m.store.markets[order.Market].marketDepth.removeWithRemaining(&order)

	return nil
}

// Checks to see if we have a market on the related memory store with given identifier.
// Returns an error if the market cannot be found and nil otherwise.
func (m *orderStore) marketExists(market string) error {
	if !m.store.marketExists(market) {
		return NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	return nil
}

func (m *orderStore) partyExists(party string) bool {
	if m.store.partyExists(party) {
		return true
	}
	return false
}


func (m *orderStore) newMemParty(party string) (*memParty, error) {
	exists := m.partyExists(party)
	if exists {
		return nil, errors.New(fmt.Sprintf("party %s already exists", party))
	}
	memParty := memParty{
		party:             party,
		ordersByTimestamp: []*memOrder{},
		tradesByTimestamp: []*memTrade{},
	}
	m.store.parties[party] = &memParty
	return &memParty, nil
}

func (m *orderStore) validate(order *Order) error {
	if err := m.marketExists(order.Market); err != nil {
		return err
	}

	// more validation here

	return nil
}

// move this to markets store in the future
func (m *orderStore) GetMarkets() ([]string, error) {
	var markets []string
	for key, _ := range m.store.markets {
		markets = append(markets, key)
	}
	return markets, nil
}

// filter results and paginate based on query filters
func (m *orderStore) filterResults(input []*memOrder, queryFilters *filters.OrderQueryFilters) (output []Order, error error) {
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
			if applyOrderFilters(input[i].order, queryFilters) {
				if queryFilters.Skip != nil && *queryFilters.Skip > 0 && skipped < *queryFilters.Skip {
					skipped++
					continue
				}
				output = append(output, input[i].order)
				pos++
			}
		}
	} else {
		// default is descending 'last' n items
		for i := len(input) - 1; i >= 0; i-- {
			if queryFilters.Last != nil && *queryFilters.Last > 0 && pos == *queryFilters.Last {
				break
			}
			if applyOrderFilters(input[i].order, queryFilters) {
				if queryFilters.Skip != nil && *queryFilters.Skip > 0 && skipped < *queryFilters.Skip {
					skipped++
					continue
				}
				output = append(output, input[i].order)
				pos++
			}
		}
	}

	return output, nil
}

// filter results and paginate based on query filters
func (m *orderStore) filterResults2(input []Order, queryFilters *filters.OrderQueryFilters) (output []Order, error error) {
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
			if applyOrderFilters(input[i], queryFilters) {
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
			if applyOrderFilters(input[i], queryFilters) {
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