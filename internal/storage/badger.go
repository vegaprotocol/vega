package storage

import (
	"fmt"

	"code.vegaprotocol.io/vega/internal/logging"
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

const badgerNamedLogger = "badger"

var (
	ErrTimeoutReached = errors.New("context cancelled due to timeout")
)

type BadgerStore struct {
	DB *badger.DB
}

func (bs *BadgerStore) getIterator(txn *badger.Txn, descending bool) *badger.Iterator {
	if descending {
		return bs.descendingIterator(txn)
	}
	return bs.ascendingIterator(txn)
}

func badgerOptionsFromConfig(cfg storcfg.StorageConfig, log *logging.Logger) badger.Options {
	opts := badger.Options{
		Dir:                     cfg.Path,
		ValueDir:                cfg.Path,
		SyncWrites:              cfg.Badger.SyncWrites,
		TableLoadingMode:        cfg.Badger.TableLoadingMode.Get(),
		ValueLogLoadingMode:     cfg.Badger.ValueLogLoadingMode.Get(),
		NumVersionsToKeep:       cfg.Badger.NumVersionsToKeep,
		MaxTableSize:            cfg.Badger.MaxTableSize,
		LevelSizeMultiplier:     cfg.Badger.LevelSizeMultiplier,
		MaxLevels:               cfg.Badger.MaxLevels,
		ValueThreshold:          cfg.Badger.ValueThreshold,
		NumMemtables:            cfg.Badger.NumMemtables,
		NumLevelZeroTables:      cfg.Badger.NumLevelZeroTables,
		NumLevelZeroTablesStall: cfg.Badger.NumLevelZeroTablesStall,
		LevelOneSize:            cfg.Badger.LevelOneSize,
		ValueLogFileSize:        cfg.Badger.ValueLogFileSize,
		ValueLogMaxEntries:      cfg.Badger.ValueLogMaxEntries,
		NumCompactors:           cfg.Badger.NumCompactors,
		CompactL0OnClose:        cfg.Badger.CompactL0OnClose,
		ReadOnly:                cfg.Badger.ReadOnly,
		Truncate:                cfg.Badger.Truncate,
		Logger:                  log.Named(badgerNamedLogger),
	}
	return opts
}

func (bs *BadgerStore) descendingIterator(txn *badger.Txn) *badger.Iterator {
	opts := badger.DefaultIteratorOptions
	opts.Reverse = true
	return txn.NewIterator(opts)
}

func (bs *BadgerStore) ascendingIterator(txn *badger.Txn) *badger.Iterator {
	opts := badger.DefaultIteratorOptions
	return txn.NewIterator(opts)
}

func (bs *BadgerStore) partyPrefix(party string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	return bs.getPrefix("P", party, descending)
}

func (bs *BadgerStore) marketPrefix(market string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	return bs.getPrefix("M", market, descending)
}

func (bs *BadgerStore) orderPrefix(order string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	return bs.getPrefix("O", order, descending)
}

func (bs *BadgerStore) accountMarketPrefix(market string, descending bool) ([]byte, []byte) {
	return bs.getPrefix("AMR", market, descending)
}

func (bs *BadgerStore) accountOwnerPrefix(owner string, descending bool) ([]byte, []byte) {
	return bs.getPrefix("AR", owner, descending)
}

func (bs *BadgerStore) accountTypePrefix(owner string, accountType types.AccountType, descending bool) ([]byte, []byte) {
	return bs.getPrefix("ATR", fmt.Sprintf("%s:%s", owner, accountType.String()), descending)
}

func (bs *BadgerStore) accountReferencePrefix(owner, market string, descending bool) ([]byte, []byte) {
	return bs.getPrefix("AR", fmt.Sprintf("%s:%s", owner, market), descending)
}

func (bs *BadgerStore) accountAssetPrefix(owner, asset string, descending bool) ([]byte, []byte) {
	return bs.getPrefix("AA", fmt.Sprintf("%s:%s", owner, asset), descending)
}

func (bs *BadgerStore) accountKeyPrefix(owner, asset, market string, descending bool) ([]byte, []byte) {
	return bs.getPrefix("A", fmt.Sprintf("%s:%s:%s", owner, asset, market), descending)
}

func (bs *BadgerStore) getPrefix(modifier string, prefix string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	validForPrefix = []byte(fmt.Sprintf("%s:%s_", modifier, prefix))
	keyPrefix = validForPrefix
	if descending {
		keyPrefix = append(keyPrefix, 0xFF)
	}
	return keyPrefix, validForPrefix
}

func (bs *BadgerStore) candlePrefix(market string, interval types.Interval, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	validForPrefix = []byte(fmt.Sprintf("M:%s_I:%s_T:", market, interval))
	keyPrefix = validForPrefix
	if descending {
		keyPrefix = append(keyPrefix, 0xFF)
	}
	return keyPrefix, validForPrefix
}

func (bs *BadgerStore) readTransaction() *badger.Txn {
	return bs.DB.NewTransaction(false)
}

func (bs *BadgerStore) writeTransaction() *badger.Txn {
	return bs.DB.NewTransaction(true)
}

func (bs *BadgerStore) accountTypeReferenceKey(owner, market, asset string, accountType types.AccountType) []byte {
	return []byte(fmt.Sprintf("ATR:%s:%s:%s:%s", owner, accountType.String(), asset, market))
}

func (bs *BadgerStore) accountMarketReferenceKey(market, owner, asset string, accountType types.AccountType) []byte {
	return []byte(fmt.Sprintf("AMR:%s:%s:%s:%s", market, owner, asset, accountType.String()))
}

func (bs *BadgerStore) accountReferenceKey(owner, market, asset string, accountType types.AccountType) []byte {
	return []byte(fmt.Sprintf("AR:%s:%s:%s:%s", owner, market, asset, accountType.String()))
}

func (bs *BadgerStore) accountAssetReferenceKey(owner, asset, market string, accountType types.AccountType) []byte {
	return []byte(fmt.Sprintf("AA:%s:%s:%s:%s", owner, asset, market, accountType.String()))
}

func (bs *BadgerStore) accountKey(owner, asset, market string, accountType types.AccountType) []byte {
	return []byte(fmt.Sprintf("A:%s:%s:%s:%s", owner, asset, market, accountType.String()))
}

func (bs *BadgerStore) lastCandleKey(
	marketID string, interval types.Interval) []byte {
	return []byte(fmt.Sprintf("LCM:%s_I:%s", marketID, interval.String()))
}

func (bs *BadgerStore) marketKey(marketID string) []byte {
	return []byte(fmt.Sprintf("MID:%v", marketID))
}

func (bs *BadgerStore) candleKey(market string, interval types.Interval, timestamp int64) []byte {
	return []byte(fmt.Sprintf("M:%s_I:%s_T:%d", market, interval.String(), timestamp))
}

func (bs *BadgerStore) orderMarketKey(market string, Id string) []byte {
	return []byte(fmt.Sprintf("M:%s_ID:%s", market, Id))
}

func (bs *BadgerStore) orderReferenceKey(ref string) []byte {
	return []byte(fmt.Sprintf("R:%s", ref))
}

func (bs *BadgerStore) orderIdKey(Id string) []byte {
	return []byte(fmt.Sprintf("ID:%s", Id))
}

func (bs *BadgerStore) orderPartyKey(party string, Id string) []byte {
	return []byte(fmt.Sprintf("P:%s_ID:%s", party, Id))
}

func (bs *BadgerStore) tradeMarketKey(market string, Id string) []byte {
	return []byte(fmt.Sprintf("M:%s_ID:%s", market, Id))
}

func (bs *BadgerStore) tradeIdKey(Id string) []byte {
	return []byte(fmt.Sprintf("ID:%s", Id))
}

func (bs *BadgerStore) tradePartyKey(party, Id string) []byte {
	return []byte(fmt.Sprintf("P:%s_ID:%s", party, Id))
}

func (bs *BadgerStore) tradeOrderIdKey(orderId, Id string) []byte {
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
func (bs *BadgerStore) WriteBatch(kv map[string][]byte) (int, error) {
	// create transaction
	txn := bs.writeTransaction()
	defer txn.Discard()
	// add to transaction batch
	txns := []*badger.Txn{
		txn,
	}

	for k, v := range kv {
		// First attempt: put kv pair in current transaction
		if err := txn.Set([]byte(k), v); err != nil {
			if err != badger.ErrTxnTooBig {
				return 0, err
			}
			// Start a new transaction WITHOUT commiting any previous ones, in order
			// to maintain atomicity.
			txn = bs.writeTransaction()
			defer txn.Discard()
			txns = append(txns, txn)
			// Second attempt: put kv pair in new transaction
			if err := txn.Set([]byte(k), v); err != nil {
				return 0, err
			}
		}
	}

	// At this point, we have filled one or more transactions with the all the kv
	// pairs, and we have commited none of the transactions.
	for j, tx := range txns {
		if err := tx.Commit(); err != nil {
			// This is very bad. We committed some transactions, but have now failed
			// to commit a transaction.
			return j, err
		}
	}

	return len(txns) + 1, nil
}
