package datastore

import (
	"errors"
	"fmt"
	"vega/msg"
	"sync"
	"vega/log"
	"vega/filters"
)

// memOrderStore should implement OrderStore interface.
type memOrderStore struct {
	store *MemStore
	subscribers map[uint64] chan<- []Order
	buffer []Order
	subscriberId uint64
	mu sync.Mutex
}

// NewOrderStore initialises a new OrderStore backed by a MemStore.
func NewOrderStore(ms *MemStore) OrderStore {
	return &memOrderStore{store: ms}
}

func (m *memOrderStore) Subscribe(orders chan<- []Order) uint64 {
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

func (m *memOrderStore) Unsubscribe(id uint64) error {
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

func (m *memOrderStore) Notify() error {

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
	for _, sub := range m.subscribers {
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
			log.Debugf("Orders state could not been updated")
		}
	}
	return nil
}

func (m *memOrderStore) queueEvent(o Order) error {
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

func (m *memOrderStore) GetByMarket(market string, queryFilters *filters.OrderQueryFilters) ([]Order, error) {
	if err := m.marketExists(market); err != nil {
		return nil, err
	}
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}
	return m.filterResults(m.store.markets[market].ordersByTimestamp, queryFilters)
}

// Get retrieves an order for a given market and id.
func (m *memOrderStore) GetByMarketAndId(market string, id string) (Order, error) {
	if err := m.marketExists(market); err != nil {
		return Order{}, err
	}
	v, ok := m.store.markets[market].orders[id]
	if !ok {
		return Order{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return v.order, nil
}

func (m *memOrderStore) GetByParty(party string, queryFilters *filters.OrderQueryFilters) ([]Order, error) {
	if !m.partyExists(party) {
		return nil, NotFoundError{fmt.Errorf("could not find party %s", party)}
	}
	if queryFilters == nil {
		queryFilters = &filters.OrderQueryFilters{}
	}
	return m.filterResults(m.store.parties[party].ordersByTimestamp, queryFilters)
}

// Get retrieves an order for a given market and id.
func (m *memOrderStore) GetByPartyAndId(party string, id string) (Order, error) {
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
	return m.store.parties[party].ordersByTimestamp[at].order, nil
}

func (m *memOrderStore) GetByPartyAndReference(party string, reference string) (Order, error) {
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
func (m *memOrderStore) Post(order Order) error {
	if err := m.validate(&order); err != nil {
		return err
	}

	// Order cannot already exist in the store
	if _, exists := m.store.markets[order.Market].orders[order.Id]; exists {
		return fmt.Errorf("order exists in memstore: %s", order.Id)
	}

	// Party 'name' is added on the fly to the parties store
	if !m.partyExists(order.Party) {
		m.newMemParty(order.Party)
		log.Debugf("new party added to store: %v", order.Party)
	}

	newOrder := &memOrder{
		trades: make([]*memTrade, 0),
		order:  order,
	}

	// Insert new order struct into lookup hash table
	m.store.markets[order.Market].orders[order.Id] = newOrder

	// Insert new order into slice of orders ordered by timestamp
	m.store.markets[order.Market].ordersByTimestamp = append(m.store.markets[order.Market].ordersByTimestamp, newOrder)

	// Insert new order into Party map of slices of orders
	m.store.parties[order.Party].ordersByTimestamp = append(m.store.parties[order.Party].ordersByTimestamp, newOrder)

	// Insert into buySideRemainingOrders and sellSideRemainingOrders - these are ordered
	if newOrder.order.Remaining != uint64(0) {
		if newOrder.order.Side == msg.Side_Buy {
			m.store.markets[order.Market].buySideRemainingOrders.insert(&order)
		} else {
			m.store.markets[order.Market].sellSideRemainingOrders.insert(&order)
		}
	}

	m.queueEvent(order)
	return nil
}

// Put updates an existing order in the memory store.
func (m *memOrderStore) Put(order Order) error {
	if err := m.validate(&order); err != nil {
		return err
	}

	if !m.partyExists(order.Party) {
		return NotFoundError{fmt.Errorf("could not find party %s", order.Party)}
	}

	if _, exists := m.store.markets[order.Market].orders[order.Id]; !exists {
		return NotFoundError{fmt.Errorf("order not found in memstore: %s", order.Id)}
	}

	m.store.markets[order.Market].orders[order.Id].order = order

	if order.Remaining == uint64(0) || order.Status == msg.Order_Cancelled {
		// update buySideRemainingOrders sellSideRemainingOrders
		if order.Side == msg.Side_Buy {
			m.store.markets[order.Market].buySideRemainingOrders.remove(&order)
		} else {
			m.store.markets[order.Market].sellSideRemainingOrders.remove(&order)
		}
	} else {
		// update buySideRemainingOrders sellSideRemainingOrders
		if order.Side == msg.Side_Buy {
			m.store.markets[order.Market].buySideRemainingOrders.update(&order)
		} else {
			m.store.markets[order.Market].sellSideRemainingOrders.update(&order)
		}
	}

	m.queueEvent(order)
	return nil
}

// Delete removes an order from the memory store.
func (m *memOrderStore) Delete(order Order) error {
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

	// remove from buySideRemainingOrders sellSideRemainingOrders
	if order.Side == msg.Side_Buy {
		m.store.markets[order.Market].buySideRemainingOrders.remove(&order)
	} else {
		m.store.markets[order.Market].sellSideRemainingOrders.remove(&order)
	}

	return nil
}

// Checks to see if we have a market on the related memory store with given identifier.
// Returns an error if the market cannot be found and nil otherwise.
func (m *memOrderStore) marketExists(market string) error {
	if !m.store.marketExists(market) {
		return NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	return nil
}

func (m *memOrderStore) partyExists(party string) bool {
	if m.store.partyExists(party) {
		return true
	}
	return false
}


func (m *memOrderStore) newMemParty(party string) (*memParty, error) {
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

func (m *memOrderStore) validate(order *Order) error {
	if err := m.marketExists(order.Market); err != nil {
		return err
	}

	// more validation here

	return nil
}

// move this to markets store in the future
func (m *memOrderStore) GetMarkets() ([]string, error) {
	var markets []string
	for key, _ := range m.store.markets {
		markets = append(markets, key)
	}
	return markets, nil
}

// filter results and paginate based on query filters
func (m *memOrderStore) filterResults(input []*memOrder, queryFilters *filters.OrderQueryFilters) (output []Order, error error) {
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
