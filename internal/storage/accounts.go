package storage

import (
	"sync"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	ErrDuplicateAccount    = errors.New("account already exists")
	ErrMarketAccountsExist = errors.New("accounts for market already exist")
	ErrMarketNotFound      = errors.New("market accounts not found")
	ErrOwnerNotFound       = errors.New("owner has no known accounts")
)

const (
	InsurancePool AccountType = "insurance"
	Settlement    AccountType = "settlement"
	Margin        AccountType = "margin"
	MarketTrader  AccountType = "market-trader"
	GeneralTrader AccountType = "general-trader"

	SystemOwner = "system"
)

// AccountType - defines some constants for account types
type AccountType string

// AccountRecord - placeholder type for an account, should be in protobuf, though...
type AccountRecord struct {
	ID       string
	Owner    string
	Balance  int64
	Market   string
	Type     AccountType // stuff like insurance, settlement,...
	ownerIdx int
}

type Account struct {
	*Config
	mu            *sync.RWMutex
	byMarketOwner map[string]map[string][]*AccountRecord
	byOwner       map[string][]*AccountRecord
	byID          map[string]*AccountRecord
}

func NewAccounts(c *Config) (*Account, error) {
	return &Account{
		Config:        c,
		mu:            &sync.RWMutex{},
		byMarketOwner: map[string]map[string][]*AccountRecord{},
		byOwner:       map[string][]*AccountRecord{},
		byID:          map[string]*AccountRecord{},
	}, nil
}

// Create an account, adds in all lists simultaneously
func (a *Account) Create(rec *AccountRecord) error {
	// default to new ID
	if rec.ID == "" {
		rec.ID = uuid.NewV4().String()
	}
	a.mu.Lock()
	if _, ok := a.byID[rec.ID]; ok {
		a.mu.Unlock()
		return ErrDuplicateAccount
	}
	cpy := *rec
	// pass a copy, avoid working on the argument from caller directly
	a.createAccount(&cpy)
	a.mu.Unlock()
	return nil
}

// internal create function, assumes mutex is locked correctly by caller
func (a *Account) createAccount(rec *AccountRecord) {
	a.byID[rec.ID] = rec
	if _, ok := a.byOwner[rec.Owner]; !ok {
		a.byOwner[rec.Owner] = []*AccountRecord{}
	}
	rec.ownerIdx = len(a.byOwner[rec.Owner])
	a.byOwner[rec.Owner] = append(a.byOwner[rec.Owner], rec)
	if _, ok := a.byMarketOwner[rec.Market]; !ok {
		a.byMarketOwner[rec.Market] = map[string][]*AccountRecord{
			rec.Owner: []*AccountRecord{},
		}
	}
	if _, ok := a.byMarketOwner[rec.Market][rec.Owner]; !ok {
		a.byMarketOwner[rec.Market][rec.Owner] = []*AccountRecord{}
	}
	a.byMarketOwner[rec.Market][rec.Owner] = append(a.byMarketOwner[rec.Market][rec.Owner], rec)
}

// CreateMarketAccounts - shortcut to quickly add the system balances for a market
func (a *Account) CreateMarketAccounts(market string, insuranceBalance int64) error {
	owner := SystemOwner
	a.mu.Lock()
	// add market entry, but do not set system accounts here, yet... ensure they don't exist yet
	if _, ok := a.byMarketOwner[market]; !ok {
		a.byMarketOwner[market] = map[string][]*AccountRecord{}
	}
	if _, ok := a.byMarketOwner[market][owner]; ok {
		a.mu.Unlock()
		return ErrMarketAccountsExist
	}
	a.byMarketOwner[market][owner] = []*AccountRecord{}
	// we can unlock here, we've set the byMarketOwner keys, duplicates are impossible
	a.mu.Unlock()
	accounts := []*AccountRecord{
		{
			Market:  market,
			Owner:   owner,
			Type:    InsurancePool,
			Balance: insuranceBalance,
		},
		{
			Market: market,
			Owner:  owner,
			Type:   Settlement,
		},
	}
	// add them in the usual way
	for _, account := range accounts {
		if err := a.Create(account); err != nil {
			// this is next to impossible, but ah well...
			return err
		}
	}
	return nil
}

// CreateTraderMarketAccounts - sets up accounts for trader for a particular market
// checks general accounts, and creates those, too if needed
func (a *Account) CreateTraderMarketAccounts(owner, market string) error {
	// does this trader actually have any accounts yet?
	accounts := []*AccountRecord{
		{
			ID:     uuid.NewV4().String(),
			Market: market,
			Owner:  owner,
			Type:   MarketTrader,
		},
	}
	a.mu.Lock()
	if _, ok := a.byOwner[owner]; !ok {
		// add general + margin account for trader
		accounts = append(
			accounts,
			&AccountRecord{
				ID:    uuid.NewV4().String(),
				Owner: owner,
				Type:  GeneralTrader,
			},
			&AccountRecord{
				ID:    uuid.NewV4().String(),
				Owner: owner,
				Type:  Margin,
			},
		)
	}
	for _, acc := range accounts {
		a.createAccount(acc)
	}
	a.mu.Unlock()
	return nil
}

func (a *Account) GetMarketAccounts(market string) ([]AccountRecord, error) {
	a.mu.RLock()
	byOwner, ok := a.byMarketOwner[market]
	if !ok {
		a.mu.RUnlock()
		return nil, ErrMarketNotFound
	}
	accounts := make([]AccountRecord, 0, len(a.byMarketOwner)*2) // each owner has 2 accounts -> for market, and margin, system has 2 (insurance + settlement)
	for owner, records := range byOwner {
		// this shouldn't be possible, but you never know
		if len(records) == 0 {
			continue
		}
		// system accounts are appended as they are
		if owner == SystemOwner {
			for _, r := range records {
				accounts = append(accounts, *r)
			}
			continue
		}
		var mTrader *AccountRecord
		// there should only be 1 here
		for _, r := range records {
			if r.Type == MarketTrader {
				mTrader = r
				break
			}
		}
		if mTrader == nil {
			continue
		}
		accounts = append(accounts, *mTrader)
		// get margin account
		ownerAcc := a.byOwner[owner]
		for _, acc := range ownerAcc {
			if acc.Type == Margin {
				accounts = append(accounts, *acc)
				break
			}
		}
	}
	a.mu.RUnlock()
	return accounts, nil
}

func (a *Account) GetMarketAccountsForOwner(market, owner string) ([]AccountRecord, error) {
	a.mu.RLock()
	owners, ok := a.byMarketOwner[market]
	if !ok {
		a.mu.RUnlock()
		return nil, ErrMarketNotFound
	}
	records, ok := owners[owner]
	if !ok {
		a.mu.RUnlock()
		return nil, ErrOwnerNotFound
	}
	accounts := make([]AccountRecord, 0, 2) // there's always 2 accounts given the market + owner
	// system owner -> copy both, non-system, there's only 1
	for _, record := range records {
		accounts = append(accounts, *record)
	}
	if owner != SystemOwner {
		for _, record := range a.byOwner[owner] {
			if record.Type == Margin {
				accounts = append(accounts, *record)
				break
			}
		}
	}
	a.mu.RUnlock()
	return accounts, nil
}

func (a *Account) GetAccountsForOwner(owner string) ([]AccountRecord, error) {
	a.mu.RLock()
	acc, ok := a.byOwner[owner]
	if !ok {
		a.mu.RUnlock()
		return nil, ErrOwnerNotFound
	}
	ret := make([]AccountRecord, 0, len(acc))
	for _, r := range acc {
		ret = append(ret, *r)
	}
	a.mu.RUnlock()
	return ret, nil
}
