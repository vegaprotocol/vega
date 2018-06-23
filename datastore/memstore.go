package datastore

import (
	"fmt"
)

// In memory order struct keeps an internal map of pointers to trades for an order.
type memOrder struct {
	order *Order
	trades map[string]*memTrade
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
	orders map[string]*memOrder
	trades map[string]*memTrade
}

// NewMemStore creates an instance of the ram based data store.
// This store is simply backed by maps/slices for trades and orders.
func NewMemStore() MemStore {
	return MemStore{
		orders: map[string]*memOrder{},
		trades: map[string]*memTrade{},
	}
}

func NewTradeStore(ms *MemStore) TradeStore {
	return &memTradeStore{store: ms}
}

func NewOrderStore(ms *MemStore) OrderStore {
	return &memOrderStore{store: ms}
}

// Get implements datastore.OrderStore.Get().
func (t *memOrderStore) Get(id string) (*Order, error) {
	v, ok := t.store.orders[id]
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
	if _, exists := t.store.orders[or.ID]; exists {
		t.store.orders[or.ID].order = or
	} else {
		order := &memOrder {
			trades: make(map[string]*memTrade, 0),
			order: or,
		}
		t.store.orders[or.ID] = order
	}
	return nil
}

// Delete implements storage.TradeStore.Delete().
func (t *memOrderStore) Delete(or *Order) error {
	delete(t.store.orders, or.ID)
	return nil
}


// Get implements datastore.TradeStore.Get().
func (t *memTradeStore) Get(id string) (*Trade, error) {
	v, ok := t.store.trades[id]
	if !ok {
		return nil, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return v.trade, nil
}

// FindByOrderId retrieves all trades for a given order id.
func (t *memTradeStore) FindByOrderId(orderId string) ([]*Trade, error) {
	trades := make([]*Trade, 0)

	for k, v := range t.store.trades {
		fmt.Printf("key[%s] value[%v]\n", k, v)
		if v.trade.OrderID == orderId {
			trades = append(trades, v.trade)
		}
	}
	return trades, nil
}

// Put implements storage.TradeStore.Put().
func (t *memTradeStore) Put(tr *Trade) error {
	//todo validation of incoming trade 
	// if err := tr.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}
	if o, exists := t.store.orders[tr.OrderID]; exists {
		trade := &memTrade {
			trade: tr,
			order: o,
		}
		// todo check if trade with ID already exists
		t.store.trades[tr.ID] = trade
		return nil
	} else {
		return fmt.Errorf("trade order not found in memstore: %s", tr.OrderID)
	}
}

// Delete implements storage.TradeStore.Delete().
func (t *memTradeStore) Delete(tr *Trade) error {
	delete(t.store.trades, tr.ID)
	return nil
}
