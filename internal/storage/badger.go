package storage

import (
	"fmt"

	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"github.com/pkg/errors"
)

const badgerNamedLogger = "badger"

var (
	ErrTimeoutReached = errors.New("context cancelled due to timeout")
)

type badgerStore struct {
	db *badger.DB
}

// BadgerOptions are params for creating a DB object.
type BadgerOptions struct {
	// Dir                  string // not customisable by end user
	// ValueDir             string // not customisable by end user
	SyncWrites              bool
	TableLoadingMode        cfgencoding.FileLoadingMode
	ValueLogLoadingMode     cfgencoding.FileLoadingMode
	NumVersionsToKeep       int
	MaxTableSize            int64
	LevelSizeMultiplier     int
	MaxLevels               int
	ValueThreshold          int
	NumMemtables            int
	NumLevelZeroTables      int
	NumLevelZeroTablesStall int
	LevelOneSize            int64
	ValueLogFileSize        int64
	ValueLogMaxEntries      uint32
	NumCompactors           int
	CompactL0OnClose        bool
	ReadOnly                bool
	Truncate                bool
	// Logger               logging.Logger // not customisable by end user
}

func (bs *badgerStore) getIterator(txn *badger.Txn, descending bool) *badger.Iterator {
	if descending {
		return bs.descendingIterator(txn)
	}
	return bs.ascendingIterator(txn)
}

// DefaultBadgerOptions supplies default badger options to be used for all stores.
func DefaultBadgerOptions() BadgerOptions {
	opts := BadgerOptions{
		// Dir:                  TBD,
		// ValueDir:             TBD,
		SyncWrites:              true,
		TableLoadingMode:        cfgencoding.FileLoadingMode{FileLoadingMode: options.FileIO},
		ValueLogLoadingMode:     cfgencoding.FileLoadingMode{FileLoadingMode: options.FileIO},
		NumVersionsToKeep:       1,
		MaxTableSize:            64 << 20,
		LevelSizeMultiplier:     10,
		MaxLevels:               7,
		ValueThreshold:          32,
		NumMemtables:            1,
		NumLevelZeroTables:      1,
		NumLevelZeroTablesStall: 2,
		LevelOneSize:            256 << 20,
		ValueLogFileSize:        1<<30 - 1,
		ValueLogMaxEntries:      1000000,
		NumCompactors:           2,
		CompactL0OnClose:        true,
		ReadOnly:                false,
		Truncate:                false,
		// Logger:               TBD,
	}
	return opts
}

func badgerOptionsFromConfig(cfg BadgerOptions, dir string, log *logging.Logger) badger.Options {
	opts := badger.Options{
		Dir:                     dir,
		ValueDir:                dir,
		SyncWrites:              cfg.SyncWrites,
		TableLoadingMode:        cfg.TableLoadingMode.Get(),
		ValueLogLoadingMode:     cfg.ValueLogLoadingMode.Get(),
		NumVersionsToKeep:       cfg.NumVersionsToKeep,
		MaxTableSize:            cfg.MaxTableSize,
		LevelSizeMultiplier:     cfg.LevelSizeMultiplier,
		MaxLevels:               cfg.MaxLevels,
		ValueThreshold:          cfg.ValueThreshold,
		NumMemtables:            cfg.NumMemtables,
		NumLevelZeroTables:      cfg.NumLevelZeroTables,
		NumLevelZeroTablesStall: cfg.NumLevelZeroTablesStall,
		LevelOneSize:            cfg.LevelOneSize,
		ValueLogFileSize:        cfg.ValueLogFileSize,
		ValueLogMaxEntries:      cfg.ValueLogMaxEntries,
		NumCompactors:           cfg.NumCompactors,
		CompactL0OnClose:        cfg.CompactL0OnClose,
		ReadOnly:                cfg.ReadOnly,
		Truncate:                cfg.Truncate,
		Logger:                  log.Named(badgerNamedLogger),
	}
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

func (bs *badgerStore) marketKey(marketID string) []byte {
	return []byte(fmt.Sprintf("MID:%v", marketID))
}

func (bs *badgerStore) candleKey(market string, interval types.Interval, timestamp int64) []byte {
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
