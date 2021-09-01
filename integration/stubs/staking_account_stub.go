package stubs

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/types/num"
)

type StakingAccountStub struct {
	partyToStake         map[string]*num.Uint
	partyToStakeForEpoch map[time.Time]map[string]*num.Uint
}

func NewStakingAccountStub() *StakingAccountStub {
	return &StakingAccountStub{
		partyToStake:         make(map[string]*num.Uint),
		partyToStakeForEpoch: make(map[time.Time]map[string]*num.Uint),
	}
}

func (t *StakingAccountStub) GetAvailableBalance(party string) (*num.Uint, error) {
	ret, ok := t.partyToStake[party]
	if !ok {
		return nil, fmt.Errorf("party not found")
	}
	return ret, nil
}

func (t *StakingAccountStub) GetAvailableBalanceInRange(party string, from, to time.Time) (*num.Uint, error) {
	ret, ok := t.partyToStakeForEpoch[from]
	if !ok {
		return nil, fmt.Errorf("time not found")
	}

	p, ok := ret[party]
	if !ok {
		return nil, fmt.Errorf("party not found")
	}

	return p, nil
}
