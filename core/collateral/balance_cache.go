package collateral

import (
	"sync"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type balanceCache struct {
	mu            sync.RWMutex
	partyCache    map[string]map[string]*num.Uint
	accountIDFunc func(marketID, partyID, asset string, ty types.AccountType) string
	updated       map[string]struct{}
}

var accountTypes = map[types.AccountType]struct{}{
	types.AccountTypeGeneral:     {},
	types.AccountTypeMargin:      {},
	types.AccountTypeOrderMargin: {},
	types.AccountTypeBond:        {},
	types.AccountTypeHolding:     {}}

func NewBalanceCache(accountIDFunc func(string, string, string, types.AccountType) string) *balanceCache {
	return &balanceCache{
		partyCache: map[string]map[string]*num.Uint{},
		updated:    map[string]struct{}{},
	}
}

func (bc *balanceCache) accountUpdated(party string, accountID string, accountType types.AccountType) {
	if party == systemOwner {
		return
	}
	if _, ok := accountTypes[accountType]; ok {
		bc.updated[accountID] = struct{}{}
	}
}

func (bc *balanceCache) update(accounts []*types.Account) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	coldStart := len(bc.partyCache) == 0

	for _, acc := range accounts {
		_, relevantAccount := accountTypes[acc.Type]
		if _, ok := bc.updated[acc.ID]; ok || (coldStart && relevantAccount) {
			if _, ok := bc.partyCache[acc.Owner]; !ok && acc.Owner != systemOwner {
				bc.partyCache[acc.Owner] = map[string]*num.Uint{}
			}
			bc.partyCache[acc.Owner][acc.ID] = acc.Balance.Clone()
		}
	}
	bc.updated = map[string]struct{}{}
}

func (bc *balanceCache) getPartyBalance(asset, market, party string) *num.Uint {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	balance := num.UintZero()
	pc, ok := bc.partyCache[party]
	if !ok {
		return balance
	}
	for tp := range accountTypes {
		if accountBalance, ok := pc[bc.accountIDFunc(market, party, asset, tp)]; ok {
			balance.AddSum(accountBalance)
		}
	}
	return balance
}
