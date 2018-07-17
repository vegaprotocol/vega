package datastore

// MemStore is a RAM based top level structure to hold information about all markets.
// It is initialised by calling NewMemStore with a list of markets.
type MemStore struct {
	markets map[string]*memMarket
	parties map[string]*memParty
}

// NewMemStore creates an instance of the ram based data store.
// This store is simply backed by maps/slices for trades and orders.
func NewMemStore(markets, parties []string) MemStore {
	memMarkets := make(map[string]*memMarket, len(markets))
	for _, name := range markets {
		memMarket := memMarket{
			name:   name,
			orders: map[string]*memOrder{},
			trades: map[string]*memTrade{},
		}
		memMarkets[name] = &memMarket
	}

	memParties :=  make(map[string]*memParty, len(parties))
	for _, name := range parties {
		memParty := memParty{
			party:   name,
			ordersByTimestamp: []*memOrder{},
			tradesByTimestamp: []*memTrade{},
		}
		memParties[name] = &memParty
	}

	return MemStore{
		markets: memMarkets,
		parties: memParties,
	}
}

// memMarket should keep track of the trades/orders operating on a Market.
type memMarket struct {
	name              string
	ordersByTimestamp []*memOrder
	tradesByTimestamp []*memTrade
	orders            map[string]*memOrder
	trades            map[string]*memTrade
}

// memParty should keep track of the trades/orders per Party.
type memParty struct {
	party              string
	ordersByTimestamp []*memOrder
	tradesByTimestamp []*memTrade
}

// In memory order struct keeps an internal map of pointers to trades for an order.
type memOrder struct {
	order  Order
	trades []*memTrade
}

func (mo *memOrder) String() string {
	return "memOrder::order-id=" + mo.order.Id
}

// In memory trade struct keeps a pointer to the related order.
type memTrade struct {
	trade      Trade
	aggressive *memOrder
	passive    *memOrder
}

func (mt *memTrade) String() string {
	return "memTrade::trade-id=" + mt.trade.Id
}

// Helper function to check if a market exists within the memory store.
func (ms *MemStore) marketExists(market string) bool {
	if _, exists := ms.markets[market]; exists {
		return true
	}
	return false
}

// Helper function to check if a party exists within the memory store.
func (ms *MemStore) partyExists(party string) bool {
	if _, exists := ms.parties[party]; exists {
		return true
	}
	return false
}
