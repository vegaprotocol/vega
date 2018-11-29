package datastore

import (
	"errors"
	"fmt"
	"sync"
	"vega/log"
	"vega/filters"
	"vega/msg"
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
)

// tradeStore should implement TradeStore interface.
type tradeStore struct {
	badger *badgerStore
	buffer []msg.Trade

	subscribers map[uint64] chan<- []msg.Trade
	subscriberId uint64
	mu sync.Mutex
}

func NewTradeStore(dir string) TradeStore {
	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	opts.SyncWrites = true
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalf(err.Error())
	}
	bs := badgerStore{db: db}
	return &tradeStore{badger: &bs,  buffer: make([]msg.Trade, 0), subscribers: make(map[uint64] chan<- []msg.Trade)}
}

func (ts *tradeStore) Close() {
	ts.badger.db.Close()
}

func (ts *tradeStore) Subscribe(trades chan<- []msg.Trade) uint64 {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.subscriberId = ts.subscriberId+1
	ts.subscribers[ts.subscriberId] = trades

	log.Debugf("TradeStore -> Subscribe: Trade subscriber added: %d", ts.subscriberId)
	return ts.subscriberId
}

func (ts *tradeStore) Unsubscribe(id uint64) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if len(ts.subscribers) == 0 {
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

func (ts *tradeStore) Commit() error {
	ts.mu.Lock()
	items := ts.buffer
	ts.buffer = make([]msg.Trade, 0)
	ts.mu.Unlock()

	ts.PostBatch(items)
	ts.Notify(items)
	return nil
}

func (ts *tradeStore) Notify(items []msg.Trade) error {

	if len(ts.subscribers) == 0 {
		log.Debugf("TradeStore -> Notify: No subscribers connected")
		return nil
	}

	if len(ts.buffer) == 0 {
		// Only publish when we have items
		log.Debugf("TradeStore -> Notify: No trades in buffer")
		return nil
	}

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

func (ts *tradeStore) addToBuffer(t msg.Trade) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.buffer = append(ts.buffer, t)

	log.Debugf("TradeStore -> addToBuffer: Adding trade to buffer: %+v", t)
	return nil
}

// GetByMarket retrieves all trades for a given market.
func (ts *tradeStore) GetByMarket(market string, queryFilters *filters.TradeQueryFilters) ([]*msg.Trade, error) {
	var result []*msg.Trade
	if queryFilters == nil {
		queryFilters = &filters.TradeQueryFilters{}
	}

	txn := ts.badger.db.NewTransaction(false)
	filter := TradeFilter{queryFilter: queryFilters}
	descending := filter.queryFilter.HasLast()
	it := ts.badger.getIterator(txn, descending)

	defer it.Close()
	marketPrefix, validForPrefix := ts.badger.marketPrefix(market, descending)
	for it.Seek(marketPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		item := it.Item()
		tradeBuf, _ := item.ValueCopy(nil)
		var trade msg.Trade
		if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
			log.Errorf("Failed to unmarshal %s", err.Error())
		}
		if filter.apply(&trade) {
			result = append(result, &trade)
		}
		if filter.isFull() {
			break
		}
	}
	return result, nil
}

// GetByMarketAndId retrieves a trade for a given id.
func (ts *tradeStore) GetByMarketAndId(market string, Id string) (*msg.Trade, error) {
	txn := ts.badger.db.NewTransaction(false)
	marketKey := ts.badger.tradeMarketKey(market, Id)
	item, err := txn.Get(marketKey)
	if err != nil {
		return nil, err
	}
	tradeBuf, _ := item.ValueCopy(nil)
	var trade msg.Trade
	if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
		log.Errorf("Failed to unmarshal %s", err.Error())
		return nil, err
	}
	return &trade, err
}

// GetByPart retrieves all trades for a given party.
func (ts *tradeStore) GetByParty(party string, queryFilters *filters.TradeQueryFilters) ([]*msg.Trade, error) {
	var result []*msg.Trade
	if queryFilters == nil {
		queryFilters = &filters.TradeQueryFilters{}
	}
	
	txn := ts.badger.db.NewTransaction(false)
	filter := TradeFilter{queryFilter: queryFilters}
	descending := filter.queryFilter.HasLast()
	it := ts.badger.getIterator(txn, descending)
	defer it.Close()
	partyPrefix, validForPrefix := ts.badger.partyPrefix(party, descending)
	for it.Seek(partyPrefix); it.ValidForPrefix(validForPrefix); it.Next() {
		marketKeyItem := it.Item()
		marketKey, _ := marketKeyItem.ValueCopy(nil)
		tradeItem, err := txn.Get(marketKey)
		if err != nil {
			log.Infof("TRADE %s DOES NOT EXIST", string(marketKey))
		}
		tradeBuf, _ := tradeItem.ValueCopy(nil)
		var trade msg.Trade
		if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
			log.Errorf("Failed to unmarshal %s", err.Error())
		}
		if filter.apply(&trade) {
			result = append(result, &trade)
		}
		if filter.isFull() {
			break
		}
	}
	
	return result, nil
}

// GetByPartyAndId retrieves a trade for a given id.
func (ts *tradeStore) GetByPartyAndId(party string, Id string) (*msg.Trade, error) {
	var trade msg.Trade
	err := ts.badger.db.View(func(txn *badger.Txn) error {
		partyKey := ts.badger.tradePartyKey(party, Id)
		marketKeyItem, err := txn.Get(partyKey)
		if err != nil {
			return err
		}
		marketKey, err := marketKeyItem.ValueCopy(nil)
		if err != nil {
			return err
		}
		tradeItem, err := txn.Get(marketKey)
		if err != nil {
			return err
		}

		tradeBuf, err := tradeItem.ValueCopy(nil)
		if err != nil {
			log.Errorf("TRADE %s DOES NOT EXIST\n", string(marketKey))
			return err
		}
		if err := proto.Unmarshal(tradeBuf, &trade); err != nil {
			log.Errorf("Failed to unmarshal %s", err.Error())
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &trade, nil
}


// Post creates a new trade in the memory store.
func (ts *tradeStore) Post(trade *msg.Trade) error {
	ts.addToBuffer(*trade)
	return nil
}

func (ts *tradeStore) PostBatch(batch []msg.Trade) error {
	wb := ts.badger.db.NewWriteBatch()
	defer wb.Cancel()

	insertBatchAtomically := func() error {
		for idx := range batch {
			tradeBuf, err := proto.Marshal(&batch[idx])
			if err != nil {
				log.Errorf("Marshal failed %s", err.Error())
			}
			marketKey := ts.badger.tradeMarketKey(batch[idx].Market, batch[idx].Id)
			idKey := ts.badger.tradeIdKey(batch[idx].Id)
			buyerPartyKey := ts.badger.tradePartyKey(batch[idx].Buyer, batch[idx].Id)
			sellerPartyKey := ts.badger.tradePartyKey(batch[idx].Seller, batch[idx].Id)
			if err := wb.Set(marketKey, tradeBuf, 0); err != nil {
				return err
			}
			if err := wb.Set(idKey, marketKey, 0); err != nil {
				return err
			}
			if err := wb.Set(buyerPartyKey, marketKey, 0); err != nil {
				return err
			}
			if err := wb.Set(sellerPartyKey, marketKey, 0); err != nil {
				return err
			}
		}
		return nil
	}

	if err := insertBatchAtomically(); err == nil {
		if err := wb.Flush(); err != nil {
			log.Errorf("failed to flush batch %+v \n", err)
		}
		// implement retry mechanism
	} else {
		wb.Cancel()
		// implement retry mechanism
	}

	return nil
}

// Removes trade from the store.
func (ts *tradeStore) Delete(trade *msg.Trade) error {

	txn := ts.badger.db.NewTransaction(true)
	deleteAtomically := func() error {
		marketKey := ts.badger.tradeMarketKey(trade.Market, trade.Id)
		idKey := ts.badger.tradeIdKey(trade.Id)
		partyBuyerKey := ts.badger.tradePartyKey(trade.Buyer, trade.Id)
		partySellerKey := ts.badger.tradePartyKey(trade.Seller, trade.Id)
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
	//last := uint64(1)
	f := &filters.TradeQueryFilters{}
	//f.Last = &last
	recentTrade, err := ts.GetByMarket(market, f)
	if err != nil {
		return 0, err
	}
	log.Debugf("recentTrade: %+v\n", recentTrade)
	if len(recentTrade) == 0 {
		return 0, fmt.Errorf("NO TRADES")
	}
	return recentTrade[0].Price, nil
}


type TradeFilter struct {
	queryFilter *filters.TradeQueryFilters
	skipped uint64
	found uint64
}

func (f *TradeFilter) apply(trade *msg.Trade) (include bool) {
	if f.queryFilter.First == nil && f.queryFilter.Last == nil && f.queryFilter.Skip == nil {
		include = true
	} else {

		if f.queryFilter.HasFirst() && f.found < *f.queryFilter.First {
			include = true
		}
		if f.queryFilter.HasLast() && f.found < *f.queryFilter.Last {
			include = true
		}
		if f.queryFilter.HasSkip() && f.skipped < *f.queryFilter.Skip {
			f.skipped++
			return false
		}
	}

	if !applyTradeFilters(trade, f.queryFilter) {
		return false
	}

	// if item passes the filter, increment the found counter
	if include {
		f.found++
	}
	return include
}

func (f *TradeFilter) isFull() bool {
	if f.queryFilter.HasLast() && f.found == *f.queryFilter.Last {
		return true
	}
	return false
}