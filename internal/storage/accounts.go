package storage

import (
	"fmt"
	"sync"

	cfgencoding "code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrMarketNotFound  = errors.New("no accounts found for market")
	ErrOwnerNotFound   = errors.New("no accounts found for party")
	ErrAccountNotFound = errors.New("account not found")
)

// Data structure representing a collateral account store
type Account struct {
	Config

	log          *logging.Logger
	badger       *badgerStore
	subscribers  map[uint64]chan []*types.Account
	subscriberID uint64
	mu           sync.Mutex
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
	generalAccounts, err := a.getAccountsForPrefix(keyPrefix, validFor, 3)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error loading general accounts for party: %s", partyID))
	}
	// Read all MARGIN accounts for party
	keyPrefix, validFor = a.badger.accountPartyPrefix(types.AccountType_MARGIN, partyID, false)
	marginAccounts, err := a.getAccountsForPrefix(keyPrefix, validFor, 3)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("error loading margin accounts for party: %s", partyID))
	}
	return append(generalAccounts, marginAccounts...), nil
}

// GetByPartyAndMarket will return all accounts (if available) relating to the provided party and market.
//  - Only MARGIN accounts are supported by this call, as they have market scope.
func (a *Account) GetByPartyAndMarket(accType types.AccountType, partyID string, marketID string) ([]*types.Account, error) {
	if accType != types.AccountType_MARGIN {
		return nil, errors.New("invalid type for query, only MARGIN accounts for a party and market supported")
	}
	keyPrefix, validFor := a.badger.accountMarketPartyPrefix(types.AccountType_MARGIN, marketID, partyID, false)
	return a.getAccountsForPrefix(keyPrefix, validFor, 3)
}

// GetByPartyAndType will return all accounts (if available) relating to the provided party and account type.
//  - GENERAL and MARGIN accounts are supported by this call, will return all MARGIN accounts for all markets.
func (a *Account) GetByPartyAndType(accType types.AccountType, partyID string) ([]*types.Account, error) {
	if accType != types.AccountType_GENERAL && accType != types.AccountType_MARGIN {
		return nil, errors.New("invalid type for query, only GENERAL and MARGIN accounts for a party supported")
	}
	keyPrefix, validFor := a.badger.accountPartyPrefix(accType, partyID, false)
	return a.getAccountsForPrefix(keyPrefix, validFor, 3)
}

// getAccountsForPrefix does the work of querying the badger store for creating key prefixes and loading values from
// the underlying based collateral account store.
func (a *Account) getAccountsForPrefix(prefix, validFor []byte, capacity int) ([]*types.Account, error) {
	var err error
	ret := make([]*types.Account, 0, capacity)

	txn := a.badger.readTransaction()
	defer txn.Discard()

	it := a.badger.getIterator(txn, false)
	defer it.Close()

	var accountBuf, keyBuf []byte
	for it.Seek(prefix); it.ValidForPrefix(validFor); it.Next() {
		if keyBuf, err = it.Item().ValueCopy(keyBuf); err != nil {
			return nil, err
		}
		item, err := txn.Get(keyBuf)
		if err != nil {
			return nil, err
		}
		if accountBuf, err = item.ValueCopy(accountBuf); err != nil {
			return nil, err
		}
		var acc types.Account
		if err := proto.Unmarshal(accountBuf, &acc); err != nil {
			a.log.Error(
				"Failed to unmarshal account",
				logging.String("account-id", string(keyBuf)),
				logging.Error(err),
			)
			return nil, err
		}
		ret = append(ret, &acc)
	}

	return ret, nil
}

// SaveBatch writes a slice of account changes to the underlying store.
func (a *Account) SaveBatch(accs []*types.Account) error {

	batch, err := a.parseBatch(accs...)
	if err != nil {
		return err
	}

	if logging.DebugLevel == a.log.GetLevel() {
		// todo: log out each account to be written to store, will include updates?
	}

	_, err = a.badger.writeBatch(batch)
	if err != nil {
		a.log.Error(
			"Unable to write accounts batch",
			logging.Error(err),
			logging.Int("batch-size", len(batch)))
		return err
	}

	if logging.DebugLevel == a.log.GetLevel() {
		a.log.Debug("Accounts store updated", logging.Int("batch-size", len(batch)))
	}

	a.notify(accs)

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

	//todo log whats happening here

	batch := make(map[string][]byte)
	for _, acc := range accounts {

		// todo: drop account ID unless its needed in the core

		// todo: is this required, safety checking?
		market := acc.MarketID
		if market == "" {
			a.log.Warn("Account has an empty marketID", logging.Account(*acc))
			if acc.Type != types.AccountType_GENERAL {
				a.log.Warn("Not of account type GENERAL")
			}
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
			generalIdKey := a.badger.accountGeneralIdKey(acc.Owner, acc.Asset)
			generalAssetKey := a.badger.accountAssetKey(acc.Asset, string(generalIdKey))
			batch[string(generalIdKey)] = buf
			batch[string(generalAssetKey)] = generalIdKey
		}
		// Check the type of account and write only the data/keys required for MARGIN accounts.
		if acc.Type == types.AccountType_MARGIN {
			marginIdKey := a.badger.accountMarginIdKey(acc.Owner, market, acc.Asset)
			marginMarketKey := a.badger.accountMarketKey(market, string(marginIdKey))
			marginAssetKey := a.badger.accountAssetKey(acc.Asset, string(marginIdKey))
			batch[string(marginIdKey)] = buf
			batch[string(marginMarketKey)] = marginIdKey
			batch[string(marginAssetKey)] = marginIdKey
		}
	}
	return batch, nil
}

// Subscribe to account store updates, any changes will be pushed out on this channel.
func (a *Account) Subscribe(c chan []*types.Account) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.subscriberID += 1
	a.subscribers[a.subscriberID] = c

	a.log.Debug("Account subscriber added in account store",
		logging.Uint64("subscriber-id", a.subscriberID))

	return a.subscriberID
}

//Unsubscribe from account store updates.
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

// DefaultAccountStoreOptions supplies default options we use for account stores.
// Currently we want to load account keys and value into RAM.
func DefaultAccountStoreOptions() ConfigOptions {
	opts := DefaultStoreOptions()
	opts.TableLoadingMode = cfgencoding.FileLoadingMode{FileLoadingMode: options.LoadToRAM}
	opts.ValueLogLoadingMode = cfgencoding.FileLoadingMode{FileLoadingMode: options.MemoryMap}
	return opts
}

// todo: ------ additional queries reqd?

//todo: do we need getByPartyAndAsset ?
//func (a *Account) GetAccountsByOwnerAndAsset(owner, asset string) ([]*types.Account, error) {
//	prefix, valid := a.badger.accountAssetPrefix(owner, asset, false)
//	return a.getAccountsForPrefix(prefix, valid, 3) // at least 3 accounts, I suppose
//}

//todo: do we need getByPartyMarketAndAsset ?
//func (a *Account) GetMarketAssetAccounts(owner, asset, market string) ([]*types.Account, error) {
//	prefix, valid := a.badger.accountKeyPrefix(owner, asset, market, false)
//	return a.getAccountsForPrefix(prefix, valid, 3)
//}

//todo: do we need GetByMarket(market string) - all accounts on a market for all parties?
//func (a *Account) GetMarketAccounts(market string) ([]*types.Account, error) {
//	keyPrefix, validFor := a.badger.accountMarketPrefix(market, false)
//	return a.getAccountsForPrefix(keyPrefix, validFor, 0)
//}
