package datastore

import (
	"fmt"
)

type memMarket struct {
	name   string
	orders map[string]*memOrder
	trades map[string]*memTrade
}

// In memory order struct keeps an internal map of pointers to trades for an order.
type memOrder struct {
	order  *Order
	trades []*memTrade
}

// OrderStore implements storage.OrderStore.
type memOrderStore struct {
	store *MemStore
}

// In memory trade struct keeps a pointer to the related order.
type memTrade struct {
	trade *Trade
	order *memOrder
}

// tradeStore implements datastore.TradeStore.
type memTradeStore struct {
	store *MemStore
}

type MemStore struct {
	// markets is the top level structure holding trades and orders.
	markets map[string]*memMarket
}

// NewMemStore creates an instance of the ram based data store.
// This store is simply backed by maps/slices for trades and orders.
func NewMemStore(markets []string) MemStore {
	memMarkets := make(map[string]*memMarket, len(markets))
	for _, name := range markets {
		memMarket := memMarket{
			name:   name,
			orders: map[string]*memOrder{},
			trades: map[string]*memTrade{},
		}
		memMarkets[name] = &memMarket
	}
	return MemStore{
		markets: memMarkets,
	}
}

func NewTradeStore(ms *MemStore) TradeStore {
	return &memTradeStore{store: ms}
}

func NewOrderStore(ms *MemStore) OrderStore {
	return &memOrderStore{store: ms}
}

func (ms *MemStore) marketExists(market string) bool {
	if _, exists := ms.markets[market]; exists {
		return true
	}
	return false
}

func (t *memOrderStore) All(market string) ([]*Order, error) {
	if !t.store.marketExists(market) {
		return nil, NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	orders := make([]*Order, 0)
	for _, value := range t.store.markets[market].orders {
		orders = append(orders, value.order)
	}
	return orders, nil
}

// Get implements datastore.OrderStore.Get().
func (t *memOrderStore) Get(market string, id string) (*Order, error) {
	if !t.store.marketExists(market) {
		return nil, NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	v, ok := t.store.markets[market].orders[id]
	if !ok {
		return nil, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return v.order, nil
}

// Put implements storage.OrderStore.Put().
func (t *memOrderStore) Put(or *Order) error {

	// todo validation of incoming order
	//	if err := or.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}
	if !t.store.marketExists(or.Market) {
		return NotFoundError{fmt.Errorf("could not find market %s", or.Market)}
	}
	if _, exists := t.store.markets[or.Market].orders[or.ID]; exists {
		fmt.Println("Updating order with ID ", or.ID)

		t.store.markets[or.Market].orders[or.ID].order = or
	} else {
		fmt.Println("Adding new order with ID ", or.ID)

		order := &memOrder{
			trades: make([]*memTrade, 0),
			order:  or,
		}
		t.store.markets[or.Market].orders[or.ID] = order
	}
	return nil
}

// Delete implements storage.TradeStore.Delete().
func (t *memOrderStore) Delete(or *Order) error {
	delete(t.store.markets[or.Market].orders, or.ID)
	return nil
}

// Get implements datastore.TradeStore.Get().
func (t *memTradeStore) Get(market string, id string) (*Trade, error) {
	v, ok := t.store.markets[market].trades[id]
	if !ok {
		return nil, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return v.trade, nil
}

// FindByOrderId retrieves all trades for a given order id.
func (t *memTradeStore) FindByOrderID(market string, orderID string) ([]*Trade, error) {

	order := t.store.markets[market].orders[orderID]
	if order == nil {
		return nil, fmt.Errorf("order not found in memstore: %s", orderID)
	} else {
		trades := make([]*Trade, 0)
		for _, v := range order.trades {
			trades = append(trades, v.trade)
		}
		return trades, nil
	}

	//trades := make([]*Trade, 0)
	//for k, v := range t.store.trades {
	//	fmt.Printf("key[%s] value[%v]\n", k, v)
	//	if v.trade.OrderID == orderID {
	//		trades = append(trades, v.trade)
	//	}
	//}
	//return trades, nil
}

// Put implements storage.TradeStore.Put().
func (t *memTradeStore) Put(tr *Trade) error {
	//todo validation of incoming trade
	// if err := tr.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}
	if o, exists := t.store.markets[tr.Market].orders[tr.OrderID]; exists {
		trade := &memTrade{
			trade: tr,
			order: o,
		}
		// todo check if trade with ID already exists
		t.store.markets[tr.Market].trades[tr.ID] = trade
		o.trades = append(o.trades, trade)
		return nil
	} else {
		return fmt.Errorf("trade order not found in memstore: %s", tr.OrderID)
	}
}

// Delete implements storage.TradeStore.Delete().
func (t *memTradeStore) Delete(tr *Trade) error {
	delete(t.store.markets[tr.Market].trades, tr.ID)
	return nil
}
