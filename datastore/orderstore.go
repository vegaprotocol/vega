package datastore

import (
	"errors"
	"fmt"
	"vega/msg"
	"sync"
	"vega/log"
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
		log.Debugf("OrderStore -> Unsubscribe: No subscribers available")
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
	for _, sub := range m.subscribers {
		sub <- items
	}
	
	return nil
}

func (m *memOrderStore) queueEvent(o Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil || len(m.subscribers) == 0 {
		log.Debugf("Orders->queueEvent: No subscribers available")
		return nil
	}

	if m.buffer == nil {
		m.buffer = make([]Order, 0)
	}

	log.Debugf("Orders->queueEvent: Adding order to buffer: %+v", o)
	m.buffer = append(m.buffer, o)
	return nil
}


func (m *memOrderStore) GetByMarket(market string, params GetOrderParams) ([]Order, error) {
	if err := m.marketExists(market); err != nil {
		return nil, err
	}

	var (
		pos    uint64
		output []Order
	)

	// limit is descending. Get me most recent N orders
	for i := len(m.store.markets[market].ordersByTimestamp) - 1; i >= 0; i-- {
		if params.Limit > 0 && pos == params.Limit {
			break
		}
		if applyOrderFilter(m.store.markets[market].ordersByTimestamp[i].order, params) {
			output = append(output, m.store.markets[market].ordersByTimestamp[i].order)
			pos++
		}
	}
	return output, nil
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

func (m *memOrderStore) GetByParty(party string, params GetOrderParams) ([]Order, error) {
	if err := m.partyExists(party); err != nil {
		return nil, err
	}

	var (
		pos    uint64
		output []Order
	)

	// limit is descending. Get me most recent N orders
	for i := len(m.store.parties[party].ordersByTimestamp) - 1; i >= 0; i-- {
		if params.Limit > 0 && pos == params.Limit {
			break
		}
		if applyOrderFilter(m.store.parties[party].ordersByTimestamp[i].order, params) {
			output = append(output, m.store.parties[party].ordersByTimestamp[i].order)
			pos++
		}
	}
	return output, nil
}

// Get retrieves an order for a given market and id.
func (m *memOrderStore) GetByPartyAndId(party string, id string) (Order, error) {
	if err := m.partyExists(party); err != nil {
		return Order{}, err
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

// Post creates a new order in the memory store.
func (m *memOrderStore) Post(order Order) error {
	if err := m.validate(&order); err != nil {
		fmt.Printf("error: %+v\n", err)
		return err
	}

	if _, exists := m.store.markets[order.Market].orders[order.Id]; exists {
		return fmt.Errorf("order exists in memstore: %s", order.Id)
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
		fmt.Printf("error: %+v\n", err)
		return err
	}

	if _, exists := m.store.markets[order.Market].orders[order.Id]; !exists {
		return fmt.Errorf("order not found in memstore: %s", order.Id)
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
		fmt.Printf("error: %+v\n", err)
		return err
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

	m.queueEvent(order)
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

func (m *memOrderStore) partyExists(party string) error {
	if !m.store.partyExists(party) {
		memParty := memParty{
			party:             party,
			ordersByTimestamp: []*memOrder{},
			tradesByTimestamp: []*memTrade{},
		}
		m.store.parties[party] = &memParty
		return nil
	}
	return nil
}

func (m *memOrderStore) validate(order *Order) error {
	if err := m.marketExists(order.Market); err != nil {
		return err
	}

	if err := m.partyExists(order.Party); err != nil {
		return err
	}

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
