package staking

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrNoBalanceForParty = errors.New("no balance for party")
)

type Accounting struct {
	log      *logging.Logger
	accounts map[string]*StakingAccount
}

func NewAccounting(log *logging.Logger) *Accounting {
	return &Accounting{
		log:      log,
		accounts: map[string]*StakingAccount{},
	}
}

func (a *Accounting) AddEvent(evt *StakingEvent) {
	acc, ok := a.accounts[evt.Party]
	if !ok {
		acc = NewStakingAccount(evt.Party)
		a.accounts[evt.Party] = acc
	}

	// errors here do not really matter I'd say
	// they are either validation issue, that we can just log
	// but should never happen as things should be created properly
	// or errors from event being received in the wrong order
	// but that we cannot really prevent and that the account
	// would recover from by itself later on.
	if err := acc.AddEvent(evt); err != nil {
		a.log.Error("could not add event to staking account",
			logging.Error(err))
	}
}

func (a *Accounting) GetAvailableBalance(party string) (*num.Uint, error) {
	acc, ok := a.accounts[party]
	if !ok {
		return num.Zero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalance(), nil
}

func (a *Accounting) GetAvailableBalanceAt(
	party string, at time.Time) (*num.Uint, error) {
	acc, ok := a.accounts[party]
	if !ok {
		return num.Zero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalanceAt(at)
}

func (a *Accounting) GetAvailableBalanceInRange(
	party string, from, to time.Time) (*num.Uint, error) {
	acc, ok := a.accounts[party]
	if !ok {
		return num.Zero(), ErrNoBalanceForParty
	}

	return acc.GetAvailableBalanceInRange(from, to)
}
