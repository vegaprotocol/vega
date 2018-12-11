package datastore

import (
	"fmt"
	"github.com/dgraph-io/badger"
	"vega/msg"
	"github.com/dgraph-io/badger/options"
)

type badgerStore struct {
	db *badger.DB
}

func (bs *badgerStore) getIterator(txn *badger.Txn, descending bool) *badger.Iterator {
	if descending {
		return bs.descendingIterator(txn)
	} else {
		return bs.ascendingIterator(txn)
	}
}

func customBadgerOptions(dir string) badger.Options {
	opts := badger.DefaultOptions
	opts.Dir, opts.ValueDir = dir, dir

	opts.MaxTableSize  = 32 << 20
	opts.NumMemtables  = 1
	opts.NumLevelZeroTables = 1
	opts.NumLevelZeroTablesStall = 2
	opts.NumCompactors = 2
	
	opts.TableLoadingMode, opts.ValueLogLoadingMode = options.FileIO, options.FileIO
	return opts
}

func (bs *badgerStore) descendingIterator(txn *badger.Txn) *badger.Iterator {
	opts := badger.DefaultIteratorOptions
	opts.Reverse = true
	return txn.NewIterator(opts)
}

func (bs *badgerStore) ascendingIterator(txn *badger.Txn) *badger.Iterator {
	opts := badger.DefaultIteratorOptions
	return txn.NewIterator(opts)
}

func (bs *badgerStore) partyPrefix(party string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	return bs.getPrefix("P", party, descending)
}

func (bs *badgerStore) marketPrefix(market string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	return bs.getPrefix("M", market, descending)
}

func (bs *badgerStore) getPrefix(modifier string, prefix string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	validForPrefix = []byte(fmt.Sprintf("%s:%s_", modifier, prefix))
	keyPrefix = validForPrefix
	if descending {
		keyPrefix = append(keyPrefix, 0xFF)
	}
	return keyPrefix, validForPrefix
}

func (bs *badgerStore) candlePrefix(market string, interval msg.Interval, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	validForPrefix = []byte(fmt.Sprintf("M:%s_I:%s_T:", market, interval))
	keyPrefix = validForPrefix
	if descending {
		keyPrefix = append(keyPrefix, 0xFF)
	}
	return keyPrefix, validForPrefix
}

func (bs *badgerStore) readTransaction() *badger.Txn {
	return bs.db.NewTransaction(false)
}

func (bs *badgerStore) writeTransaction() *badger.Txn {
	return bs.db.NewTransaction(true)
}

func (bs *badgerStore) candleKey(market string, interval msg.Interval, timestamp uint64) []byte {
	return []byte(fmt.Sprintf("M:%s_I:%s_T:%d", market, interval.String(), timestamp))
}

func (bs *badgerStore) orderMarketKey(market string, Id string) []byte {
	return []byte(fmt.Sprintf("M:%s_ID:%s", market, Id))
}

func (bs *badgerStore) orderIdKey(Id string) []byte {
	return []byte(fmt.Sprintf("ID:%s", Id))
}

func (bs *badgerStore) orderPartyKey(party string, Id string) []byte {
	return []byte(fmt.Sprintf("P:%s_ID:%s", party, Id))
}

func (bs *badgerStore) tradeMarketKey(market string, Id string) []byte {
	return []byte(fmt.Sprintf("M:%s_ID:%s", market, Id))
}

func (bs *badgerStore) tradeIdKey(Id string) []byte {
	return []byte(fmt.Sprintf("ID:%s", Id))
}

func (bs *badgerStore) tradePartyKey(party, Id string) []byte {
	return []byte(fmt.Sprintf("P:%s_ID:%s", party, Id))
}
