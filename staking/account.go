package staking

import (
	"errors"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

var (
	ErrEventAlreadyExists = errors.New("event already exists")
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInvalidEventKind   = errors.New("invalid event kind")
	ErrMissingEventID     = errors.New("missing event id")
	ErrMissingTimestamp   = errors.New("missing timestamp")
	ErrNegativeBalance    = errors.New("negative balance")
	ErrInvalidParty       = errors.New("invalid party")
)

type StakingAccount struct {
	Party   string
	Balance *num.Uint
	Events  []*types.StakeLinking
}

func NewStakingAccount(party string) *StakingAccount {
	return &StakingAccount{
		Party:   party,
		Balance: num.Zero(),
		Events:  []*types.StakeLinking{},
	}
}

func (s *StakingAccount) validateEvent(evt *types.StakeLinking) error {
	if evt.Amount == nil || evt.Amount.IsZero() {
		return ErrInvalidAmount
	}
	if evt.Type != types.StakeLinkingTypeDeposited && evt.Type != types.StakeLinkingTypeRemoved {
		return ErrInvalidEventKind
	}
	if evt.TS <= 0 {
		return ErrMissingTimestamp
	}
	if len(evt.ID) <= 0 {
		return ErrMissingEventID
	}
	if evt.Party != s.Party {
		return ErrInvalidParty
	}

	for _, v := range s.Events {
		if evt.ID == v.ID {
			return ErrEventAlreadyExists
		}
	}

	return nil
}

// AddEvent will add a new event to the account
func (s *StakingAccount) AddEvent(evt *types.StakeLinking) error {
	if err := s.validateEvent(evt); err != nil {
		return err
	}

	// save the new events
	s.insertSorted(evt)

	// now update the ongoing balance
	return s.computeOngoingBalance()
}

func (s *StakingAccount) GetAvailableBalance() *num.Uint {
	return s.Balance.Clone()
}

func (s *StakingAccount) GetAvailableBalanceAt(at time.Time) (*num.Uint, error) {
	atUnix := at.UnixNano()
	return s.calculateBalance(func(evt *types.StakeLinking) bool {
		return evt.TS <= atUnix
	})
}

// GetAvailableBalanceInRange could return a negative balance
// if some event are still expected to be received from the bridge
func (s *StakingAccount) GetAvailableBalanceInRange(from, to time.Time) (*num.Uint, error) {
	// first compute the balance before the from time.
	balance, err := s.GetAvailableBalanceAt(from)
	if err != nil {
		return num.Zero(), err
	}

	// now we have the balance at the from time.
	// we will want to check how much was added / removed
	// during the epoch, and make sure that the initial
	// balance is still covered
	var (
		fromUnix  = from.UnixNano()
		toUnix    = to.UnixNano()
		deposited = num.Zero()
		withdrawn = num.Zero()
	)
	for i := 0; i < len(s.Events) && s.Events[i].TS <= toUnix; i++ {
		if s.Events[i].TS > fromUnix {
			evt := s.Events[i]
			switch evt.Type {
			case types.StakeLinkingTypeDeposited:
				deposited.Add(deposited, evt.Amount)
			case types.StakeLinkingTypeRemoved:
				withdrawn.Add(withdrawn, evt.Amount)
			}
		}
	}

	// now we'll check if what was deposited during the epoch
	// cover what we have at the start of it. and see
	if withdrawn.GT(deposited) {
		// we withdrawn more than we deposited, so we'll deduce
		// the difference to the stake to be returned

		withdrawn = withdrawn.Sub(withdrawn, deposited)
		if withdrawn.GT(balance) {
			return num.Zero(), nil
		}

		return balance.Sub(balance, withdrawn), nil
	}

	return balance, nil
}

// computeOnGoingBalance can return only 1 error which would
// be ErrNegativeBalancem, while this sounds bad, it can happen
// because of event being processed out of order but we can't
// really prevent that, and would have to wait for the network
// to have seen all events before getting a positive balance.
func (s *StakingAccount) computeOngoingBalance() error {
	balance, err := s.calculateBalance(func(evt *types.StakeLinking) bool {
		return true
	})
	s.Balance.Set(balance)
	return err
}

func (s *StakingAccount) insertSorted(evt *types.StakeLinking) {
	s.Events = append(s.Events, evt)
	// sort anyway, but we would expect the events to come in a sorted manner
	sort.SliceStable(s.Events, func(i, j int) bool {
		// check if timestamps are the same
		if s.Events[i].TS == s.Events[j].TS {
			// now we want to put deposit first to avoid any remove
			// event before a withdraw
			if s.Events[i].Type == types.StakeLinkingTypeRemoved && s.Events[j].Type == types.StakeLinkingTypeDeposited {
				// we return false so they can switched
				return false
			}
			// any other case is find to be as they are
			return true
		}

		return s.Events[i].TS < s.Events[j].TS
	})
}

type timeFilter func(*types.StakeLinking) bool

func (s *StakingAccount) calculateBalance(f timeFilter) (*num.Uint, error) {
	balance := num.Zero()
	for _, evt := range s.Events {
		if f(evt) {
			switch evt.Type {
			case types.StakeLinkingTypeDeposited:
				balance.Add(balance, evt.Amount)
			case types.StakeLinkingTypeRemoved:
				if balance.LT(evt.Amount) {
					return num.Zero(), ErrNegativeBalance
				}
				balance.Sub(balance, evt.Amount)
			}
		}
	}
	return balance, nil
}
