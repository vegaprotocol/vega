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

func (t *memOrderStore) GetAll(market string, params GetParams) ([]*Order, error) {
	if !t.store.marketExists(market) {
		return nil, NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	pos := uint64(0)
	orders := make([]*Order, 0)
	for _, value := range t.store.markets[market].orders {
		orders = append(orders, value.order)
		if params.Limit > 0 && pos == params.Limit {
			break
		}
		pos++
	}
	return orders, nil
}

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

func (t *memOrderStore) Put(or *Order) error {
	// todo validation of incoming order
	//	if err := or.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}
	if !t.store.marketExists(or.Market) {
		return NotFoundError{fmt.Errorf("could not find market %s", or.Market)}
	}
	if _, exists := t.store.markets[or.Market].orders[or.Id]; exists {
		fmt.Println("Updating order with ID ", or.Id)
		t.store.markets[or.Market].orders[or.Id].order = or
	} else {
		return fmt.Errorf("order not found in memstore: %s", or.Id)
	}
	return nil
}

func (t *memOrderStore) Post(or *Order) error {
	// todo validation of incoming order
	//	if err := or.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}
	if !t.store.marketExists(or.Market) {
		return NotFoundError{fmt.Errorf("could not find market %s", or.Market)}
	}
	if _, exists := t.store.markets[or.Market].orders[or.Id]; exists {
		return fmt.Errorf("order exists in memstore: %s", or.Id)
	} else {
		fmt.Println("Adding new order with ID ", or.Id)
		order := &memOrder{
			trades: make([]*memTrade, 0),
			order:  or,
		}
		t.store.markets[or.Market].orders[or.Id] = order
	}
	return nil
}

func (t *memOrderStore) Delete(or *Order) error {
	delete(t.store.markets[or.Market].orders, or.Id)
	return nil
}

func (t *memTradeStore) GetAll(market string, params GetParams) ([]*Trade, error) {
	if !t.store.marketExists(market) {
		return nil, NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	pos := uint64(0)
	trades := make([]*Trade, 0)
	for _, value := range t.store.markets[market].trades {
		trades = append(trades, value.trade)
		if params.Limit > 0 && pos == params.Limit {
			break
		}
		pos++
	}
	return trades, nil
}

func (t *memTradeStore) Get(market string, id string) (*Trade, error) {
	v, ok := t.store.markets[market].trades[id]
	if !ok {
		return nil, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return v.trade, nil
}


// GetByOrderId retrieves all trades for a given order id.
func (t *memTradeStore) GetByOrderId(market string, orderId string, params GetParams) ([]*Trade, error) {

	order := t.store.markets[market].orders[orderId]
	if order == nil {
		return nil, fmt.Errorf("order not found in memstore: %s", orderId)
	} else {
		pos := uint64(0)
		trades := make([]*Trade, 0)
		for _, v := range order.trades {
			trades = append(trades, v.trade)
			if params.Limit > 0 && pos == params.Limit {
				break
			}
			pos++
		}
		return trades, nil
	}
}

func (t *memTradeStore) Post(tr *Trade) error {
	//todo validation of incoming trade
	// if err := tr.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}
	if o, exists := t.store.markets[tr.Market].orders[tr.OrderId]; exists {
		trade := &memTrade{
			trade: tr,
			order: o,
		}
		if _, exists := t.store.markets[tr.Market].trades[tr.Id]; exists {
			return fmt.Errorf("trade exists in memstore: %s", tr.Id)
		} else {
			// Map new trade to memstore and append trade to order
			t.store.markets[tr.Market].trades[tr.Id] = trade
			o.trades = append(o.trades, trade)
		}
		return nil
	} else {
		return fmt.Errorf("related order for trade not found in memstore: %s", tr.OrderId)
	}
}

func (t *memTradeStore) Put(tr *Trade) error {
	//todo validation of incoming trade
	// if err := tr.Validate(); err != nil {
	//		return fmt.Errorf("cannot store record: %s", err)
	//	}
	if o, exists := t.store.markets[tr.Market].orders[tr.OrderId]; exists {
		trade := &memTrade{
			trade: tr,
			order: o,
		}
		if _, exists := t.store.markets[tr.Market].trades[tr.Id]; exists {
			// Perform the update
			t.store.markets[tr.Market].trades[tr.Id] = trade
		} else {
			return fmt.Errorf("trade not found in memstore: %s", tr.Id)
		}
		//o.trades = append(o.trades, trade)
		return nil
	} else {
		return fmt.Errorf("related order for trade not found in memstore: %s", tr.OrderId)
	}
}

func (t *memTradeStore) Delete(tr *Trade) error {
	delete(t.store.markets[tr.Market].trades, tr.Id)
	return nil
}
