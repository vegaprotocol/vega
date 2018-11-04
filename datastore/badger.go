package datastore

import (
	"fmt"
	"github.com/dgraph-io/badger"
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
