package buffer

import (
	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/buffer AccountStore
type AccountStore interface {
	SaveBatch([]*types.Account) error
}

type Account struct {
	store AccountStore
	accs  map[string]types.Account
}

func NewAccount(store AccountStore) *Account {
	return &Account{
		store: store,
		accs:  map[string]types.Account{},
	}
}

func (a *Account) Add(acc types.Account) {
	key := acc.Id // set the key to the internal account type, set by the colateral
	acc.Id = ""   // reset the actual id to be set by the storage later on
	a.accs[key] = acc
}

func (a *Account) Flush() error {
	accsToBatch := a.accs
	a.accs = map[string]types.Account{}

	accs := make([]*types.Account, 0, len(accsToBatch))
	for _, v := range accsToBatch {
		accs = append(accs, &v)
	}
	return a.store.SaveBatch(accs)
}
