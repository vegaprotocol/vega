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
	"testing"

	"code.vegaprotocol.io/vega/core/activitystreak"
	"code.vegaprotocol.io/vega/core/activitystreak/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testEngine struct {
	*activitystreak.Engine

	ctrl         *gomock.Controller
	broker       *mocks.MockBroker
	marketsStats *mocks.MockMarketsStatsAggregator
}

func getTestEngine(t *testing.T) *testEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	marketsStats := mocks.NewMockMarketsStatsAggregator(ctrl)
	broker := mocks.NewMockBroker(ctrl)

	return &testEngine{
		Engine: activitystreak.New(
			logging.NewTestLogger(), marketsStats, broker,
		),
		ctrl:         ctrl,
		broker:       broker,
		marketsStats: marketsStats,
	}
}

func TestStreak(t *testing.T) {
	engine := getTestEngine(t)

	engine.OnMinQuantumOpenNationalVolumeUpdate(context.Background(), num.NewUint(100))
	engine.OnMinQuantumTradeVolumeUpdate(context.Background(), num.NewUint(200))
	assert.NoError(t, engine.OnBenefitTiersUpdate(context.Background(), &vegapb.ActivityStreakBenefitTiers{
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

	t.Run("no streak for a party == 1x", func(t *testing.T) {
		tradeX, volumeX := engine.GetRewardsDistributionMultiplier("party1"), engine.GetRewardsVestingMultiplier("party1")

		assert.Equal(t, num.DecimalOne(), tradeX)
		assert.Equal(t, num.DecimalOne(), volumeX)
	})

	t.Run("add volume < min == 1x", func(t *testing.T) {
		engine.marketsStats.EXPECT().GetMarketStats().Times(1).Return(
			map[string]*types.MarketStats{
				"market1": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
				"market2": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
			},
		)

		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(
			func(evts []events.Event) {
				assert.Len(t, evts, 1)

				pas := evts[0].(*events.PartyActivityStreak)
				assert.False(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 0)
				assert.Equal(t, int(pas.Proto().InactiveFor), 1)
				assert.Equal(t, int(pas.Proto().Epoch), 1)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "1")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1")
			},
		)

		engine.OnEpochEvent(context.Background(), types.Epoch{
			Seq:    1,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})

		tradeX, volumeX := engine.GetRewardsDistributionMultiplier("party1"), engine.GetRewardsVestingMultiplier("party1")

		assert.Equal(t, num.DecimalOne(), tradeX)
		assert.Equal(t, num.DecimalOne(), volumeX)
	})

	t.Run("add volume > min == increase multipliers", func(t *testing.T) {
		engine.marketsStats.EXPECT().GetMarketStats().Times(1).Return(
			map[string]*types.MarketStats{
				"market1": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(100),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
				"market2": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
			},
		)

		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(
			func(evts []events.Event) {
				assert.Len(t, evts, 1)

				pas := evts[0].(*events.PartyActivityStreak)
				assert.True(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 1)
				assert.Equal(t, int(pas.Proto().InactiveFor), 0)
				assert.Equal(t, int(pas.Proto().Epoch), 2)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "2")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1.5")
			},
		)

		engine.OnEpochEvent(context.Background(), types.Epoch{
			Seq:    2,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})

		tradeX, volumeX := engine.GetRewardsDistributionMultiplier("party1"), engine.GetRewardsVestingMultiplier("party1")

		assert.Equal(t, num.MustDecimalFromString("2"), tradeX)
		assert.Equal(t, num.MustDecimalFromString("1.5"), volumeX)
	})

	t.Run("add volume > min many time == move to next tier", func(t *testing.T) {
		engine.marketsStats.EXPECT().GetMarketStats().Times(6).Return(
			map[string]*types.MarketStats{
				"market1": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(100),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
				"market2": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
			},
		)

		// discard first 5
		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(5)

		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(
			func(evts []events.Event) {
				assert.Len(t, evts, 1)

				pas := evts[0].(*events.PartyActivityStreak)
				assert.True(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 7)
				assert.Equal(t, int(pas.Proto().InactiveFor), 0)
				assert.Equal(t, int(pas.Proto().Epoch), 8)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "3")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "2.5")
			},
		)

		for i := 3; i <= 8; i++ {
			engine.OnEpochEvent(context.Background(), types.Epoch{
				Seq:    uint64(i),
				Action: vegapb.EpochAction_EPOCH_ACTION_END,
			})
		}

		tradeX, volumeX := engine.GetRewardsDistributionMultiplier("party1"), engine.GetRewardsVestingMultiplier("party1")

		assert.Equal(t, num.MustDecimalFromString("3"), tradeX)
		assert.Equal(t, num.MustDecimalFromString("2.5"), volumeX)
	})

	t.Run("add volume < min less times than current streak == inactive but still have benefits", func(t *testing.T) {
		engine.marketsStats.EXPECT().GetMarketStats().Times(4).Return(
			map[string]*types.MarketStats{
				"market1": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
				"market2": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
			},
		)

		// discard first 5
		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(3)

		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(
			func(evts []events.Event) {
				assert.Len(t, evts, 1)

				pas := evts[0].(*events.PartyActivityStreak)
				assert.False(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 7)
				assert.Equal(t, int(pas.Proto().InactiveFor), 4)
				assert.Equal(t, int(pas.Proto().Epoch), 12)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "3")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "2.5")
			},
		)

		for i := 9; i <= 12; i++ {
			engine.OnEpochEvent(context.Background(), types.Epoch{
				Seq:    uint64(i),
				Action: vegapb.EpochAction_EPOCH_ACTION_END,
			})
		}

		tradeX, volumeX := engine.GetRewardsDistributionMultiplier("party1"), engine.GetRewardsVestingMultiplier("party1")

		assert.Equal(t, num.MustDecimalFromString("3"), tradeX)
		assert.Equal(t, num.MustDecimalFromString("2.5"), volumeX)
	})

	t.Run("add volume > min again == becomes active again", func(t *testing.T) {
		engine.marketsStats.EXPECT().GetMarketStats().Times(1).Return(
			map[string]*types.MarketStats{
				"market1": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(100),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
				"market2": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
			},
		)

		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(
			func(evts []events.Event) {
				assert.Len(t, evts, 1)

				pas := evts[0].(*events.PartyActivityStreak)
				assert.True(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 8)
				assert.Equal(t, int(pas.Proto().InactiveFor), 0)
				assert.Equal(t, int(pas.Proto().Epoch), 13)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "3")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "2.5")
			},
		)

		engine.OnEpochEvent(context.Background(), types.Epoch{
			Seq:    uint64(13),
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})

		tradeX, volumeX := engine.GetRewardsDistributionMultiplier("party1"), engine.GetRewardsVestingMultiplier("party1")

		assert.Equal(t, num.MustDecimalFromString("3"), tradeX)
		assert.Equal(t, num.MustDecimalFromString("2.5"), volumeX)
	})

	t.Run("add volume < min more times than current streak looses benefits", func(t *testing.T) {
		engine.marketsStats.EXPECT().GetMarketStats().Times(11).Return(
			map[string]*types.MarketStats{
				"market1": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
				"market2": {
					PartiesOpenNotionalVolume: map[string]*num.Uint{
						"party1": num.NewUint(20),
					},
					PartiesTotalTradeVolume: map[string]*num.Uint{
						"party1": num.NewUint(50),
					},
				},
			},
		)

		// discard first 5
		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(10)

		engine.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(
			func(evts []events.Event) {
				assert.Len(t, evts, 1)

				pas := evts[0].(*events.PartyActivityStreak)
				assert.False(t, pas.Proto().IsActive)
				assert.Equal(t, int(pas.Proto().ActiveFor), 0)
				assert.Equal(t, int(pas.Proto().InactiveFor), 11)
				assert.Equal(t, int(pas.Proto().Epoch), 24)
				assert.Equal(t, pas.Proto().RewardDistributionActivityMultiplier, "1")
				assert.Equal(t, pas.Proto().RewardVestingActivityMultiplier, "1")
			},
		)

		for i := 14; i <= 24; i++ {
			engine.OnEpochEvent(context.Background(), types.Epoch{
				Seq:    uint64(i),
				Action: vegapb.EpochAction_EPOCH_ACTION_END,
			})
		}

		tradeX, volumeX := engine.GetRewardsDistributionMultiplier("party1"), engine.GetRewardsVestingMultiplier("party1")

		assert.Equal(t, num.MustDecimalFromString("1"), tradeX)
		assert.Equal(t, num.MustDecimalFromString("1"), volumeX)
	})
}
