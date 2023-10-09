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

package activitystreak

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

var ActivityStreakKey = (&types.PayloadActivityStreak{}).Key()

type SnapshotEngine struct {
	*Engine
}

func NewSnapshotEngine(
	log *logging.Logger,
	marketStats MarketsStatsAggregator,
	broker Broker,
) *SnapshotEngine {
	se := &SnapshotEngine{
		Engine: New(log, marketStats, broker),
	}

	return se
}

func (e *SnapshotEngine) Namespace() types.SnapshotNamespace {
	return types.ActivityStreakSnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return []string{ActivityStreakKey}
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
	case *types.PayloadActivityStreak:
		e.loadStateFromSnapshot(data.ActivityStreak)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) loadStateFromSnapshot(state *snapshotpb.ActivityStreak) {
	for _, v := range state.PartiesActivityStreak {
		var rewardDistributionActivityMultiplier num.Decimal
		if len(v.RewardDistributionMultiplier) > 0 {
			rewardDistributionActivityMultiplier, _ = num.UnmarshalBinaryDecimal(v.RewardDistributionMultiplier)
		}

		var rewardVestingActivityMultiplier num.Decimal
		if len(v.RewardVestingMultiplier) > 0 {
			rewardVestingActivityMultiplier, _ = num.UnmarshalBinaryDecimal(v.RewardVestingMultiplier)
		}

		e.partiesActivity[v.Party] = &PartyActivity{
			Active:                               v.Active,
			Inactive:                             v.Inactive,
			RewardDistributionActivityMultiplier: rewardDistributionActivityMultiplier,
			RewardVestingActivityMultiplier:      rewardVestingActivityMultiplier,
		}
	}
}

func (e *SnapshotEngine) serialise(k string) ([]byte, error) {
	switch k {
	case ActivityStreakKey:
		return e.serialiseAll()
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (e *SnapshotEngine) serialiseAll() ([]byte, error) {
	out := snapshotpb.ActivityStreak{
		PartiesActivityStreak: make([]*snapshotpb.PartyActivityStreak, 0, len(e.partiesActivity)),
	}

	for party, activity := range e.partiesActivity {
		rewardDistributionMultiplier, _ := activity.RewardDistributionActivityMultiplier.MarshalBinary()
		rewardVestingMultiplier, _ := activity.RewardVestingActivityMultiplier.MarshalBinary()
		out.PartiesActivityStreak = append(out.PartiesActivityStreak, &snapshotpb.PartyActivityStreak{
			Party:                        party,
			Active:                       activity.Active,
			Inactive:                     activity.Inactive,
			RewardDistributionMultiplier: rewardDistributionMultiplier,
			RewardVestingMultiplier:      rewardVestingMultiplier,
		})
	}

	sort.Slice(out.PartiesActivityStreak, func(i, j int) bool {
		return out.PartiesActivityStreak[i].Party < out.PartiesActivityStreak[j].Party
	})

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_ActivityStreak{
			ActivityStreak: &out,
		},
	}

	serialized, err := proto.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("could not serialize team switches payload: %w", err)
	}

	return serialized, nil
}
