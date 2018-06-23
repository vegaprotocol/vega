package datastore

import (
	"fmt"
)

type MemStore struct {
	orders map[string]*memOrder
	trades map[string]*memTrade
}

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

// In memory order struct keeps an internal map of pointers to trades for an order
type memOrder struct {
	order *Order
	trades map[string]*memTrade
}

// OrderStore implements storage.OrderStore.
type memOrderStore struct {
	store *MemStore
}

// Get implements datastore.OrderStore.Get().
func (t *memOrderStore) Get(id string) (*Order, error) {
	v, ok := t.store.orders[id]
	if !ok {
		return nil, NotFoundError{fmt.Errorf("could not find id %d", id)}
	}
	return v.order, nil
}

// Put implements storage.OrderStore.Put().
func (t *memOrderStore) Put(or *Order) error {
	// todo validation
	//	if err := r.Validate(); err != nil {
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
func (t *memOrderStore) Delete(r *Order) error {
	delete(t.store.orders, r.ID)
	return nil
}

// In memory trade struct keeps a pointer to the related order
type memTrade struct {
	trade *Trade
	order *memOrder
}

// tradeStore implements datastore.TradeStore.
type memTradeStore struct {
	store *MemStore
}

// Get implements datastore.TradeStore.Get().
func (t *memTradeStore) Get(id string) (*Trade, error) {
	v, ok := t.store.trades[id]
	if !ok {
		return nil, NotFoundError{fmt.Errorf("could not find id %d", id)}
	}
	return v.trade, nil
}

// FindByOrderId retrieves all trades for a given order id.
func (t *memTradeStore) FindByOrderId(orderId string) ([]*Trade, error) {
	trades := make([]*Trade, 0)

	for k, v := range t.store.trades {
		fmt.Printf("key[%s] value[%s]\n", k, v)
		if v.trade.OrderID == orderId {
			trades = append(trades, v.trade)
		}
	}
	return trades, nil
}

// Put implements storage.TradeStore.Put().
func (t *memTradeStore) Put(tr *Trade) error {
	//todo validation
	// if err := r.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}

	// try and find order

	if o, exists := t.store.orders[tr.OrderID]; exists {
		// Look up order
		trade := &memTrade {
			trade: tr,
			order: o,
		}
		t.store.trades[tr.ID] = trade
		return nil
	} else {
		return fmt.Errorf("trade order not found in memstore: %s", tr.OrderID)
	}
}

// Delete implements storage.TradeStore.Delete().
func (t *memTradeStore) Delete(r *Trade) error {
	delete(t.store.trades, r.ID)
	return nil
}
