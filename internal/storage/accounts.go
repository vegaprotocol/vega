package storage

import (
	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

var (
	ErrDuplicateAccount    = errors.New("account already exists")
	ErrMarketAccountsExist = errors.New("accounts for market already exist")
	ErrMarketNotFound      = errors.New("market accounts not found")
	ErrOwnerNotFound       = errors.New("owner has no known accounts")
	ErrAccountNotFound     = errors.New("account not found")
)

const (
	SystemOwner = "system"
	NoMarket    = "general"
)

type accountRecord struct {
	*types.Account
	ownerIdx int
}

type Account struct {
	Config

	log    *logging.Logger
	badger *badgerStore
}

func NewAccounts(log *logging.Logger, c Config) (*Account, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(c.Level.Get())

	if err := InitStoreDirectory(c.AccountStoreDirPath); err != nil {
		return nil, errors.Wrap(err, "error on init badger database for account storage")
	}
	db, err := badger.Open(badgerOptionsFromConfig(c.BadgerOptions, c.AccountStoreDirPath, log))
	if err != nil {
		return nil, errors.Wrap(err, "error opening badger database for account storage")
	}

	return &Account{
		log:    log,
		Config: c,
		badger: &badgerStore{db: db},
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

// Create an account, adds in all lists simultaneously
func (a *Account) Create(rec *types.Account) error {
	records, err := a.createAccountRecords(rec)
	if err != nil {
		return err
	}
	if _, err := a.badger.writeBatch(records); err != nil {
		a.log.Error(
			"Failed to create the given account",
			logging.String("account-id", rec.Id),
			logging.Error(err),
		)
		return err
	}
	return nil
}

// GetAccountByID - returns a given account by ID (if it exists, obviously)
func (a *Account) GetAccountByID(id string) (*types.Account, error) {
	return a.getAccountByID(nil, id)
}

func (a *Account) createAccounts(accounts ...*types.Account) error {
	if len(accounts) == 0 {
		return nil
	}
	records, err := a.createAccountRecords(accounts...)
	if err != nil {
		return err
	}
	if _, err := a.badger.writeBatch(records); err != nil {
		a.log.Error(
			"Failed to create accounts",
			logging.Error(err),
		)
		return err
	}
	return nil
}

func (a *Account) hasAccount(acc *types.Account) (bool, error) {
	market := acc.MarketID
	if market == "" {
		market = NoMarket
	}
	key := a.badger.accountKey(acc.Owner, acc.Asset, market, acc.Type)
	// set Id here - if account exists, we still want to return the full record with ID'
	acc.Id = string(key)
	err := a.badger.db.View(func(txn *badger.Txn) error {
		account, err := txn.Get(key)
		if err != nil {
			return err
		}
		buf, err := account.ValueCopy(nil)
		if err != nil {
			return err
		}
		if err := proto.Unmarshal(buf, acc); err != nil {
			a.log.Error(
				"Failed to unmarshal account",
				logging.Error(err),
				logging.String("badger-key", string(key)),
				logging.String("raw-bytes", string(buf)))
			return err
		}
		return nil
	})
	// no errors, so key exists, and we got the account
	if err == nil {
		return true, nil
	}
	// key not found, account doesn't exist
	if err == badger.ErrKeyNotFound {
		return false, nil
	}
	// something went wrong
	return false, err
}

func (a *Account) createAccountRecords(accounts ...*types.Account) (map[string][]byte, error) {
	m := make(map[string][]byte, len(accounts)*5) // each account has its key + 1 reference key, so map == nr of accounts * 2
	for _, acc := range accounts {
		// for general accounts, a market isn't specified
		market := acc.MarketID
		if market == "" {
			market = NoMarket
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

func (a *Account) CreateMarketAccounts(market string, insuranceBalance int64) ([]*types.Account, error) {
	// all market accounts that the system should have available to it
	accounts := []*types.Account{
		{
			Owner:    SystemOwner,
			MarketID: market,
			Asset:    string(market[:3]),
			Type:     types.AccountType_INSURANCE,
			Balance:  insuranceBalance,
		},
		{
			Owner:    SystemOwner,
			MarketID: NoMarket,
			Asset:    string(market[:3]),
			Type:     types.AccountType_GENERAL,
		},
		{
			Owner:    SystemOwner,
			MarketID: market,
			Asset:    string(market[:3]),
			Type:     types.AccountType_SETTLEMENT,
		},
	}
	// This should probably be a single transaction
	create := make([]*types.Account, 0, len(accounts))
	for _, acc := range accounts {
		ok, err := a.hasAccount(acc)
		if err != nil {
			return nil, err
		}
		if !ok {
			create = append(create, acc)
		}
	}
	if err := a.createAccounts(create...); err != nil {
		return nil, err
	}
	// don't return just the ones we've created, we should return all the accounts we need
	return accounts, nil
}

// CreateTraderMarketAccounts - sets up accounts for trader for a particular market
// checks general accounts, and creates those, too if needed
func (a *Account) CreateTraderMarketAccounts(owner, market string) ([]*types.Account, error) {
	// does this trader actually have any accounts yet?
	accounts := []*types.Account{
		{
			MarketID: market,
			Owner:    owner,
			Asset:    string(market[:3]),
			Type:     types.AccountType_MARKET,
		},
		{
			MarketID: market,
			Asset:    string(market[:3]),
			Owner:    owner,
			Type:     types.AccountType_MARGIN,
		},
		{
			MarketID: NoMarket,
			Owner:    owner,
			Asset:    string(market[:3]),
			Type:     types.AccountType_GENERAL,
		},
	}
	// Again, probably better to put this in a transaction, even though accounts are created
	// in a deterministic flow (sequential), and as such, this is safe
	create := make([]*types.Account, 0, len(accounts))
	for _, acc := range accounts {
		ok, err := a.hasAccount(acc)
		if err != nil {
			return nil, err
		}
		if !ok {
			// no errors returned by check, no account found
			create = append(create, acc)
		}
	}
	if err := a.createAccounts(create...); err != nil {
		return nil, err
	}
	// again, return not just the created accounts, return all of them
	return accounts, nil
}

func (a *Account) GetAccountsByOwnerAndAsset(owner, asset string) ([]*types.Account, error) {
	prefix, valid := a.badger.accountAssetPrefix(owner, asset, false)
	return a.getByReference(prefix, valid, 3) // at least 3 accounts, I suppose
}

func (a *Account) GetMarketAssetAccounts(owner, asset, market string) ([]*types.Account, error) {
	prefix, valid := a.badger.accountKeyPrefix(owner, asset, market, false)
	return a.getByReference(prefix, valid, 3)
}

func (a *Account) GetMarketAccounts(market string) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountMarketPrefix(market, false)
	return a.getByReference(keyPrefix, validFor, 0)
}

func (a *Account) GetMarketAccountsForOwner(market, owner string) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountReferencePrefix(owner, market, false)
	// an owner will have 3 accounts in a market at most, or a multiple thereof (based on assets), so cap of 3 is sensible
	return a.getByReference(keyPrefix, validFor, 3)
}

func (a *Account) GetAccountsForOwner(owner string) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountOwnerPrefix(owner, false)
	// again, cap of 3 is reasonable: 3 per asset, per market, regardless of system/trader ownership
	return a.getByReference(keyPrefix, validFor, 3)
}

func (a *Account) GetAccountsForOwnerByType(owner string, accType types.AccountType) ([]*types.Account, error) {
	keyPrefix, validFor := a.badger.accountTypePrefix(owner, accType, false)
	return a.getByReference(keyPrefix, validFor, 0)
}

func (a *Account) getByReference(prefix, validFor []byte, capacity int) ([]*types.Account, error) {
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

func (a *Account) UpdateBalance(id string, balance int64) error {
	txn := a.badger.writeTransaction()
	defer txn.Discard()
	var account []byte
	acc, err := a.getAccountByID(txn, id)
	// internal func does the logging already
	if err != nil {
		return err
	}
	// update balance
	acc.Balance = balance
	// can't see how this would fail to marshal, but best check...
	if account, err = proto.Marshal(acc); err != nil {
		a.log.Error(
			"Failed to marshal valid account record",
			logging.Error(err),
		)
		return err
	}
	if err = txn.Set([]byte(id), account); err != nil {
		a.log.Error(
			"Failed to save updated account balance",
			logging.String("account-id", id),
			logging.Error(err),
		)
		return err
	}
	return txn.Commit()
}

func (a *Account) IncrementBalance(id string, inc int64) error {
	txn := a.badger.writeTransaction()
	defer txn.Discard()
	var account []byte
	acc, err := a.getAccountByID(txn, id)
	if err != nil {
		return err
	}
	// increment balance
	acc.Balance += inc
	if account, err = proto.Marshal(acc); err != nil {
		a.log.Error(
			"Failed to marshal account record",
			logging.Error(err),
			logging.String("account-id", id),
		)
		return err
	}
	if err = txn.Set([]byte(id), account); err != nil {
		a.log.Error(
			"Failed to update account balance",
			logging.String("account-id", id),
			logging.Int64("account-balance", acc.Balance),
			logging.Error(err),
		)
		return err
	}
	return txn.Commit()
}

func (a *Account) getAccountByID(txn *badger.Txn, id string) (*types.Account, error) {
	if txn == nil {
		// default to read txn if no txn was provided
		txn = a.badger.readTransaction()
		defer txn.Discard()
	}
	item, err := txn.Get([]byte(id))
	if err != nil {
		a.log.Error(
			"Failed to get account by ID",
			logging.String("account-id", id),
			logging.Error(err),
		)
		return nil, err
	}
	account, err := item.ValueCopy(nil)
	if err != nil {
		a.log.Error("Failed to get value for account item", logging.Error(err))
		return nil, err
	}
	var acc types.Account
	if err := proto.Unmarshal(account, &acc); err != nil {
		a.log.Error(
			"Failed to unmarshal account",
			logging.String("account-id", id),
			logging.String("account-raw", string(account)),
			logging.Error(err),
		)
		return nil, err
	}
	return &acc, nil
}
