package inmemory

import (
	"vega/datastore"
	"fmt"
)

// In memory trade struct keeps a pointer to the related order
type trade struct {
	trade *datastore.Trade
	order *order
}

// tradeStore implements datastore.TradeStore.
type tradeStore struct {
	store *MemStore
}

// Get implements datastore.TradeStore.Get().
func (t *tradeStore) Get(id string) (*datastore.Trade, error) {
	v, ok := t.store.trades[id]
	if !ok {
		return nil, datastore.NotFoundError{fmt.Errorf("could not find id %d", id)}
	}
	return v.trade, nil
}

// FindByOrderId retrieves all trades for a given order id.
func (t *tradeStore) FindByOrderId(orderId string) ([]*datastore.Trade, error) {
	trades := make([]*datastore.Trade, 0)

	for k, v := range t.store.trades {
		fmt.Printf("key[%s] value[%s]\n", k, v)
		if v.trade.OrderID == orderId {
			trades = append(trades, v.trade)
		}
	}
	return trades, nil
}

// Put implements storage.TradeStore.Put().
func (t *tradeStore) Put(tr *datastore.Trade) error {
	//todo validation
	// if err := r.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}

	// try and find order 

	if o, exists := t.store.orders[tr.OrderID]; exists {
		// Look up order
		trade := &trade {
			trade: tr,
			order: o,
		}
		t.store.trades[tr.ID] = trade
		return nil
	} else {
		return fmt.Errorf("trade order not found in memstore: %s", r.OrderID)
	}
}

// Delete implements storage.TradeStore.Delete().
func (t *tradeStore) Delete(r *datastore.Trade) error {
	delete(t.store.trades, r.ID)
	return nil
}
