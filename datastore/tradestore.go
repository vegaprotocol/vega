package datastore

import (
	"errors"
	"fmt"
	"sync"
	"vega/log"
	"vega/filters"
	"vega/msg"
	"github.com/dgraph-io/badger"
)

// tradeStore should implement TradeStore interface.
type tradeStore struct {
	persistentStore *badger.DB

	subscribers map[uint64] chan<- []msg.Trade
	buffer []msg.Trade
	subscriberId uint64
	mu sync.Mutex
}

func NewTradeStore(dir string) TradeStore {
	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return &tradeStore{persistentStore: db}
}

func (ts *tradeStore) Close() {
	ts.persistentStore.Close()
}

func (ts *tradeStore) Subscribe(trades chan<- []msg.Trade) uint64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.subscribers == nil {
		log.Debugf("TradeStore -> Subscribe: Creating subscriber chan map")
		ts.subscribers = make(map[uint64] chan<- []msg.Trade)
	}

	ts.subscriberId = ts.subscriberId+1
	ts.subscribers[ts.subscriberId] = trades
	log.Debugf("TradeStore -> Subscribe: Trade subscriber added: %d", ts.subscriberId)
	return ts.subscriberId
}

func (ts *tradeStore) Unsubscribe(id uint64) error {
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

func (ts *tradeStore) Notify() error {

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
	var ok bool
	for id, sub := range ts.subscribers {
		select {
		case sub <- items:
			ok = true
			break
		default:
			ok = false
		}
		if ok{
			log.Debugf("Trades state updated")
		} else {
			log.Infof("Trades state could not been updated for subscriber %id", id)
		}
	}

	return nil
}

func (ts *tradeStore) queueEvent(t msg.Trade) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.subscribers == nil || len(ts.subscribers) == 0 {
		log.Debugf("TradeStore -> queueEvent: No subscribers connected")
		return nil
	}

	if ts.buffer == nil {
		ts.buffer = make([]msg.Trade, 0)
	}

	log.Debugf("TradeStore -> queueEvent: Adding trade to buffer: %+v", t)
	ts.buffer = append(ts.buffer, t)
	return nil
}

// GetByMarket retrieves all trades for a given market.
func (ts *tradeStore) GetByMarket(market string, queryFilters *filters.TradeQueryFilters) ([]*msg.Trade, error) {
	if queryFilters == nil {
		queryFilters = &filters.TradeQueryFilters{}
	}

	var (
		result []*msg.Trade
	)

	it := ts.persistentStore.NewTransaction(false).NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	marketPrefix := []byte(fmt.Sprintf("M:%s_", market))
	filter := TradeFilter{queryFilters, 0, 0}

	for it.Seek(marketPrefix); it.ValidForPrefix(marketPrefix); it.Next() {
		item := it.Item()
		tradeBuf, _ := item.ValueCopy(nil)

		var trade msg.Trade
		trade.XXX_Unmarshal(tradeBuf)
		if filter.apply(&trade) {
			result = append(result, &trade)
		}
	}

	fmt.Printf("trades fetched %d\n", len(result))
	return result, nil
}

// GetByMarketAndId retrieves a trade for a given id.
func (ts *tradeStore) GetByMarketAndId(market string, id string) (*msg.Trade, error) {
	var trade msg.Trade
	txn := ts.persistentStore.NewTransaction(false)
	marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", market, id))

	item, err := txn.Get(marketKey)
	if err != nil {
		return nil, err
	}

	tradeBuf, _ := item.ValueCopy(nil)
	if err := trade.XXX_Unmarshal(tradeBuf); err != nil {
		return nil, err
	}
	return &trade, err
}

// GetByPart retrieves all trades for a given party.
func (ts *tradeStore) GetByParty(party string, queryFilters *filters.TradeQueryFilters) ([]*msg.Trade, error) {

	if queryFilters == nil {
		queryFilters = &filters.TradeQueryFilters{}
	}

	var (
		result []*msg.Trade
	)

	txn := ts.persistentStore.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	defer it.Close()

	partyPrefix := []byte(fmt.Sprintf("P:%s_", party))
	filter := TradeFilter{queryFilters, 0, 0}

	for it.Seek(partyPrefix); it.ValidForPrefix(partyPrefix); it.Next() {
		marketKeyItem := it.Item()
		marketKey, _ := marketKeyItem.ValueCopy(nil)
		tradeItem, err := txn.Get(marketKey)
		if err != nil {
			fmt.Printf("TRADE %s DOES NOT EXIST", string(marketKey))
		}

		tradeBuf, _ := tradeItem.ValueCopy(nil)

		var trade msg.Trade
		trade.XXX_Unmarshal(tradeBuf)
		if filter.apply(&trade) {
			result = append(result, &trade)
		}
	}

	return result, nil
}

// GetByPartyAndId retrieves a trade for a given id.
func (ts *tradeStore) GetByPartyAndId(party string, id string) (*msg.Trade, error) {
	var trade msg.Trade
	err := ts.persistentStore.View(func(txn *badger.Txn) error {
		partyKey := []byte(fmt.Sprintf("P:%s_ID:%s", party, id))
		marketKeyItem, err := txn.Get(partyKey)
		if err != nil {
			return err
		}
		marketKey, err := marketKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		fmt.Printf("marketKey %s\n", string(marketKey))
		tradeItem, err := txn.Get(marketKey)
		if err != nil {
			return err
		}

		tradeBuf, err := tradeItem.ValueCopy(nil)
		if err != nil {
			fmt.Printf("TRADE %s DOES NOT EXIST\n", string(marketKey))
			return err
		}
		trade.XXX_Unmarshal(tradeBuf)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &trade, nil
}


// Post creates a new trade in the memory store.
func (ts *tradeStore) Post(trade *msg.Trade) error {

	txn := ts.persistentStore.NewTransaction(true)
	insertAtomically := func(txn *badger.Txn) error {
		orderBuf, _ := trade.XXX_Marshal(nil, true)
		marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", trade.Market, trade.Id))
		idKey := []byte(fmt.Sprintf("ID:%s", trade.Id))
		partyBuyerKey := []byte(fmt.Sprintf("P:%s_ID:%s", trade.Buyer, trade.Id))
		partySellerKey := []byte(fmt.Sprintf("P:%s_ID:%s", trade.Seller, trade.Id))
		if err := txn.Set(marketKey, orderBuf); err != nil {
			return err
		}
		if err := txn.Set(idKey, marketKey); err != nil {
			return err
		}
		if err := txn.Set(partyBuyerKey, marketKey); err != nil {
			return err
		}
		if err := txn.Set(partySellerKey, marketKey); err != nil {
			return err
		}
		return	nil
	}

	if err := insertAtomically(txn); err != nil {
		txn.Discard()
		return err
	}

	if err := txn.Commit(); err != nil {
		txn.Discard()
		return err
	}

	ts.queueEvent(*trade)
	return nil
}

// Removes trade from the store.
func (ts *tradeStore) Delete(trade *msg.Trade) error {

	txn := ts.persistentStore.NewTransaction(true)
	deleteAtomically := func() error {
		marketKey := []byte(fmt.Sprintf("M:%s_ID:%s", trade.Market, trade.Id))
		idKey := []byte(fmt.Sprintf("ID:%s", trade.Id))
		partyBuyerKey := []byte(fmt.Sprintf("P:%s_ID:%s", trade.Buyer, trade.Id))
		partySellerKey := []byte(fmt.Sprintf("P:%s_ID:%s", trade.Seller, trade.Id))
		if err := txn.Delete(marketKey); err != nil {
			return err
		}
		if err := txn.Delete(idKey); err != nil {
			return err
		}
		if err := txn.Delete(partyBuyerKey); err != nil {
			return err
		}
		if err := txn.Delete(partySellerKey); err != nil {
			return err
		}
		return nil
	}

	if err := deleteAtomically(); err != nil {
		txn.Discard()
		return err
	}

	if err := txn.Commit(); err != nil {
		txn.Discard()
		return err
	}
	return nil
}

func (ts *tradeStore) GetMarkPrice(market string) (uint64, error) {
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


type TradeFilter struct {
	queryFilter *filters.TradeQueryFilters
	skipped uint64
	Q uint64
}

func (f *TradeFilter) apply(trade *msg.Trade) (include bool) {
	if f.queryFilter.First == nil && f.queryFilter.Skip == nil {
		include = true
	} else {
		if f.queryFilter.First != nil && *f.queryFilter.First > 0 && f.Q < *f.queryFilter.First {
			include = true
		}

		if f.queryFilter.Skip != nil && *f.queryFilter.Skip > 0 && f.skipped < *f.queryFilter.Skip {
			f.skipped++
			return false
		}
	}

	if !applyTradeFilters(trade, f.queryFilter) {
		return false
	}

	// if order passes the filter, increment the Q
	if include {
		f.Q++
	}
	return include
}

// filter results and paginate based on query filters
//func (ts *tradeStore) filterResults(input []*msg.Trade, queryFilters *filters.TradeQueryFilters) (output []*msg.Trade, error error) {
//	var pos, skipped uint64
//
//	// Last == descending by timestamp
//	// First == ascending by timestamp
//	// Skip == offset by value, then first/last depending on direction
//
//	if queryFilters.First != nil && *queryFilters.First > 0 {
//		// If first is set we iterate ascending
//		for i := 0; i < len(input); i++ {
//			if pos == *queryFilters.First {
//				break
//			}
//			if applyTradeFilters(input[i], queryFilters) {
//				if queryFilters.Skip != nil && *queryFilters.Skip > 0 && skipped < *queryFilters.Skip {
//					skipped++
//					continue
//				}
//				output = append(output, input[i])
//				pos++
//			}
//		}
//	} else {
//		// default is descending 'last' n items
//		for i := len(input) - 1; i >= 0; i-- {
//			if queryFilters.Last != nil && *queryFilters.Last > 0 && pos == *queryFilters.Last {
//				break
//			}
//			if applyTradeFilters(input[i], queryFilters) {
//				if queryFilters.Skip != nil && *queryFilters.Skip > 0 && skipped < *queryFilters.Skip {
//					skipped++
//					continue
//				}
//				output = append(output, input[i])
//				pos++
//			}
//		}
//	}
//
//	return output, nil
//}
