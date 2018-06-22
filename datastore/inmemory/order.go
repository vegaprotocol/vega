package inmemory

import (
	"vega/datastore"
	"fmt"
)

// In memory order struct keeps an internal map of pointers to trades for an order
type order struct {
	order *datastore.Order
	trades map[string]*trade
}

// OrderStore implements storage.OrderStore.
type orderStore struct {
	store *MemStore
}

// Get implements datastore.OrderStore.Get().
func (t *orderStore) Get(id string) (*datastore.Order, error) {
	v, ok := t.store.orders[id]
	if !ok {
		return nil, datastore.NotFoundError{fmt.Errorf("could not find id %d", id)}
	}
	return v.order, nil
}

// Put implements storage.OrderStore.Put().
func (t *orderStore) Put(or *datastore.Order) error {
	// todo validation
	//	if err := r.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}
	if _, exists := t.store.orders[or.ID]; exists {
		t.store.orders[or.ID].order = or
	} else {
		order := &order {
			trades: make(map[string]*trade, 0),
			order: or,
		}
		t.store.orders[or.ID] = order
	}
	return nil
}

// Delete implements storage.TradeStore.Delete().
func (t *orderStore) Delete(r *datastore.Order) error {
	delete(t.store.orders, r.ID)
	return nil
}

