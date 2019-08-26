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

// ConfigOptions are params for creating a DB object.
// todo: Make this struct generic/base options for stores
type ConfigOptions struct {
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

// DefaultStoreOptions supplies default options to be used for all stores.
func DefaultStoreOptions() ConfigOptions {
	/*
		Notes:
		* MaxTableSize: set low to avoid badger grabbing-then-releasing gigs of memory (#147)
		* ValueThreshold: set low to move most data out of the LSM tree (#147)
	*/
	fileio := cfgencoding.FileLoadingMode{FileLoadingMode: options.FileIO}
	opts := ConfigOptions{
		// Dir:                  TBD,       // string
		// ValueDir:             TBD,       // string
		SyncWrites:              true,      // bool
		TableLoadingMode:        fileio,    // options.FileLoadingMode, default options.MemoryMap
		ValueLogLoadingMode:     fileio,    // options.FileLoadingMode, default options.MemoryMap
		NumVersionsToKeep:       1,         // int
		MaxTableSize:            64 << 20,  // int64, default 64<<20 (64MB)
		LevelSizeMultiplier:     10,        // int
		MaxLevels:               7,         // int
		ValueThreshold:          32,        // int, default 32
		NumMemtables:            5,         // int, default 5
		NumLevelZeroTables:      5,         // int, default 5
		NumLevelZeroTablesStall: 10,        // int, default 10
		LevelOneSize:            256 << 20, // int64, default 256<<20
		ValueLogFileSize:        1<<30 - 1, // int64, default 1<<30-1 (almost 1GB)
		ValueLogMaxEntries:      1000000,   // uint32
		NumCompactors:           2,         // int, default 2
		CompactL0OnClose:        true,      // bool
		ReadOnly:                false,     // bool
		Truncate:                false,     // bool
		// Logger:               TBD,       // Logger, default defaultLogger
	}
	return opts
}

func getOptionsFromConfig(cfg ConfigOptions, dir string, log *logging.Logger) badger.Options {
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

func (bs *badgerStore) assetPrefix(asset string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	return bs.getPrefix("A", asset, descending)
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

func (bs *badgerStore) accountPartyPrefix(accType types.AccountType, party string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	return bs.getPrefix(bs.getAccountTypePrefix(accType), party, descending)
}

func (bs *badgerStore) accountPartyMarketPrefix(accType types.AccountType, partyID string, marketID string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	validForPrefix = []byte(fmt.Sprintf("%s:%s_M:%s_", bs.getAccountTypePrefix(accType), partyID, marketID))
	keyPrefix = validForPrefix
	if descending {
		keyPrefix = append(keyPrefix, 0xFF)
	}
	return keyPrefix, validForPrefix
}

func (bs *badgerStore) accountPartyAssetPrefix(partyID string, asset string, descending bool) (keyPrefix []byte, validForPrefix []byte) {
	validForPrefix = []byte(fmt.Sprintf("A:%s_%s_ID:", asset, partyID))
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

// Market store keys

func (bs *badgerStore) marketKey(marketID string) []byte {
	return []byte(fmt.Sprintf("ID:%v", marketID))
}

// Candle store keys

func (bs *badgerStore) lastCandleKey(
	marketID string, interval types.Interval) []byte {
	return []byte(fmt.Sprintf("LCM:%s_I:%s", marketID, interval.String()))
}

func (bs *badgerStore) candleKey(market string, interval types.Interval, timestamp int64) []byte {
	return []byte(fmt.Sprintf("M:%s_I:%s_T:%d", market, interval.String(), timestamp))
}

// Order store keys

func (bs *badgerStore) orderMarketKey(market string, Id string) []byte {
	return []byte(fmt.Sprintf("M:%s_ID:%s", market, Id))
}

func (bs *badgerStore) orderReferenceKey(ref string) []byte {
	return []byte(fmt.Sprintf("R:%s", ref))
}

func (bs *badgerStore) orderIdKey(Id string) []byte {
	return []byte(fmt.Sprintf("ID:%s", Id))
}

func (bs *badgerStore) orderPartyKey(party string, Id string) []byte {
	return []byte(fmt.Sprintf("P:%s_ID:%s", party, Id))
}

// Trade store keys

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

// Account store keys

// accountGeneralKey relates only to a party and asset, no market index/references
func (bs *badgerStore) accountGeneralIdKey(partyID string, assetID string) []byte {
	return []byte(fmt.Sprintf("%s:%s_A:%s",
		bs.getAccountTypePrefix(types.AccountType_GENERAL), partyID, assetID))
}
// accountMarginKey is composed from a party market and asset, has a market index (future work could add an asset index)
func (bs *badgerStore) accountMarginIdKey(partyID string, marketID string, assetID string) []byte {
	return []byte(fmt.Sprintf("%s:%s_M:%s_A:%s",
		bs.getAccountTypePrefix(types.AccountType_MARGIN), partyID, marketID, assetID))
}
// accountMarketKey is used to provide an index of all accounts for a particular market (no party scope).
// Id should be a reference to the accountMarginIdKey generated above, general accounts span
// all of VEGA without having market scope. Currently used for MARGIN type only.
func (bs *badgerStore) accountMarketKey(market string, accountID string) []byte {
	return []byte(fmt.Sprintf("M:%s_ID:%s", market, accountID))
}
// accountAssetKey is used to provide an index of accounts for a particular asset.
// Used by both general and margin accounts.
func (bs *badgerStore) accountAssetKey(assetID string, partyID string, accountID string) []byte {
	return []byte(fmt.Sprintf("A:%s_%s_ID:%s", assetID, partyID, accountID))
}

// getAccountTypePrefix returns the correct code for a particular account type.
// Currently we only write GENERAL and MARGIN type records to store.
func (bs *badgerStore) getAccountTypePrefix(accType types.AccountType) string {
	switch accType {
	case types.AccountType_MARGIN:
		return "MP"
	case types.AccountType_SETTLEMENT:
		return "SP"
	case types.AccountType_INSURANCE:
		return "IP"
	case types.AccountType_GENERAL:
		return "GP"
	default:
		return "ERR"
	}
}

// writeBatch writes an arbitrarily large map to a Badger store, using as many
// transactions as necessary.
//
// Return values:
// N, nil: The map was successfully committed, in N transactions.
// 0, err: None of the map was committed.
// N, err: The map was partially committed. The first N transactions were
//         committed successfully, but an error was returned on the transaction
//         number N+1.
func (bs *badgerStore) writeBatch(kv map[string][]byte) (int, error) {
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
