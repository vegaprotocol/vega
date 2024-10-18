// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package stubs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type StakingAccountStub struct {
	partyToStake         map[string]*num.Uint
	partyToStakeForEpoch map[int64]map[string]*num.Uint
	currentEpoch         *types.Epoch
}

func (t *StakingAccountStub) AddEvent(ctx context.Context, evt *types.StakeLinking) {}

func (t *StakingAccountStub) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if t.currentEpoch == nil || t.currentEpoch.Seq != epoch.Seq {
		t.currentEpoch = &epoch
		emptyT := time.Time{}
		if epoch.EndTime == emptyT {
			t.partyToStakeForEpoch[epoch.StartTime.UnixNano()] = map[string]*num.Uint{}
			for p, s := range t.partyToStake {
				t.partyToStakeForEpoch[epoch.StartTime.UnixNano()][p] = s.Clone()
			}
		}
	}
}

func (t *StakingAccountStub) OnEpochRestore(_ context.Context, _ types.Epoch) {}

func (t *StakingAccountStub) IncrementBalance(party string, amount *num.Uint) error {
	if _, ok := t.partyToStake[party]; !ok {
		t.partyToStake[party] = num.UintZero()
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
	t.partyToStakeForEpoch[t.currentEpoch.StartTime.UnixNano()][party] = t.partyToStake[party].Clone()
	return nil
}

func NewStakingAccountStub() *StakingAccountStub {
	return &StakingAccountStub{
		partyToStake:         make(map[string]*num.Uint),
		partyToStakeForEpoch: make(map[int64]map[string]*num.Uint),
	}
}

func (t *StakingAccountStub) GetAllStakingParties() []string {
	keys := make([]string, 0, len(t.partyToStake))
	for k := range t.partyToStake {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (t *StakingAccountStub) GetAvailableBalance(party string) (*num.Uint, error) {
	ret, ok := t.partyToStake[party]
	if !ok {
		return num.UintZero(), fmt.Errorf("party not found")
	}
	return ret, nil
}

func (t *StakingAccountStub) GetAvailableBalanceInRange(party string, from, _ time.Time) (*num.Uint, error) {
	partyStake, ok := t.partyToStakeForEpoch[from.UnixNano()][party]
	if !ok {
		return num.UintZero(), fmt.Errorf("party not found")
	}
	return partyStake, nil
}
