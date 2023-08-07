// Copyright (c) 2023 Gobalsky Labs Limited
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

package vesting

import (
	"context"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
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
) *SnapshotEngine {
	se := &SnapshotEngine{
		Engine: New(log, c, asvm, broker, assets),
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

func (e *SnapshotEngine) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadVesting:
		e.loadStateFromSnapshot(data.Vesting)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) loadStateFromSnapshot(state *snapshotpb.Vesting) {
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
				e.increaseLockedForAsset(
					entry.Party, locked.Asset, balance, epochBalance.Epoch)
			}
		}
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
