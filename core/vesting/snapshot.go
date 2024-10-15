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

package vesting

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

var VestingKey = (&types.PayloadVesting{}).Key()

type SnapshotEngine struct {
	*Engine
}

func NewSnapshotEngine(
	log *logging.Logger,
	c Collateral,
	asvm ActivityStreakVestingMultiplier,
	broker Broker,
	assets Assets,
	parties Parties,
	t Time,
) *SnapshotEngine {
	se := &SnapshotEngine{
		Engine: New(log, c, asvm, broker, assets, parties, t),
	}

	return se
}

func (e *SnapshotEngine) Namespace() types.SnapshotNamespace {
	return types.VestingSnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return []string{VestingKey}
}

func (e *SnapshotEngine) Stopped() bool {
	return false
}

func (e *SnapshotEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise(k)
	return state, nil, err
}

func (e *SnapshotEngine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadVesting:
		e.loadStateFromSnapshot(ctx, data.Vesting)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) loadStateFromSnapshot(ctx context.Context, state *snapshotpb.Vesting) {
	for _, entry := range state.PartiesReward {
		for _, v := range entry.InVesting {
			balance, underflow := num.UintFromString(v.Balance, 10)
			if underflow {
				e.log.Panic("uint256 in snapshot underflow",
					logging.String("value", v.Balance))
			}
			e.increaseVestingBalance(entry.Party, v.Asset, balance)
		}

		for _, locked := range entry.AssetLocked {
			for _, epochBalance := range locked.EpochBalances {
				balance, underflow := num.UintFromString(epochBalance.Balance, 10)
				if underflow {
					e.log.Panic("uint256 in snapshot underflow",
						logging.String("value", epochBalance.Balance))
				}
				e.increaseLockedForAsset(entry.Party, locked.Asset, balance, epochBalance.Epoch)
			}
		}
	}

	if vgcontext.InProgressUpgradeFrom(ctx, "v0.78.8") {
		e.updateStakingOnUpgrade79(ctx)
	}
}

// updateStakingOnUpgrade79 update staking balance for party which had
// rewards vesting before the upgrade.
func (e *SnapshotEngine) updateStakingOnUpgrade79(ctx context.Context) {
	for i, v := range e.c.GetAllVestingAndVestedAccountForAsset(e.stakingAsset) {
		e.updateStakingAccount(ctx, v.Owner, v.Balance.Clone(), uint64(i), e.broker.Stage)
	}
}

func (e *SnapshotEngine) serialise(k string) ([]byte, error) {
	switch k {
	case VestingKey:
		return e.serialiseAll()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshotEngine) serialiseAll() ([]byte, error) {
	out := snapshotpb.Vesting{}

	for party, rewardState := range e.state {
		partyReward := &snapshotpb.PartyReward{
			Party: party,
		}

		for asset, balance := range rewardState.Vesting {
			partyReward.InVesting = append(partyReward.InVesting, &snapshotpb.InVesting{
				Asset:   asset,
				Balance: balance.String(),
			})
		}

		sort.Slice(partyReward.InVesting, func(i, j int) bool {
			return partyReward.InVesting[i].Asset < partyReward.InVesting[j].Asset
		})

		for asset, epochBalances := range rewardState.Locked {
			assetLocked := &snapshotpb.AssetLocked{
				Asset: asset,
			}

			for epoch, balance := range epochBalances {
				assetLocked.EpochBalances = append(assetLocked.EpochBalances, &snapshotpb.EpochBalance{
					Epoch:   epoch,
					Balance: balance.String(),
				})
			}

			sort.Slice(assetLocked.EpochBalances, func(i, j int) bool {
				return assetLocked.EpochBalances[i].Epoch < assetLocked.EpochBalances[j].Epoch
			})

			partyReward.AssetLocked = append(partyReward.AssetLocked, assetLocked)
		}

		sort.Slice(partyReward.AssetLocked, func(i, j int) bool {
			return partyReward.AssetLocked[i].Asset < partyReward.AssetLocked[j].Asset
		})

		out.PartiesReward = append(out.PartiesReward, partyReward)
	}

	sort.Slice(out.PartiesReward, func(i, j int) bool { return out.PartiesReward[i].Party < out.PartiesReward[j].Party })

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_Vesting{
			Vesting: &out,
		},
	}

	serialized, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize team switches payload: %w", err)
	}

	return serialized, nil
}
