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

const (
	noMarket    = "general"
)

var (
	ErrMarketNotFound      = errors.New("market accounts not found")
	ErrOwnerNotFound       = errors.New("owner has no known accounts")
	ErrAccountNotFound     = errors.New("account not found")
)

type Account struct {
	Config

	log          *logging.Logger
	badger       *badgerStore
	subscribers  map[uint64]chan []*types.Account
	subscriberID uint64
	mu           sync.Mutex
}

func NewAccounts(log *logging.Logger, c Config) (*Account, error) {
	// setup logger
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

// GetAccountByID - returns a given account by ID (if it exists, obviously)
//func (a *Account) GetAccountByID(id string) (*types.Account, error) {
//	return a.getAccountByID(nil, id)
//}

//func (a *Account) hasAccount(acc *types.Account) (bool, error) {
//	market := acc.MarketID
//	if market == "" {
//		market = NoMarket
//	}
//	key := a.badger.accountKey(acc.Owner, acc.Asset, market, acc.Type)
//	// set Id here - if account exists, we still want to return the full record with ID'
//	acc.Id = string(key)
//	err := a.badger.db.View(func(txn *badger.Txn) error {
//		account, err := txn.Get(key)
//		if err != nil {
//			return err
//		}
//		buf, err := account.ValueCopy(nil)
//		if err != nil {
//			return err
//		}
//		if err := proto.Unmarshal(buf, acc); err != nil {
//			a.log.Error(
//				"Failed to unmarshal account",
//				logging.Error(err),
//				logging.String("badger-key", string(key)),
//				logging.String("raw-bytes", string(buf)))
//			return err
//		}
//		return nil
//	})
//	// no errors, so key exists, and we got the account
//	if err == nil {
//		return true, nil
//	}
//	// key not found, account doesn't exist
//	if err == badger.ErrKeyNotFound {
//		return false, nil
//	}
//	// something went wrong
//	return false, err
//}


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

func (a *Account) GetByPartyAndMarket(partyID, marketID string) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountReferencePrefix(partyID, marketID, false)
	// an owner will have 3 accounts in a market at most, or a multiple thereof (based on assets), so cap of 3 is sensible
	return a.getAccountsForPrefix(keyPrefix, validFor, 3)
}

func (a *Account) GetByParty(partyID string) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountOwnerPrefix(partyID, false)
	// again, cap of 3 is reasonable: 3 per asset, per market, regardless of system/trader ownership
	return a.getAccountsForPrefix(keyPrefix, validFor, 3)
}

func (a *Account) GetByPartyAndType(partyID string, accType types.AccountType) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountTypePrefix(partyID, accType, false)
	return a.getAccountsForPrefix(keyPrefix, validFor, 0)
}

func (a *Account) getAccountsForPrefix(prefix, validFor []byte, capacity int) ([]*types.Account, error) {
	var err error
	ret := make([]*types.Account, 0, capacity)
	txn := a.badger.readTransaction()
	defer txn.Discard()
	it := a.badger.getIterator(txn, false)
	defer it.Close()
	keyBuf, accountBuf := []byte{}, []byte{}
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

//func (a *Account) UpdateBalance(id string, balance int64) error {
//	txn := a.badger.writeTransaction()
//	defer txn.Discard()
//	var account []byte
//	acc, err := a.getAccountByID(txn, id)
//	// internal func does the logging already
//	if err != nil {
//		return err
//	}
//	// update balance
//	acc.Balance = balance
//	// can't see how this would fail to marshal, but best check...
//	if account, err = proto.Marshal(acc); err != nil {
//		a.log.Error(
//			"Failed to marshal valid account record",
//			logging.Error(err),
//		)
//		return err
//	}
//	if err = txn.Set([]byte(id), account); err != nil {
//		a.log.Error(
//			"Failed to save updated account balance",
//			logging.String("account-id", id),
//			logging.Error(err),
//		)
//		return err
//	}
//	return txn.Commit()
//}

//func (a *Account) IncrementBalance(id string, inc int64) error {
//	txn := a.badger.writeTransaction()
//	defer txn.Discard()
//	var account []byte
//	acc, err := a.getAccountByID(txn, id)
//	if err != nil {
//		return err
//	}
//	// increment balance
//	acc.Balance += inc
//	if account, err = proto.Marshal(acc); err != nil {
//		a.log.Error(
//			"Failed to marshal account record",
//			logging.Error(err),
//			logging.String("account-id", id),
//		)
//		return err
//	}
//	if err = txn.Set([]byte(id), account); err != nil {
//		a.log.Error(
//			"Failed to update account balance",
//			logging.String("account-id", id),
//			logging.Int64("account-balance", acc.Balance),
//			logging.Error(err),
//		)
//		return err
//	}
//	return txn.Commit()
//}
//
//func (a *Account) getAccountByID(txn *badger.Txn, id string) (*types.Account, error) {
//	if txn == nil {
//		// default to read txn if no txn was provided
//		txn = a.badger.readTransaction()
//		defer txn.Discard()
//	}
//	item, err := txn.Get([]byte(id))
//	if err != nil {
//		a.log.Error(
//			"Failed to get account by ID",
//			logging.String("account-id", id),
//			logging.Error(err),
//		)
//		return nil, err
//	}
//	account, err := item.ValueCopy(nil)
//	if err != nil {
//		a.log.Error("Failed to get value for account item", logging.Error(err))
//		return nil, err
//	}
//	var acc types.Account
//	if err := proto.Unmarshal(account, &acc); err != nil {
//		a.log.Error(
//			"Failed to unmarshal account",
//			logging.String("account-id", id),
//			logging.String("account-raw", string(account)),
//			logging.Error(err),
//		)
//		return nil, err
//	}
//	return &acc, nil
//}

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

func (a *Account) parseBatch(accounts ...*types.Account) (map[string][]byte, error) {
	m := make(map[string][]byte, len(accounts)*5)
	for _, acc := range accounts {
		// for general accounts, a market isn't specified
		market := acc.MarketID
		if market == "" {
			market = noMarket
		}
		accKey := a.badger.accountKey(
			acc.Owner, acc.Asset, market, acc.Type,
		)
		acc.Id = string(accKey)
		refKey := a.badger.accountReferenceKey(
			acc.Owner, market, acc.Asset, acc.Type,
		)
		mrefKey := a.badger.accountMarketReferenceKey(
			market, acc.Owner, acc.Asset, acc.Type,
		)
		trefKey := a.badger.accountTypeReferenceKey(
			acc.Owner, market, acc.Asset, acc.Type,
		)
		assetRef := a.badger.accountAssetReferenceKey(
			acc.Owner, acc.Asset, market, acc.Type,
		)
		buf, err := proto.Marshal(acc)
		if err != nil {
			a.log.Error("unable to marshal account",
				logging.String("account-id", acc.Id),
				logging.Error(err),
			)
			return nil, err
		}
		// id is the key here
		m[acc.Id] = buf
		// reference key points to actual ID
		m[string(refKey)] = accKey
		m[string(mrefKey)] = accKey
		m[string(trefKey)] = accKey
		m[string(assetRef)] = accKey
	}
	return m, nil
}


func (a *Account) Subscribe(c chan []*types.Account) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.subscriberID += 1
	a.subscribers[a.subscriberID] = c

	a.log.Debug("Account subscriber added in account store",
		logging.Uint64("subscriber-id", a.subscriberID))

	return a.subscriberID
}

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
