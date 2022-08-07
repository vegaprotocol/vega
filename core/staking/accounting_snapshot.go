// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package staking

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

	"code.vegaprotocol.io/vega/libs/proto"
)

var accountsKey = (&types.PayloadStakingAccounts{}).Key()

type accountingSnapshotState struct {
	serialised  []byte
	changed     bool
	isRestoring bool
}

func (a *Accounting) serialiseStakingAccounts() ([]byte, error) {
	accounts := make([]*types.StakingAccount, 0, len(a.hashableAccounts))
	a.log.Debug("serialsing staking accounts", logging.Int("n", len(a.hashableAccounts)))
	for _, acc := range a.hashableAccounts {
		accounts = append(accounts,
			&types.StakingAccount{
				Party:   acc.Party,
				Balance: acc.Balance,
				Events:  acc.Events,
			})
	}

	pl := types.Payload{
		Data: &types.PayloadStakingAccounts{
			StakingAccounts: &types.StakingAccounts{
				Accounts:                accounts,
				StakingAssetTotalSupply: a.stakingAssetTotalSupply.Clone(),
			},
		},
	}

	return proto.Marshal(pl.IntoProto())
}

// get the serialised form and hash of the given key.
func (a *Accounting) serialise(k string) ([]byte, error) {
	if k != accountsKey {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	if !a.HasChanged(k) {
		return a.accState.serialised, nil
	}

	data, err := a.serialiseStakingAccounts()
	if err != nil {
		return nil, err
	}

	a.accState.serialised = data
	a.accState.changed = false
	return data, nil
}

func (a *Accounting) OnStateLoaded(_ context.Context) error {
	a.accState.isRestoring = false
	return nil
}

func (a *Accounting) OnStateLoadStarts(_ context.Context) error {
	a.accState.isRestoring = true
	return nil
}

func (a *Accounting) Namespace() types.SnapshotNamespace {
	return types.StakingSnapshot
}

func (a *Accounting) Keys() []string {
	return []string{accountsKey}
}

func (a *Accounting) Stopped() bool {
	return false
}

func (a *Accounting) HasChanged(k string) bool {
	return true
	// return a.accState.changed
}

func (a *Accounting) GetState(k string) ([]byte, []types.StateProvider, error) {
	data, err := a.serialise(k)
	return data, nil, err
}

func (a *Accounting) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if a.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadStakingAccounts:
		return nil, a.restoreStakingAccounts(ctx, pl.StakingAccounts, payload)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (a *Accounting) restoreStakingAccounts(ctx context.Context, accounts *types.StakingAccounts, p *types.Payload) error {
	a.hashableAccounts = make([]*StakingAccount, 0, len(accounts.Accounts))
	a.log.Debug("restoring staking accounts",
		logging.Int("n", len(accounts.Accounts)),
	)
	evts := []events.Event{}
	pevts := []events.Event{}
	for _, acc := range accounts.Accounts {
		stakingAcc := &StakingAccount{
			Party:   acc.Party,
			Balance: acc.Balance,
			Events:  acc.Events,
		}
		a.hashableAccounts = append(a.hashableAccounts, stakingAcc)
		a.accounts[acc.Party] = stakingAcc
		pevts = append(pevts, events.NewPartyEvent(ctx, types.Party{Id: acc.Party}))
		for _, e := range acc.Events {
			evts = append(evts, events.NewStakeLinking(ctx, *e))
		}
	}

	a.stakingAssetTotalSupply = accounts.StakingAssetTotalSupply.Clone()
	var err error
	a.accState.changed = false
	a.accState.serialised, err = proto.Marshal(p.IntoProto())
	a.broker.SendBatch(evts)
	a.broker.SendBatch(pevts)
	return err
}
