package datastore

import "fmt"

// memOrderStore should implement OrderStore interface.
type memOrderStore struct {
	store *MemStore
}

// NewOrderStore initialises a new OrderStore backed by a MemStore.
func NewOrderStore(ms *MemStore) OrderStore {
	return &memOrderStore{store: ms}
}

func (store *memOrderStore) GetByMarket(market string, params GetParams) ([]Order, error) {
	if err := store.marketExists(market); err != nil {
		return nil, err
	}

	var (
		pos uint64
		output []Order
	)

	// limit is descending. Get me most recent N orders
	for i := len(store.store.markets[market].ordersByTimestamp) - 1; i >= 0; i-- {
		if params.Limit > 0 && pos == params.Limit {
			break
		}
		// TODO: apply filters
		output = append(output, store.store.markets[market].ordersByTimestamp[i].order)
		pos++
	}
	return output, nil
}

// Get retrieves an order for a given market and id.
func (store *memOrderStore) GetByMarketAndId(market string, id string) (Order, error) {
	if err := store.marketExists(market); err != nil {
		return Order{}, err
	}
	v, ok := store.store.markets[market].orders[id]
	if !ok {
		return Order{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return v.order, nil
}

func (store *memOrderStore) GetByParty(party string, params GetParams) ([]Order, error) {
	if err := store.partyExists(party); err != nil {
		return nil, err
	}

	var (
		pos uint64
		output []Order
	)

	// limit is descending. Get me most recent N orders
	for i := len(store.store.parties[party].ordersByTimestamp) - 1; i >= 0; i-- {
		if params.Limit > 0 && pos == params.Limit {
			break
		}
		// TODO: apply filters
		output = append(output, store.store.parties[party].ordersByTimestamp[i].order)
		pos++
	}
	return output, nil
}

// Get retrieves an order for a given market and id.
func (store *memOrderStore) GetByPartyAndId(party string, id string) (Order, error) {
	if err := store.partyExists(party); err != nil {
		return Order{}, err
	}

	var at = -1
	for idx, order := range store.store.parties[party].ordersByTimestamp {
		if order.order.Id == id {
			at = idx
			break
		}
	}

	if at == -1 {
		return Order{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return store.store.parties[party].ordersByTimestamp[at].order, nil
}


// Post creates a new order in the memory store.
func (store *memOrderStore) Post(order Order) error {
	if err := store.validate(&order); err != nil {
		fmt.Printf("error: %+v\n", err)
		return err
	}

	if _, exists := store.store.markets[order.Market].orders[order.Id]; exists {
		return fmt.Errorf("order exists in memstore: %s", order.Id)
	}

	newOrder := &memOrder{
		trades: make([]*memTrade, 0),
		order:  order,
	}

	// Insert new order struct into lookup hash table
	store.store.markets[order.Market].orders[order.Id] = newOrder

	// Insert new order into slice of orders ordered by timestamp
	store.store.markets[order.Market].ordersByTimestamp = append(store.store.markets[order.Market].ordersByTimestamp, newOrder)


	// Insert new order into Party map of slices of orders
	store.store.parties[order.Party].ordersByTimestamp = append(store.store.parties[order.Party].ordersByTimestamp, newOrder)
	return nil
}

// Put updates an existing order in the memory store.
func (store *memOrderStore) Put(order Order) error {
	if err := store.validate(&order); err != nil {
		fmt.Printf("error: %+v\n", err)
		return err
	}

	if _, exists := store.store.markets[order.Market].orders[order.Id]; !exists {
		return fmt.Errorf("order not found in memstore: %s", order.Id)
	}

	store.store.markets[order.Market].orders[order.Id].order = order
	return nil
}

// Delete removes an order from the memory store.
func (store *memOrderStore) Delete(order Order) error {
	if err := store.validate(&order); err != nil {
		fmt.Printf("error: %+v\n", err)
		return err
	}

	// Remove from orders map
	delete(store.store.markets[order.Market].orders, order.Id)

	// Remove from MARKET ordersByTimestamp
	var pos uint64
	for idx, v := range store.store.markets[order.Market].ordersByTimestamp {
		if v.order.Id == order.Id {
			pos = uint64(idx)
			break
		}
	}
	store.store.markets[order.Market].ordersByTimestamp =
		append(store.store.markets[order.Market].ordersByTimestamp[:pos], store.store.markets[order.Market].ordersByTimestamp[pos+1:]...)

	// Remove from PARTIES ordersByTimestamp
	pos = 0
	for idx, v := range store.store.parties[order.Party].ordersByTimestamp {
		if v.order.Id == order.Id {
			pos = uint64(idx)
			break
		}
	}
	store.store.parties[order.Party].ordersByTimestamp =
		append(store.store.parties[order.Party].ordersByTimestamp[:pos], store.store.parties[order.Party].ordersByTimestamp[pos+1:]...)

	return nil
}

// Checks to see if we have a market on the related memory store with given identifier.
// Returns an error if the market cannot be found and nil otherwise.
func (store *memOrderStore) marketExists(market string) error {
	if !store.store.marketExists(market) {
		return NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	return nil
}

func (store *memOrderStore) partyExists(party string) error {
	if !store.store.partyExists(party) {
		return NotFoundError{fmt.Errorf("could not find party %s", party)}
	}
	return nil
}

func (store *memOrderStore) validate(order *Order) error {
	if err := store.marketExists(order.Market); err != nil {
		return err
	}

	if err := store.partyExists(order.Party); err != nil {
		return err
	}

	return nil
}

// move this to markets store in the future
func (store *memOrderStore) GetMarkets() ([]string, error) {
	var markets []string
	for key, _ := range store.store.markets {
		markets = append(markets, key)
	}
	return markets, nil
}