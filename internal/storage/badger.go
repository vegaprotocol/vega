package storage

import (
	"fmt"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
)

const badgerNamedLogger = "badger"

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

func customBadgerOptions(dir string, log *logging.Logger) badger.Options {
	opts := badger.DefaultOptions
	opts.Dir, opts.ValueDir = dir, dir

	opts.MaxTableSize = 64 << 20
	opts.NumMemtables = 1
	opts.NumLevelZeroTables = 1
	opts.NumLevelZeroTablesStall = 2
	opts.NumCompactors = 2

	opts.TableLoadingMode, opts.ValueLogLoadingMode = options.FileIO, options.FileIO
	opts.Logger = log.Named(badgerNamedLogger)

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

func (bs *badgerStore) orderPrefix(order string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	return bs.getPrefix("O", order, descending)
}

func (bs *badgerStore) getPrefix(modifier string, prefix string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	validForPrefix = []byte(fmt.Sprintf("%s:%s_", modifier, prefix))
	keyPrefix = validForPrefix
	if descending {
		keyPrefix = append(keyPrefix, 0xFF)
	}
	return keyPrefix, validForPrefix
}

func (bs *badgerStore) candlePrefix(market string, interval types.Interval, descending bool) (keyPrefix []byte, validForPrefix []byte) {
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

func (bs *badgerStore) candleKey(market string, interval types.Interval, timestamp uint64) []byte {
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

func (bs *badgerStore) tradeOrderIdKey(orderId, Id string) []byte {
	return []byte(fmt.Sprintf("O:%s_ID:%s", orderId, Id))
}

// writeBatch writes an arbitrarily large map to a Barger store, using as many
// transactions as necessary.
//
// Return values:
// N, nil: The map was successfully committed, in N transactions.
// 0, err: None of the map was committed.
// N, err: The map was partially committed. The first N transactions were
//         committed successfully, but an error was returned on the transaction
//         number N+1.
func (bs *badgerStore) writeBatch(kv map[string][]byte) (int, error) {
	txns := make([]*badger.Txn, 0)
	lastTxnIdx := 0

	txns = append(txns, bs.writeTransaction())
	defer txns[lastTxnIdx].Discard()

	i := 0
	for k, v := range kv {
		// First attempt: put kv pair in current transaction
		err := txns[lastTxnIdx].Set([]byte(k), v)
		switch err {
		case nil: // all is well
		case badger.ErrTxnTooBig:
			// Start a new transaction WITHOUT commiting any previous ones, in order
			// to maintain atomicity.
			txns = append(txns, bs.writeTransaction())
			lastTxnIdx++
			defer txns[lastTxnIdx].Discard()

			// Second attempt: put kv pair in new transaction
			err = txns[lastTxnIdx].Set([]byte(k), v)
			if err != nil {
				return 0, err
				// All transactions will be discarded
			}
			i = 0
		default:
			return 0, err
			// All transactions will be discarded
		}
		i++
	}

	// At this point, we have filled one or more transactions with the all the kv
	// pairs, and we have commited none of the transactions.

	for j, txn := range txns {
		err := txn.Commit()
		if err != nil {
			// This is very bad. We committed some transactions, but have now failed
			// to commit a transaction.
			return j, err
		}
	}

	return lastTxnIdx + 1, nil
}
