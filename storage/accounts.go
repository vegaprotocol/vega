package storage

import (
	"fmt"
	"sync"
	"sync/atomic"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	// ErrMarketNotFound signals that the market related to the
	// the account we were looking for does not exist
	ErrMarketNotFound = errors.New("no accounts found for market")
	// ErrOwnerNotFound signals that the owner related to the
	// account we were looking for does not exists
	ErrOwnerNotFound = errors.New("no accounts found for party")
)

// Account represents a collateral account store
type Account struct {
	Config

	log             *logging.Logger
	badger          *badgerStore
	batchCountForGC int32
	subscribers     map[uint64]chan []*types.Account
	subscriberID    uint64
	mu              sync.Mutex
}

// NewAccounts creates a new account store with the logger and configuration specified.
func NewAccounts(log *logging.Logger, c Config) (*Account, error) {
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	if err := InitStoreDirectory(c.AccountsDirPath); err != nil {
		return nil, errors.Wrap(err, "error on init badger database for account storage")
	}
	db, err := badger.Open(getOptionsFromConfig(c.Accounts, c.AccountsDirPath, log))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for account storage")
	}

	return &Account{
		log:         log,
		Config:      c,
		badger:      &badgerStore{db: db},
		subscribers: map[uint64]chan []*types.Account{},
	}, nil
}

// ReloadConf will trigger a reload of all the config settings in the account store.
// Required when hot-reloading any config changes, eg. logger level.
func (a *Account) ReloadConf(cfg Config) {
	a.log.Info("reloading configuration")
	if a.log.GetLevel() != cfg.Level.Get() {
		a.log.Info("updating log level",
			logging.String("old", a.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		a.log.SetLevel(cfg.Level.Get())
	}

	a.Config = cfg
}

// Close our connection to the badger database
// ensuring errors will be returned up the stack.
func (a *Account) Close() error {
	return a.badger.db.Close()
}

// GetByParty returns all accounts for a given party, including MARGIN and GENERAL accounts
func (a *Account) GetByParty(partyID string) ([]*types.Account, error) {
	// Read all GENERAL accounts for party
	keyPrefix, validFor := a.badger.accountPartyPrefix(types.AccountType_GENERAL, partyID, false)
	generalAccounts, err := a.getAccountsForPrefix(keyPrefix, validFor, false)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error loading general accounts for party: %s", partyID))
	}
	// Read all MARGIN accounts for party
	keyPrefix, validFor = a.badger.accountPartyPrefix(types.AccountType_MARGIN, partyID, false)
	marginAccounts, err := a.getAccountsForPrefix(keyPrefix, validFor, false)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error loading margin accounts for party: %s", partyID))
	}
	return append(generalAccounts, marginAccounts...), nil
}

// GetByPartyAndMarket will return all accounts (if available) relating to the provided party and market.
//  - Only MARGIN accounts are returned by this call, as they have market scope.
func (a *Account) GetByPartyAndMarket(partyID string, marketID string) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountPartyMarketPrefix(types.AccountType_MARGIN, partyID, marketID, false)
	return a.getAccountsForPrefix(keyPrefix, validFor, false)
}

// GetByPartyAndType will return all accounts (if available) relating to the provided party and account type.
//  - GENERAL and MARGIN accounts are supported by this call, will return all MARGIN accounts for all markets.
func (a *Account) GetByPartyAndType(partyID string, accType types.AccountType) ([]*types.Account, error) {
	if accType != types.AccountType_GENERAL && accType != types.AccountType_MARGIN {
		return nil, errors.New("invalid type for query, only GENERAL and MARGIN accounts for a party supported")
	}
	keyPrefix, validFor := a.badger.accountPartyPrefix(accType, partyID, false)
	return a.getAccountsForPrefix(keyPrefix, validFor, false)
}

// GetByPartyAndAsset will return all accounts (if available) relating to the provided party and asset.
func (a *Account) GetByPartyAndAsset(partyID string, asset string) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountPartyAssetPrefix(partyID, asset, false)
	return a.getAccountsForPrefix(keyPrefix, validFor, true)
}

// getAccountsForPartyPrefix does the work of querying the badger store for key prefixes
// and loading direct values from the underlying based collateral account store.
func (a *Account) getAccountsForPrefix(prefix, validFor []byte, byReference bool) ([]*types.Account, error) {
	var err error
	ret := make([]*types.Account, 0)

	txn := a.badger.readTransaction()
	defer txn.Discard()

	it := a.badger.getIterator(txn, false)
	defer it.Close()

	var accountBuf []byte
	for it.Seek(prefix); it.ValidForPrefix(validFor); it.Next() {
		// If loading the data indirectly via a secondary index reference
		// then the caller must set `byReference` to true
		if byReference {
			var keyBuf []byte
			if keyBuf, err = it.Item().ValueCopy(keyBuf); err != nil {
				return nil, err
			}
			var item *badger.Item
			item, err = txn.Get(keyBuf)
			if err != nil {
				return nil, err
			}
			if accountBuf, err = item.ValueCopy(accountBuf); err != nil {
				return nil, err
			}
		} else {
			if accountBuf, err = it.Item().ValueCopy(accountBuf); err != nil {
				return nil, err
			}
		}
		var acc types.Account
		if err = proto.Unmarshal(accountBuf, &acc); err != nil {
			a.log.Error("Failed to unmarshal account value from badger in account store",
				logging.Error(err),
				logging.String("badger-key", string(it.Item().Key())),
				logging.String("raw-bytes", string(accountBuf)))
			return nil, err
		}

		ret = append(ret, &acc)
	}

	return ret, nil
}

// SaveBatch writes a slice of account changes to the underlying store.
func (a *Account) SaveBatch(accs []*types.Account) error {
	if len(accs) == 0 {
		// Sanity check, no need to do any processing on an empty batch.
		return nil
	}

	batch, err := a.parseBatch(accs...)
	if err != nil {
		a.log.Error(
			"Unable to parse accounts batch",
			logging.Error(err),
			logging.Int("batch-size", len(accs)))
		return err
	}

	if logging.DebugLevel == a.log.GetLevel() {
		for key, data := range batch {
			a.log.Debug("", logging.String("key", key), logging.String("data", string(data)))
		}
	}

	_, err = a.badger.writeBatch(batch)
	if err != nil {
		a.log.Error(
			"Unable to write accounts batch",
			logging.Error(err),
			logging.Int("batch-size", len(accs)))

		return err
	}
	a.notify(accs)

	if logging.DebugLevel == a.log.GetLevel() {
		a.log.Debug("Accounts store updated", logging.Int("batch-size", len(accs)))
	}

	// Using a batch counter ties the clean up to the average
	// expected size of a batch of account updates, not just time.
	atomic.AddInt32(&a.batchCountForGC, 1)
	if atomic.LoadInt32(&a.batchCountForGC) >= maxBatchesUntilValueLogGC {
		go func() {
			a.log.Info("Account store value log garbage collection",
				logging.Int32("attempt", atomic.LoadInt32(&a.batchCountForGC)-maxBatchesUntilValueLogGC))

			err := a.badger.GarbageCollectValueLog()
			if err != nil {
				a.log.Error("Unexpected problem running valueLogGC on accounts store",
					logging.Error(err))
			} else {
				atomic.StoreInt32(&a.batchCountForGC, 0)
			}
		}()
	}

	return nil
}

// notify is a helper func used to send any updates to any subscribers for mutations of the
// account store.
func (a *Account) notify(accs []*types.Account) {
	if len(accs) == 0 {
		return
	}

	a.mu.Lock()
	if len(a.subscribers) == 0 {
		a.log.Debug("No subscribers connected in accounts store")
		a.mu.Unlock()
		return
	}

	var ok bool
	for id, sub := range a.subscribers {
		select {
		case sub <- accs:
			ok = true
			break
		default:
			ok = false
		}
		if ok {
			a.log.Debug("Accounts channel updated for subscriber successfully",
				logging.Uint64("id", id))
		} else {
			a.log.Debug("Accounts channel could not be updated for subscriber",
				logging.Uint64("id", id))
		}
	}
	a.mu.Unlock()
	return
}

// parseBatch takes a list of accounts and outputs the necessary badger keys and values
// in a slice ready to write down to disk using the generic writeBatch function.
func (a *Account) parseBatch(accounts ...*types.Account) (map[string][]byte, error) {
	batch := make(map[string][]byte)
	for _, acc := range accounts {

		// Validate marketID as only MARGIN accounts should have a marketID specified
		if acc.MarketID == "" && acc.Type != types.AccountType_GENERAL {
			err := errors.New(fmt.Sprintf("general account should not have a market"))
			a.log.Error(err.Error(), logging.Account(*acc))
			return nil, err
		}

		// Marshall proto struct to byte buffer for storage
		buf, err := proto.Marshal(acc)
		if err != nil {
			a.log.Error("unable to marshal account",
				logging.String("account-id", acc.Id),
				logging.Error(err),
			)
			return nil, err
		}

		// Check the type of account and write only the data required for GENERAL accounts.
		if acc.Type == types.AccountType_GENERAL {
			// General accounts have no scope of an individual market, they span all markets.
			generalIDKey := a.badger.accountGeneralIDKey(acc.Owner, acc.Asset)
			generalAssetKey := a.badger.accountAssetKey(acc.Asset, acc.Owner, string(generalIDKey))
			batch[string(generalIDKey)] = buf
			batch[string(generalAssetKey)] = generalIDKey
		}
		// Check the type of account and write only the data/keys required for MARGIN accounts.
		if acc.Type == types.AccountType_MARGIN {
			marginIDKey := a.badger.accountMarginIDKey(acc.Owner, acc.MarketID, acc.Asset)
			marginMarketKey := a.badger.accountMarketKey(acc.MarketID, string(marginIDKey))
			marginAssetKey := a.badger.accountAssetKey(acc.Asset, acc.Owner, string(marginIDKey))
			batch[string(marginIDKey)] = buf
			batch[string(marginMarketKey)] = marginIDKey
			batch[string(marginAssetKey)] = marginIDKey
		}
	}
	return batch, nil
}

// Subscribe to account store updates, any changes will be pushed out on this channel.
func (a *Account) Subscribe(c chan []*types.Account) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.subscriberID++
	a.subscribers[a.subscriberID] = c

	a.log.Debug("Account subscriber added in account store",
		logging.Uint64("subscriber-id", a.subscriberID))

	return a.subscriberID
}

// Unsubscribe from account store updates.
func (a *Account) Unsubscribe(id uint64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.subscribers) == 0 {
		a.log.Debug("Un-subscribe called in account store, no subscribers connected",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	if _, exists := a.subscribers[id]; exists {
		delete(a.subscribers, id)

		a.log.Debug("Un-subscribe called in account store, subscriber removed",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	a.log.Warn("Un-subscribe called in account store, subscriber does not exist",
		logging.Uint64("subscriber-id", id))

	return errors.New(fmt.Sprintf("Account store subscriber does not exist with id: %d", id))
}
