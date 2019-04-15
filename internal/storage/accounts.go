package storage

import (
	"sync"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
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
)

type accountRecord struct {
	*types.Account
	ownerIdx int
}

type Account struct {
	*Config
	mu            *sync.RWMutex
	byMarketOwner map[string]map[string][]*accountRecord
	byOwner       map[string][]*accountRecord
	byID          map[string]*accountRecord
}

func NewAccounts(c *Config) (*Account, error) {
	return &Account{
		Config:        c,
		mu:            &sync.RWMutex{},
		byMarketOwner: map[string]map[string][]*accountRecord{},
		byOwner:       map[string][]*accountRecord{},
		byID:          map[string]*accountRecord{},
	}, nil
}

// Create an account, adds in all lists simultaneously
func (a *Account) Create(rec *types.Account) error {
	// default to new ID
	if rec.Id == "" {
		rec.Id = uuid.NewV4().String()
	}
	a.mu.Lock()
	if _, ok := a.byID[rec.Id]; ok {
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
func (a *Account) createAccount(cpy *types.Account) {
	rec := &accountRecord{
		Account: cpy,
	}
	a.byID[rec.Id] = rec
	if _, ok := a.byOwner[rec.Owner]; !ok {
		a.byOwner[rec.Owner] = []*accountRecord{}
	}
	// use an embedded type here to keep track of this
	rec.ownerIdx = len(a.byOwner[rec.Owner])
	a.byOwner[rec.Owner] = append(a.byOwner[rec.Owner], rec)
	if _, ok := a.byMarketOwner[rec.Market]; !ok {
		a.byMarketOwner[rec.Market] = map[string][]*accountRecord{
			rec.Owner: []*accountRecord{},
		}
	}
	if _, ok := a.byMarketOwner[rec.Market][rec.Owner]; !ok {
		a.byMarketOwner[rec.Market][rec.Owner] = []*accountRecord{}
	}
	a.byMarketOwner[rec.Market][rec.Owner] = append(a.byMarketOwner[rec.Market][rec.Owner], rec)
}

// CreateMarketAccounts - shortcut to quickly add the system balances for a market
func (a *Account) CreateMarketAccounts(market string, insuranceBalance int64) error {
	owner := SystemOwner
	a.mu.Lock()
	// add market entry, but do not set system accounts here, yet... ensure they don't exist yet
	if _, ok := a.byMarketOwner[market]; !ok {
		a.byMarketOwner[market] = map[string][]*accountRecord{}
	}
	if _, ok := a.byMarketOwner[market][owner]; ok {
		a.mu.Unlock()
		return ErrMarketAccountsExist
	}
	a.byMarketOwner[market][owner] = []*accountRecord{}
	// we can unlock here, we've set the byMarketOwner keys, duplicates are impossible
	a.mu.Unlock()
	accounts := []*types.Account{
		{
			Market:  market,
			Owner:   owner,
			Type:    types.AccountType_INSURANCE,
			Balance: insuranceBalance,
		},
		{
			Market: market,
			Owner:  owner,
			Type:   types.AccountType_SETTLEMENT,
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
	accounts := []*types.Account{
		{
			Id:     uuid.NewV4().String(),
			Market: market,
			Owner:  owner,
			Type:   types.AccountType_MARKET,
		},
	}
	a.mu.Lock()
	if _, ok := a.byOwner[owner]; !ok {
		// add general + margin account for trader
		accounts = append(
			accounts,
			&types.Account{
				Id:    uuid.NewV4().String(),
				Owner: owner,
				Type:  types.AccountType_GENERAL,
			},
			&types.Account{
				Id:    uuid.NewV4().String(),
				Owner: owner,
				Type:  types.AccountType_MARGIN,
			},
		)
	}
	for _, acc := range accounts {
		a.createAccount(acc)
	}
	a.mu.Unlock()
	return nil
}

func (a *Account) GetMarketAccounts(market string) ([]types.Account, error) {
	a.mu.RLock()
	byOwner, ok := a.byMarketOwner[market]
	if !ok {
		a.mu.RUnlock()
		return nil, ErrMarketNotFound
	}
	accounts := make([]types.Account, 0, len(a.byMarketOwner)*2) // each owner has 2 accounts -> for market, and margin, system has 2 (insurance + settlement)
	for owner, records := range byOwner {
		// this shouldn't be possible, but you never know
		if len(records) == 0 {
			continue
		}
		// system accounts are appended as they are
		if owner == SystemOwner {
			for _, r := range records {
				accounts = append(accounts, *r.Account)
			}
			continue
		}
		var mTrader *types.Account
		// there should only be 1 here
		for _, r := range records {
			if r.Type == types.AccountType_MARKET {
				mTrader = r.Account
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
			if acc.Type == types.AccountType_MARGIN {
				accounts = append(accounts, *acc.Account)
				break
			}
		}
	}
	a.mu.RUnlock()
	return accounts, nil
}

func (a *Account) GetMarketAccountsForOwner(market, owner string) ([]types.Account, error) {
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
	accounts := make([]types.Account, 0, 2) // there's always 2 accounts given the market + owner
	// system owner -> copy both, non-system, there's only 1
	for _, record := range records {
		accounts = append(accounts, *record.Account)
	}
	if owner != SystemOwner {
		for _, record := range a.byOwner[owner] {
			if record.Type == types.AccountType_MARKET {
				accounts = append(accounts, *record.Account)
				break
			}
		}
	}
	a.mu.RUnlock()
	return accounts, nil
}

func (a *Account) GetAccountsForOwner(owner string) ([]types.Account, error) {
	a.mu.RLock()
	acc, ok := a.byOwner[owner]
	if !ok {
		a.mu.RUnlock()
		return nil, ErrOwnerNotFound
	}
	ret := make([]types.Account, 0, len(acc))
	for _, r := range acc {
		ret = append(ret, *r.Account)
	}
	a.mu.RUnlock()
	return ret, nil
}

func (a *Account) GetAccountsForOwnerByType(owner string, accType types.AccountType) (*types.Account, error) {
	a.mu.RLock()
	acc, ok := a.byOwner[owner]
	if !ok {
		a.mu.RUnlock()
		return nil, ErrOwnerNotFound
	}
	for _, ac := range acc {
		if ac.Type == accType {
			cpy := *ac.Account
			a.mu.RUnlock()
			return &cpy, nil
		}
	}
	a.mu.RUnlock()
	return nil, ErrAccountNotFound
}

func (a *Account) UpdateBalance(id string, balance int64) error {
	a.mu.Lock()
	acc, ok := a.byID[id]
	if !ok {
		a.mu.Unlock()
		return ErrAccountNotFound
	}
	acc.Balance = balance
	a.mu.Unlock()
	return nil
}

func (a *Account) IncrementBalance(id string, inc int64) error {
	a.mu.Lock()
	acc, ok := a.byID[id]
	if !ok {
		a.mu.Unlock()
		return ErrAccountNotFound
	}
	acc.Balance += inc
	a.mu.Unlock()
	return nil
}
