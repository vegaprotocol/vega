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
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
)

var accountsKey = (&types.PayloadStakingAccounts{}).Key()

type accountingSnapshotState struct {
	serialised  []byte
	isRestoring bool
}

func (a *Accounting) serialiseStakingAccounts() ([]byte, error) {
	accounts := make([]*types.StakingAccount, 0, len(a.hashableAccounts))
	a.log.Debug("serialsing staking accounts", logging.Int("n", len(a.hashableAccounts)))
	for _, acc := range a.hashableAccounts {
		// dedup transactions with the same eth hash from different block heights and recalc the balance
		acc.Events = a.dedupHack(acc.Events)
		acc.computeOngoingBalance()
		accounts = append(accounts,
			&types.StakingAccount{
				Party:   acc.Party,
				Balance: acc.Balance,
				Events:  acc.Events,
			})
	}

	var psts *types.StakeTotalSupply
	if a.pendingStakeTotalSupply != nil {
		psts = a.pendingStakeTotalSupply.sts
	}

	pl := types.Payload{
		Data: &types.PayloadStakingAccounts{
			PendingStakeTotalSupply: psts,
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

	data, err := a.serialiseStakingAccounts()
	if err != nil {
		return nil, err
	}

	a.accState.serialised = data
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

		return nil, a.restoreStakingAccounts(ctx, pl.StakingAccounts, pl.PendingStakeTotalSupply, payload)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

// dedupHack takes care of events with the same ethereum tx hash originating from
// reorg - the result is that duplicates are removed and the branch with the latest block height is kept
// after calling this function, the balance should be recalculated.
func (a *Accounting) dedupHack(evts []*types.StakeLinking) []*types.StakeLinking {
	hashToEvt := map[string]*types.StakeLinking{}
	for _, sl := range evts {
		evt, ok := hashToEvt[sl.TxHash]
		if !ok {
			hashToEvt[sl.TxHash] = sl
		} else {
			if sl.BlockHeight > evt.BlockHeight {
				a.log.Warn("duplicate events with identical transaction hash found", logging.String("tx-hash", sl.TxHash), logging.Uint64("block-height1", sl.BlockHeight), logging.Uint64("block-height2", evt.BlockHeight))
				hashToEvt[sl.TxHash] = sl
			}
		}
	}
	newEvts := make([]*types.StakeLinking, 0, len(hashToEvt))
	for _, sl := range evts {
		evt := hashToEvt[sl.TxHash]
		if evt.BlockHeight == sl.BlockHeight {
			newEvts = append(newEvts, sl)
		}
	}
	return newEvts
}

func (a *Accounting) restoreStakingAccounts(ctx context.Context, accounts *types.StakingAccounts, pendingSupply *types.StakeTotalSupply, p *types.Payload) error {
	a.hashableAccounts = make([]*Account, 0, len(accounts.Accounts))
	a.log.Debug("restoring staking accounts",
		logging.Int("n", len(accounts.Accounts)),
	)
	evts := []events.Event{}
	pevts := []events.Event{}
	for _, acc := range accounts.Accounts {
		stakingAcc := &Account{
			Party:   acc.Party,
			Balance: acc.Balance,
			Events:  a.dedupHack(acc.Events),
		}
		stakingAcc.computeOngoingBalance()
		a.hashableAccounts = append(a.hashableAccounts, stakingAcc)
		a.accounts[acc.Party] = stakingAcc
		pevts = append(pevts, events.NewPartyEvent(ctx, types.Party{Id: acc.Party}))
		for _, e := range acc.Events {
			evts = append(evts, events.NewStakeLinking(ctx, *e))
		}
	}

	if pendingSupply != nil {
		expectedSupply := pendingSupply.TotalSupply.Clone()
		a.pendingStakeTotalSupply = &pendingStakeTotalSupply{
			sts: pendingSupply,
			check: func() error {
				totalSupply, err := a.getStakeAssetTotalSupply(a.stakingBridgeAddress)
				if err != nil {
					return err
				}

				if totalSupply.NEQ(expectedSupply) {
					return fmt.Errorf(
						"invalid stake asset total supply, expected %s got %s",
						expectedSupply.String(), totalSupply.String(),
					)
				}

				return nil
			},
		}
		a.witness.RestoreResource(a.pendingStakeTotalSupply, a.onStakeTotalSupplyVerified)
	}

	a.stakingAssetTotalSupply = accounts.StakingAssetTotalSupply.Clone()
	var err error
	a.accState.serialised, err = proto.Marshal(p.IntoProto())
	a.broker.SendBatch(evts)
	a.broker.SendBatch(pevts)
	return err
}
