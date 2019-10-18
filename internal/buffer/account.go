package buffer

import (
	types "code.vegaprotocol.io/vega/proto"
)

// AccountStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/buffer AccountStore
type AccountStore interface {
	SaveBatch([]*types.Account) error
}

// Account is a buffer for the accounts in vega
type Account struct {
	store AccountStore
	accs  map[string]types.Account
}

// NewAccount instanciate a new account buffer
func NewAccount(store AccountStore) *Account {
	return &Account{
		store: store,
		accs:  map[string]types.Account{},
	}
}

// Add adds a new account to the buffer
func (a *Account) Add(acc types.Account) {
	key := acc.Id // set the key to the internal account type, set by the collateral
	acc.Id = ""   // reset the actual id to be set by the storage later on
	a.accs[key] = acc
}

// Flush will save all the buffered account to store
func (a *Account) Flush() error {
	accsToBatch := a.accs
	a.accs = map[string]types.Account{}

	accs := make([]*types.Account, 0, len(accsToBatch))
	for _, v := range accsToBatch {
		v := v
		accs = append(accs, &v)
	}
	return a.store.SaveBatch(accs)
}
