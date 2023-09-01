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
		e.partiesActivity[v.Party] = &PartyActivity{
			Active:                               v.Active,
			Inactive:                             v.Inactive,
			RewardDistributionActivityMultiplier: num.MustDecimalFromString(v.RewardDistributionMultiplier),
			RewardVestingActivityMultiplier:      num.MustDecimalFromString(v.RewardVestingMultiplier),
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
		out.PartiesActivityStreak = append(out.PartiesActivityStreak, &snapshotpb.PartyActivityStreak{
			Party:                        party,
			Active:                       activity.Active,
			Inactive:                     activity.Inactive,
			RewardDistributionMultiplier: activity.RewardDistributionActivityMultiplier.String(),
			RewardVestingMultiplier:      activity.RewardVestingActivityMultiplier.String(),
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
