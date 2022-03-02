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
	partyToStakeForEpoch map[time.Time]map[string]*num.Uint
	currentEpoch         *types.Epoch
}

func (t *StakingAccountStub) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if t.currentEpoch == nil || t.currentEpoch.Seq != epoch.Seq {
		t.currentEpoch = &epoch
		emptyT := time.Time{}
		if epoch.EndTime == emptyT {
			t.partyToStakeForEpoch[epoch.StartTime] = map[string]*num.Uint{}
			for p, s := range t.partyToStake {
				t.partyToStakeForEpoch[epoch.StartTime][p] = s.Clone()
			}
		}
	}
}

func (t *StakingAccountStub) OnEpochRestore(_ context.Context, _ types.Epoch) {}

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
	t.partyToStakeForEpoch[t.currentEpoch.StartTime][party] = t.partyToStake[party].Clone()
	return nil
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
	partyStake, ok := t.partyToStakeForEpoch[from][party]
	if !ok {
		return nil, fmt.Errorf("party not found")
	}
	return partyStake, nil
}
