package stubs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type StakingAccountStub struct {
	partyToStake         map[string]*num.Uint
	partyToStakeForEpoch map[uint64]map[string]*num.Uint
	currentEpoch         uint64
}

func (t *StakingAccountStub) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	t.currentEpoch = epoch.Seq
	emptyT := time.Time{}
	if epoch.EndTime == emptyT {
		t.partyToStakeForEpoch[epoch.Seq] = map[string]*num.Uint{}
		for p, s := range t.partyToStake {
			t.partyToStakeForEpoch[epoch.Seq][p] = s
		}
	}
}

func (t *StakingAccountStub) IncrementBalance(party string, amount *num.Uint) error {
	if _, ok := t.partyToStake[party]; !ok {
		t.partyToStake[party] = num.Zero()
	}
	t.partyToStake[party].AddSum(amount)

	return nil
}

func (t *StakingAccountStub) DecrementBalance(party string, amount *num.Uint) error {
	if _, ok := t.partyToStake[party]; !ok {
		return errors.New("party staking accoung is missing")
	}
	if t.partyToStake[party].LT(amount) {
		return errors.New("incorrect balance for unstaking")
	}
	t.partyToStake[party] = t.partyToStake[party].Sub(t.partyToStake[party], amount)
	t.partyToStakeForEpoch[t.currentEpoch][party] = t.partyToStake[party]
	return nil
}

func NewStakingAccountStub() *StakingAccountStub {
	return &StakingAccountStub{
		partyToStake:         make(map[string]*num.Uint),
		partyToStakeForEpoch: make(map[uint64]map[string]*num.Uint),
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
	//TODO this should be gix to get the minimum balance of the account during the epoch
	ret, ok := t.partyToStake[party]
	if !ok {
		return nil, fmt.Errorf("party not found")
	}
	return ret, nil
}
