package buffer

import (
	"sync"

	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/account_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/buffer AccountStore
type AccountStore interface {
	SaveBatch([]types.Account) error
}

type accountKey struct {
	owner, marketID, asset string
	ty                     types.AccountType
}

type Account struct {
	store AccountStore
	accs  map[accountKey]int64
	mu    sync.Mutex
}

func NewAccount(store AccountStore) *Account {
	return &Account{
		store: store,
		accs:  map[accountKey]int64{},
	}
}

func (a *Account) Add(owner, marketID, asset string, ty types.AccountType, balance int64) {
	key := accountKey{owner, marketID, asset, ty}
	a.mu.Lock()
	a.accs[key] = balance
	a.mu.Unlock()
}

func (a *Account) Flush() error {
	a.mu.Lock()
	accsToBatch := a.accs
	a.accs = map[accountKey]int64{}
	a.mu.Unlock()

	accs := make([]types.Account, 0, len(accsToBatch))
	for k, v := range accsToBatch {
		// if marketID != empty, this is a a market
		// and the owner is system
		if len(k.owner) <= 0 && len(k.marketID) > 0 {
			k.owner = storage.SystemOwner
		}
		accs = append(accs, types.Account{
			Owner:    k.owner,
			MarketID: k.marketID,
			Asset:    k.asset,
			Type:     k.ty,
			Balance:  v,
		})
	}
	return a.store.SaveBatch(accs)
}
