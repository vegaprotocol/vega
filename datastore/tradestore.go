package datastore

import (
	"errors"
	"fmt"
	"sync"
	"vega/log"
	"vega/filters"
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

func (ts *memTradeStore) Subscribe(orders chan<- []Trade) uint64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.subscribers == nil {
		log.Debugf("TradeStore -> Subscribe: Creating subscriber chan map")
		ts.subscribers = make(map[uint64] chan<- []Trade)
	}

	ts.subscriberId = ts.subscriberId+1
	ts.subscribers[ts.subscriberId] = orders
	log.Debugf("TradeStore -> Subscribe: Trade subscriber added: %d", ts.subscriberId)
	return ts.subscriberId
}

func (ts *memTradeStore) Unsubscribe(id uint64) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.subscribers == nil || len(ts.subscribers) == 0 {
		log.Debugf("TradeStore -> Unsubscribe: No subscribers connected")
		return nil
	}

	if _, exists := ts.subscribers[id]; exists {
		delete(ts.subscribers, id)
		log.Debugf("TradeStore -> Unsubscribe: Subscriber removed: %v", id)
		return nil
	}
	return errors.New(fmt.Sprintf("TradeStore subscriber does not exist with id: %d", id))
}

func (ts *memTradeStore) Notify() error {

	if ts.subscribers == nil || len(ts.subscribers) == 0 {
		log.Debugf("TradeStore -> Notify: No subscribers connected")
		return nil
	}

	if ts.buffer == nil || len(ts.buffer) == 0 {
		// Only publish when we have items
		log.Debugf("TradeStore -> Notify: No trades in buffer")
		return nil
	}

	ts.mu.Lock()
	items := ts.buffer
	ts.buffer = nil
	ts.mu.Unlock()

	// iterate over items in buffer and push to observers
	for _, sub := range ts.subscribers {
		sub <- items
	}

	return nil
}

func (ts *memTradeStore) queueEvent(t Trade) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.subscribers == nil || len(ts.subscribers) == 0 {
		log.Debugf("TradeStore -> queueEvent: No subscribers connected")
		return nil
	}

	if ts.buffer == nil {
		ts.buffer = make([]Trade, 0)
	}

	log.Debugf("TradeStore -> queueEvent: Adding trade to buffer: %+v", t)
	ts.buffer = append(ts.buffer, t)
	return nil
}

// GetByMarket retrieves all trades for a given market.
func (ts *memTradeStore) GetByMarket(market string, queryFilters *filters.TradeQueryFilters) ([]Trade, error) {
	if err := ts.marketExists(market); err != nil {
		return nil, err
	}
	if queryFilters == nil {
		queryFilters = &filters.TradeQueryFilters{}
	}
	return ts.filterResults(ts.store.markets[market].tradesByTimestamp, queryFilters)
}

// GetByMarketAndId retrieves a trade for a given id.
func (ts *memTradeStore) GetByMarketAndId(market string, id string) (Trade, error) {
	if err := ts.marketExists(market); err != nil {
		return Trade{}, err
	}
	v, ok := ts.store.markets[market].trades[id]
	if !ok {
		return Trade{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return v.trade, nil
}

// GetByPart retrieves all trades for a given party.
func (ts *memTradeStore) GetByParty(party string, queryFilters *filters.TradeQueryFilters) ([]Trade, error) {
	if err := ts.partyExists(party); err != nil {
		return nil, err
	}
	if queryFilters == nil {
		queryFilters = &filters.TradeQueryFilters{}
	}
	return ts.filterResults(ts.store.parties[party].tradesByTimestamp, queryFilters)
}

// GetByPartyAndId retrieves a trade for a given id.
func (ts *memTradeStore) GetByPartyAndId(party string, id string) (Trade, error) {
	if err := ts.partyExists(party); err != nil {
		return Trade{}, err
	}
	var at = -1
	for idx, trade := range ts.store.parties[party].tradesByTimestamp {
		if trade.trade.Id == id {
			at = idx
			break
		}
	}
	if at == -1 {
		return Trade{}, NotFoundError{fmt.Errorf("could not find id %s", id)}
	}
	return ts.store.parties[party].tradesByTimestamp[at].trade, nil
}


// Post creates a new trade in the memory store.
func (ts *memTradeStore) Post(trade Trade) error {
	if err := ts.validate(&trade); err != nil {
		return err
	}
	if _, exists := ts.store.markets[trade.Market].trades[trade.Id]; exists {
		return fmt.Errorf("trade exists in memstore: %s", trade.Id)
	}

	// check if passive and aggressive orders exist in the order ts

	// if passive exists
	aggressiveOrder, aggressiveExists := ts.store.markets[trade.Market].orders[trade.AggressiveOrderId]
	if !aggressiveExists {
		return fmt.Errorf("aggressive order for trade not found in memstore: %s", trade.AggressiveOrderId)
	}

	// if passive exists
	passiveOrder, passiveExists := ts.store.markets[trade.Market].orders[trade.PassiveOrderId]
	if !passiveExists {
		return fmt.Errorf("passive order for trade not found in memstore: %s", trade.PassiveOrderId)
	}

	newTrade := &memTrade{
		trade:      trade,
		aggressive: aggressiveOrder,
		passive:    passiveOrder,
	}
	
	// Add new trade to trades hashtable & queue in buffer to notify observers
	ts.store.markets[trade.Market].trades[trade.Id] = newTrade
	ts.queueEvent(trade)

	// append trade to aggressive and passive order
	aggressiveOrder.trades = append(aggressiveOrder.trades, newTrade)
	passiveOrder.trades = append(passiveOrder.trades, newTrade)

	// update tradesByTimestamp for MARKETS
	ts.store.markets[trade.Market].tradesByTimestamp = append(ts.store.markets[trade.Market].tradesByTimestamp, newTrade)

	// update party for both aggressive and passive parties
	ts.store.parties[newTrade.aggressive.order.Party].tradesByTimestamp = append(ts.store.parties[newTrade.aggressive.order.Party].tradesByTimestamp, newTrade)
	ts.store.parties[newTrade.passive.order.Party].tradesByTimestamp = append(ts.store.parties[newTrade.passive.order.Party].tradesByTimestamp, newTrade)
	return nil
}

// Put updates an existing trade in the store.
func (ts *memTradeStore) Put(trade Trade) error {
	if err := ts.validate(&trade); err != nil {
		return err
	}

	if err := ts.marketExists(trade.Market); err != nil {
		return err
	}

	if _, exists := ts.store.markets[trade.Market].trades[trade.Id]; !exists {
		return fmt.Errorf("trade not found in memstore: %s", trade.Id)
	}
	// Perform the update & queue in buffer to notify observers
	ts.store.markets[trade.Market].trades[trade.Id].trade = trade
	ts.queueEvent(trade)
	return nil
}

// Removes trade from the store.
func (ts *memTradeStore) Delete(trade Trade) error {
	if err := ts.validate(&trade); err != nil {
		return err
	}

	if err := ts.partyExists(trade.Seller); err != nil {
		return err
	}

	// Remove from tradesByTimestamp
	var pos uint64
	for idx, v := range ts.store.markets[trade.Market].tradesByTimestamp {
		if v.trade.Id == trade.Id {
			pos = uint64(idx)
			break
		}
	}
	ts.store.markets[trade.Market].tradesByTimestamp =
		append(ts.store.markets[trade.Market].tradesByTimestamp[:pos], ts.store.markets[trade.Market].tradesByTimestamp[pos+1:]...)

	// Remove from PARTIES tradesByTimestamp for BUYER
	pos = 0
	for idx, v := range ts.store.parties[trade.Buyer].tradesByTimestamp {
		if v.trade.Id == trade.Id {
			pos = uint64(idx)
			break
		}
	}
	ts.store.parties[trade.Buyer].tradesByTimestamp =
		append(ts.store.parties[trade.Buyer].tradesByTimestamp[:pos], ts.store.parties[trade.Buyer].tradesByTimestamp[pos+1:]...)

	// Remove from PARTIES tradesByTimestamp for SELLER
	pos = 0
	for idx, v := range ts.store.parties[trade.Seller].tradesByTimestamp {
		if v.trade.Id == trade.Id {
			pos = uint64(idx)
			break
		}
	}
	ts.store.parties[trade.Seller].tradesByTimestamp =
		append(ts.store.parties[trade.Seller].tradesByTimestamp[:pos], ts.store.parties[trade.Seller].tradesByTimestamp[pos+1:]...)

	delete(ts.store.markets[trade.Market].trades, trade.Id)
	return nil
}

// Checks to see if we have a market on the related memory store with given identifier.
// Returns an error if the market cannot be found and nil otherwise.
func (ts *memTradeStore) marketExists(market string) error {
	if !ts.store.marketExists(market) {
		return NotFoundError{fmt.Errorf("could not find market %s", market)}
	}
	return nil
}

func (ts *memTradeStore) partyExists(party string) error {
	if !ts.store.partyExists(party) {
		memParty := memParty{
			party:   party,
			ordersByTimestamp: []*memOrder{},
			tradesByTimestamp: []*memTrade{},
		}
		ts.store.parties[party] = &memParty
		return nil
	}
	return nil
}

func (ts *memTradeStore) validate(trade *Trade) error {
	if err := ts.marketExists(trade.Market); err != nil {
		return err
	}

	if err := ts.partyExists(trade.Buyer); err != nil {
		return err
	}

	if err := ts.partyExists(trade.Seller); err != nil {
		return err
	}

	return nil
}

func (ts *memTradeStore) GetMarkPrice(market string) (uint64, error) {
	last := uint64(1)
	filters := &filters.TradeQueryFilters{}
	filters.Last = &last
	recentTrade, err := ts.GetByMarket(market, filters)
	if err != nil {
		return 0, err
	}
	if len(recentTrade) == 0 {
		return 0, fmt.Errorf("NO TRADES")
	}
	return recentTrade[0].Price, nil
}


func (ts *memTradeStore) filterResults(input []*memTrade, queryFilters *filters.TradeQueryFilters) (output []Trade, error error) {
	var pos, skipped uint64

	// Last == descending by timestamp
	// First == ascending by timestamp
	// Skip == offset by value, then first/last depending on direction

	if queryFilters.First != nil && *queryFilters.First > 0 {
		// If first is set we iterate ascending
		for i := 0; i < len(input); i++ {
			if pos == *queryFilters.First {
				break
			}
			if applyTradeFilters(input[i].trade, queryFilters) {
				if queryFilters.Skip != nil && *queryFilters.Skip > 0 && skipped < *queryFilters.Skip {
					skipped++
					continue
				}
				output = append(output, input[i].trade)
				pos++
			}
		}
	} else {
		// default is descending 'last' n items
		for i := len(input) - 1; i >= 0; i-- {
			if queryFilters.Last != nil && *queryFilters.Last > 0 && pos == *queryFilters.Last {
				break
			}
			if applyTradeFilters(input[i].trade, queryFilters) {
				if queryFilters.Skip != nil && *queryFilters.Skip > 0 && skipped < *queryFilters.Skip {
					skipped++
					continue
				}
				output = append(output, input[i].trade)
				pos++
			}
		}
	}
	
	return output, nil
}
