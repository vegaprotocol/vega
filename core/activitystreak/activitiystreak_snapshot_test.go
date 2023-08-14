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

package activitystreak_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/activitystreak"
	"code.vegaprotocol.io/vega/core/activitystreak/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testSnapshotEngine struct {
	*activitystreak.SnapshotEngine

	ctrl         *gomock.Controller
	broker       *mocks.MockBroker
	marketsStats *mocks.MockMarketsStatsAggregator
}

func getTestSnapshotEngine(t *testing.T) *testSnapshotEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	marketsStats := mocks.NewMockMarketsStatsAggregator(ctrl)
	broker := mocks.NewMockBroker(ctrl)

	e := &testSnapshotEngine{
		SnapshotEngine: activitystreak.NewSnapshotEngine(
			logging.NewTestLogger(), marketsStats, broker,
		),
		ctrl:         ctrl,
		broker:       broker,
		marketsStats: marketsStats,
	}

	e.OnMinQuantumOpenNationalVolumeUpdate(context.Background(), num.NewUint(100))
	e.OnMinQuantumTradeVolumeUpdate(context.Background(), num.NewUint(200))
	assert.NoError(t, e.OnBenefitTiersUpdate(context.Background(), &vegapb.ActivityStreakBenefitTiers{
		Tiers: []*vegapb.ActivityStreakBenefitTier{
			{
				MinimumActivityStreak: 1,
				RewardMultiplier:      "2",
				VestingMultiplier:     "1.5",
			},
			{
				MinimumActivityStreak: 7,
				RewardMultiplier:      "3",
				VestingMultiplier:     "2.5",
			},
			{
				MinimumActivityStreak: 14,
				RewardMultiplier:      "4",
				VestingMultiplier:     "3.5",
			},
		},
	}))

	return e
}

func TestSnapshot(t *testing.T) {
	e1 := getTestSnapshotEngine(t)

	t.Run("setting up engine 1", func(t *testing.T) {
		e1.marketsStats.EXPECT().GetMarketStats().Times(2).Return(
			map[string]*types.MarketStats{
				"market1": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(100),
						"party2": num.NewUint(100),
						"party3": num.NewUint(0),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
						"party2": num.NewUint(50),
						"party3": num.NewUint(150),
					},
				},
				"market2": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
						"party2": num.NewUint(20),
						"party3": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
						"party2": num.NewUint(50),
						"party3": num.NewUint(100),
					},
				},
			},
		)

		e1.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
		e1.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(
			func(evts []events.Event) {
				assert.Len(t, evts, 3)

				pas := evts[0].(*events.PartyActivityStreak)
				assert.Equal(t, pas.Proto().Party, "party1")
				assert.True(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 2)
				assert.Equal(t, int(pas.Proto().InactiveFor), 0)
				assert.Equal(t, int(pas.Proto().Epoch), 2)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "2")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1.5")
				pas = evts[1].(*events.PartyActivityStreak)
				assert.Equal(t, pas.Proto().Party, "party2")
				assert.True(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 2)
				assert.Equal(t, int(pas.Proto().InactiveFor), 0)
				assert.Equal(t, int(pas.Proto().Epoch), 2)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "2")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1.5")
				pas = evts[2].(*events.PartyActivityStreak)
				assert.Equal(t, pas.Proto().Party, "party3")
				assert.True(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 2)
				assert.Equal(t, int(pas.Proto().InactiveFor), 0)
				assert.Equal(t, int(pas.Proto().Epoch), 2)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "2")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1.5")
			},
		)

		e1.OnEpochEvent(context.Background(), types.Epoch{
			Seq:    1,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
		e1.OnEpochEvent(context.Background(), types.Epoch{
			Seq:    2,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	state1, _, err := e1.GetState(activitystreak.ActivityStreakKey)
	assert.NoError(t, err)
	assert.NotNil(t, state1)

	ppayload := &snapshotpb.Payload{}
	err = proto.Unmarshal(state1, ppayload)
	assert.NoError(t, err)

	e2 := getTestSnapshotEngine(t)
	_, err = e2.LoadState(context.Background(), types.PayloadFromProto(ppayload))
	assert.NoError(t, err)

	// now assert the v2 produce the same state
	state2, _, err := e2.GetState(activitystreak.ActivityStreakKey)
	assert.NoError(t, err)
	assert.NotNil(t, state2)

	assert.Equal(t, state1, state2)

	epochForward(t, e1, "engine 1")
	epochForward(t, e2, "engine 2")

	t.Run("finally comparing final state from both engines", func(t *testing.T) {
		state1, _, err := e1.GetState(activitystreak.ActivityStreakKey)
		assert.NoError(t, err)
		assert.NotNil(t, state1)

		ppayload := &snapshotpb.Payload{}
		err = proto.Unmarshal(state1, ppayload)
		assert.NoError(t, err)

		_, err = e2.LoadState(context.Background(), types.PayloadFromProto(ppayload))
		assert.NoError(t, err)

		// now assert the v2 produce the same state
		state2, _, err := e2.GetState(activitystreak.ActivityStreakKey)
		assert.NoError(t, err)
		assert.NotNil(t, state2)

		assert.Equal(t, state1, state2)
	})
}

func epochForward(t *testing.T, e *testSnapshotEngine, name string) {
	t.Helper()
	t.Run(fmt.Sprintf("moving time for %v", name), func(t *testing.T) {
		e.marketsStats.EXPECT().GetMarketStats().Times(1).Return(
			map[string]*types.MarketStats{},
		)

		e.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(
			func(evts []events.Event) {
				assert.Len(t, evts, 3)

				pas := evts[0].(*events.PartyActivityStreak)
				assert.Equal(t, pas.Proto().Party, "party1")
				assert.False(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 2)
				assert.Equal(t, int(pas.Proto().InactiveFor), 1)
				assert.Equal(t, int(pas.Proto().Epoch), 3)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "2")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1.5")
				pas = evts[1].(*events.PartyActivityStreak)
				assert.Equal(t, pas.Proto().Party, "party2")
				assert.False(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 2)
				assert.Equal(t, int(pas.Proto().InactiveFor), 1)
				assert.Equal(t, int(pas.Proto().Epoch), 3)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "2")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1.5")
				pas = evts[2].(*events.PartyActivityStreak)
				assert.Equal(t, pas.Proto().Party, "party3")
				assert.False(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 2)
				assert.Equal(t, int(pas.Proto().InactiveFor), 1)
				assert.Equal(t, int(pas.Proto().Epoch), 3)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "2")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1.5")
			},
		)

		e.OnEpochEvent(context.Background(), types.Epoch{
			Seq:    3,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})
}
