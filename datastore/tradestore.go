package datastore

import (
	"errors"
	"fmt"
	"sync"
	"vega/log"
)

// memTradeStore should implement TradeStore interface.
type memTradeStore struct {
	store *MemStore
	subscribers map[uint64] chan<- []Trade
	buffer []Trade
	subscriberId uint64
	mu sync.Mutex
}

// NewTradeStore initialises a new TradeStore backed by a MemStore.
func NewTradeStore(ms *MemStore) TradeStore {
	return &memTradeStore{store: ms}
}

func (m *memTradeStore) Subscribe(orders chan<- []Trade) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil {
		log.Debugf("TradeStore -> Subscribe: Creating subscriber chan map")
		m.subscribers = make(map[uint64] chan<- []Trade)
	}

	m.subscriberId = m.subscriberId+1
	m.subscribers[m.subscriberId] = orders
	log.Debugf("TradeStore -> Subscribe: Trade subscriber added: %d", m.subscriberId)
	return m.subscriberId
}

func (m *memTradeStore) Unsubscribe(id uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil || len(m.subscribers) == 0 {
		log.Debugf("TradeStore -> Unsubscribe: No subscribers connected")
		return nil
	}

	if _, exists := m.subscribers[id]; exists {
		delete(m.subscribers, id)
		log.Debugf("TradeStore -> Unsubscribe: Subscriber removed: %v", id)
		return nil
	}
	return errors.New(fmt.Sprintf("TradeStore subscriber does not exist with id: %d", id))
}

func (m *memTradeStore) Notify() error {

	if m.subscribers == nil || len(m.subscribers) == 0 {
		log.Debugf("TradeStore -> Notify: No subscribers connected")
		return nil
	}

	if m.buffer == nil || len(m.buffer) == 0 {
		// Only publish when we have items
		log.Debugf("TradeStore -> Notify: No trades in buffer")
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

func (m *memTradeStore) queueEvent(t Trade) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.subscribers == nil || len(m.subscribers) == 0 {
		log.Debugf("TradeStore -> queueEvent: No subscribers connected")
		return nil
	}

	if m.buffer == nil {
		m.buffer = make([]Trade, 0)
	}

	log.Debugf("TradeStore -> queueEvent: Adding trade to buffer: %+v", t)
	m.buffer = append(m.buffer, t)
	return nil
}

// GetByMarket retrieves all trades for a given market.
func (store *memTradeStore) GetByMarket(market string, params GetTradeParams) ([]Trade, error) {
	if err := store.marketExists(market); err != nil {
		return nil, err
	}

	var (
		pos    uint64
		output []Trade
	)

	// limit is descending. Get me most recent N orders
	for i := len(store.store.markets[market].tradesByTimestamp) - 1; i >= 0; i-- {
		if params.Limit > 0 && pos == params.Limit {
			break
		}
		if applyTradeFilter(store.store.markets[market].tradesByTimestamp[i].trade, params) {
			output = append(output, store.store.markets[market].tradesByTimestamp[i].trade)
			pos++
		}
	}
	return output, nil
}

// GetByMarketAndId retrieves a trade for a given id.
func (store *memTradeStore) GetByMarketAndId(market string, id string) (Trade, error) {
	if err := store.marketExists(market); err != nil {
		return Trade{}, err
	}
	v, ok := store.store.markets[market].trades[id]
	if !ok {
		return Trade{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return v.trade, nil
}

// GetByPart retrieves all trades for a given party.
func (store *memTradeStore) GetByParty(party string, params GetTradeParams) ([]Trade, error) {
	if err := store.partyExists(party); err != nil {
		return nil, err
	}

	var (
		pos    uint64
		output []Trade
	)

	// limit is descending. Get me most recent N orders
	for i := len(store.store.parties[party].tradesByTimestamp) - 1; i >= 0; i-- {
		if params.Limit > 0 && pos == params.Limit {
			break
		}
		if applyTradeFilter(store.store.parties[party].tradesByTimestamp[i].trade, params) {
			output = append(output, store.store.parties[party].tradesByTimestamp[i].trade)
			pos++
		}
	}
	return output, nil
}

// GetByPartyAndId retrieves a trade for a given id.
func (store *memTradeStore) GetByPartyAndId(party string, id string) (Trade, error) {
	if err := store.partyExists(party); err != nil {
		return Trade{}, err
	}

	var at = -1
	for idx, trade := range store.store.parties[party].tradesByTimestamp {
		if trade.trade.Id == id {
			at = idx
			break
		}
	}

	if at == -1 {
		return Trade{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return store.store.parties[party].tradesByTimestamp[at].trade, nil
}


// Post creates a new trade in the memory store.
func (store *memTradeStore) Post(trade Trade) error {
	if err := store.validate(&trade); err != nil {
		fmt.Printf("error: %+v\n", err)
		return err
	}

	if _, exists := store.store.markets[trade.Market].trades[trade.Id]; exists {
		return fmt.Errorf("trade exists in memstore: %s", trade.Id)
	}

	// check if passive and aggressive orders exist in the order store

	// if passive exists
	aggressiveOrder, aggressiveExists := store.store.markets[trade.Market].orders[trade.AggressiveOrderId]
	if !aggressiveExists {
		return fmt.Errorf("aggressive order for trade not found in memstore: %s", trade.AggressiveOrderId)
	}

	// if passive exists
	passiveOrder, passiveExists := store.store.markets[trade.Market].orders[trade.PassiveOrderId]
	if !passiveExists {
		return fmt.Errorf("passive order for trade not found in memstore: %s", trade.PassiveOrderId)
	}

	newTrade := &memTrade{
		trade:      trade,
		aggressive: aggressiveOrder,
		passive:    passiveOrder,
	}
	
	// Add new trade to trades hashtable & queue in buffer to notify observers
	store.store.markets[trade.Market].trades[trade.Id] = newTrade
	store.queueEvent(trade)

	// append trade to aggressive and passive order
	aggressiveOrder.trades = append(aggressiveOrder.trades, newTrade)
	passiveOrder.trades = append(passiveOrder.trades, newTrade)

	// update tradesByTimestamp for MARKETS
	store.store.markets[trade.Market].tradesByTimestamp = append(store.store.markets[trade.Market].tradesByTimestamp, newTrade)

	// update party for both aggressive and passive parties
	store.store.parties[newTrade.aggressive.order.Party].tradesByTimestamp = append(store.store.parties[newTrade.aggressive.order.Party].tradesByTimestamp, newTrade)
	store.store.parties[newTrade.passive.order.Party].tradesByTimestamp = append(store.store.parties[newTrade.passive.order.Party].tradesByTimestamp, newTrade)
	return nil
}

// Put updates an existing trade in the store.
func (store *memTradeStore) Put(trade Trade) error {
	if err := store.validate(&trade); err != nil {
		fmt.Printf("error: %+v\n", err)
		return err
	}

	if err := store.marketExists(trade.Market); err != nil {
		return err
	}

	if _, exists := store.store.markets[trade.Market].trades[trade.Id]; !exists {
		return fmt.Errorf("trade not found in memstore: %s", trade.Id)
	}
	// Perform the update & queue in buffer to notify observers
	store.store.markets[trade.Market].trades[trade.Id].trade = trade
	store.queueEvent(trade)
	return nil
}

// Removes trade from the store.
func (store *memTradeStore) Delete(trade Trade) error {
	if err := store.validate(&trade); err != nil {
		fmt.Printf("error: %+v\n", err)
		return err
	}

	if err := store.partyExists(trade.Seller); err != nil {
		return err
	}

	// Remove from tradesByTimestamp
	var pos uint64
	for idx, v := range store.store.markets[trade.Market].tradesByTimestamp {
		if v.trade.Id == trade.Id {
			pos = uint64(idx)
			break
		}
	}
	store.store.markets[trade.Market].tradesByTimestamp =
		append(store.store.markets[trade.Market].tradesByTimestamp[:pos], store.store.markets[trade.Market].tradesByTimestamp[pos+1:]...)

	// Remove from PARTIES tradesByTimestamp for BUYER
	pos = 0
	for idx, v := range store.store.parties[trade.Buyer].tradesByTimestamp {
		if v.trade.Id == trade.Id {
			pos = uint64(idx)
			break
		}
	}
	store.store.parties[trade.Buyer].tradesByTimestamp =
		append(store.store.parties[trade.Buyer].tradesByTimestamp[:pos], store.store.parties[trade.Buyer].tradesByTimestamp[pos+1:]...)

	// Remove from PARTIES tradesByTimestamp for SELLER
	pos = 0
	for idx, v := range store.store.parties[trade.Seller].tradesByTimestamp {
		if v.trade.Id == trade.Id {
			pos = uint64(idx)
			break
		}
	}
	store.store.parties[trade.Seller].tradesByTimestamp =
		append(store.store.parties[trade.Seller].tradesByTimestamp[:pos], store.store.parties[trade.Seller].tradesByTimestamp[pos+1:]...)

	delete(store.store.markets[trade.Market].trades, trade.Id)
	return nil
}

// Checks to see if we have a market on the related memory store with given identifier.
// Returns an error if the market cannot be found and nil otherwise.
func (store *memTradeStore) marketExists(market string) error {
	if !store.store.marketExists(market) {
		return NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	return nil
}

func (store *memTradeStore) partyExists(party string) error {
	if !store.store.partyExists(party) {
		memParty := memParty{
			party:   party,
			ordersByTimestamp: []*memOrder{},
			tradesByTimestamp: []*memTrade{},
		}
		store.store.parties[party] = &memParty
		return nil
	}
	return nil
}

func (store *memTradeStore) validate(trade *Trade) error {
	if err := store.marketExists(trade.Market); err != nil {
		return err
	}

	if err := store.partyExists(trade.Buyer); err != nil {
		return err
	}

	if err := store.partyExists(trade.Seller); err != nil {
		return err
	}

	return nil
}

func (store *memTradeStore) GetMarkPrice(market string) (uint64, error) {
	recentTrade, err := store.GetByMarket(market, GetTradeParams{Limit:1})
	if err != nil {
		return 0, err
	}
	if len(recentTrade) == 0 {
		return 0, fmt.Errorf("NO TRADES")
	}

	return recentTrade[0].Price, nil
}

