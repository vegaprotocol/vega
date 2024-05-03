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

package vesting_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributeAfterDelay(t *testing.T) {
	v := getTestEngine(t)

	ctx := context.Background()

	// distribute 90% as the base rate,
	// so first we distribute some, then we get under the minimum value, and all the rest
	// is distributed
	require.NoError(t, v.OnRewardVestingBaseRateUpdate(ctx, num.MustDecimalFromString("0.9")))
	// this is multiplied by the quantum, so it will make it 100% of the quantum
	require.NoError(t, v.OnRewardVestingMinimumTransferUpdate(ctx, num.MustDecimalFromString("1")))

	require.NoError(t, v.OnBenefitTiersUpdate(ctx, &vegapb.VestingBenefitTiers{
		Tiers: []*vegapb.VestingBenefitTier{
			{
				MinimumQuantumBalance: "200",
				RewardMultiplier:      "1",
			},
			{
				MinimumQuantumBalance: "350",
				RewardMultiplier:      "2",
			},
			{
				MinimumQuantumBalance: "500",
				RewardMultiplier:      "3",
			},
		},
	}))

	v.asvm.EXPECT().GetRewardsVestingMultiplier(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1"))
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)

	party := "party1"
	vegaAsset := "VEGA"

	v.col.InitVestedBalance(party, vegaAsset, num.NewUint(300))

	epochSeq := uint64(1)

	t.Run("No vesting stats and summary when no reward is being vested", func(t *testing.T) {
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats:   []*eventspb.PartyVestingStats{},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
			Seq:    epochSeq,
		})
	})

	t.Run("Add a reward locked for 3 epochs", func(t *testing.T) {
		v.AddReward(party, vegaAsset, num.NewUint(100), 3)
	})

	t.Run("Wait for 3 epochs", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			epochSeq += 1

			v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
				e, ok := evt.(*events.VestingStatsUpdated)
				require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
				assert.Equal(t, eventspb.VestingStatsUpdated{
					AtEpoch: epochSeq,
					Stats: []*eventspb.PartyVestingStats{
						{
							PartyId:               party,
							RewardBonusMultiplier: "1",
							QuantumBalance:        "300",
						},
					},
				}, e.Proto())
			}).Times(1)

			v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
				e, ok := evt.(*events.VestingBalancesSummary)
				require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
				assert.Equal(t, eventspb.VestingBalancesSummary{
					EpochSeq: epochSeq,
					PartiesVestingSummary: []*eventspb.PartyVestingSummary{
						{
							Party: party,
							PartyLockedBalances: []*eventspb.PartyLockedBalance{
								{
									Asset:      vegaAsset,
									UntilEpoch: 5,
									Balance:    "100",
								},
							},
							PartyVestingBalances: []*eventspb.PartyVestingBalance{},
						},
					},
				}, e.Proto())
			}).Times(1)

			v.OnEpochEvent(ctx, types.Epoch{
				Action: vegapb.EpochAction_EPOCH_ACTION_END,
				Seq:    epochSeq,
			})
		}
	})

	t.Run("First reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "1",
						QuantumBalance:        "300",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq: epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{
					{
						Party:               party,
						PartyLockedBalances: []*eventspb.PartyLockedBalance{},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "10",
							},
						},
					},
				},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("Second reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "2",
						QuantumBalance:        "390",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("No vesting stats and summary when no reward is being vested anymore", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats:   []*eventspb.PartyVestingStats{},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})
}

func TestDistributeWithNoDelay(t *testing.T) {
	v := getTestEngine(t)

	ctx := context.Background()

	// distribute 90% as the base rate,
	// so first we distribute some, then we get under the minimum value, and all the rest
	// is distributed
	require.NoError(t, v.OnRewardVestingBaseRateUpdate(ctx, num.MustDecimalFromString("0.9")))
	// this is multiplied by the quantum, so it will make it 100% of the quantum
	require.NoError(t, v.OnRewardVestingMinimumTransferUpdate(ctx, num.MustDecimalFromString("1")))

	require.NoError(t, v.OnBenefitTiersUpdate(ctx, &vegapb.VestingBenefitTiers{
		Tiers: []*vegapb.VestingBenefitTier{
			{
				MinimumQuantumBalance: "200",
				RewardMultiplier:      "1",
			},
			{
				MinimumQuantumBalance: "350",
				RewardMultiplier:      "2",
			},
			{
				MinimumQuantumBalance: "500",
				RewardMultiplier:      "3",
			},
		},
	}))

	// set the asvm to return always 1
	v.asvm.EXPECT().GetRewardsVestingMultiplier(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1"))

	// set asset to return proper quantum
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)

	party := "party1"
	vegaAsset := "VEGA"

	v.col.InitVestedBalance(party, vegaAsset, num.NewUint(300))

	epochSeq := uint64(1)

	t.Run("No vesting stats and summary when no reward is being vested", func(t *testing.T) {
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats:   []*eventspb.PartyVestingStats{},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
			Seq:    epochSeq,
		})
	})

	t.Run("Add a reward without epoch lock", func(t *testing.T) {
		v.AddReward(party, vegaAsset, num.NewUint(100), 0)
	})

	t.Run("First reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "1",
						QuantumBalance:        "300",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq: epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{
					{
						Party:               party,
						PartyLockedBalances: []*eventspb.PartyLockedBalance{},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "10",
							},
						},
					},
				},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("Second reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "2",
						QuantumBalance:        "390",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("No vesting stats and summary when no reward is being vested anymore", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats:   []*eventspb.PartyVestingStats{},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})
}

func TestDistributeWithStreakRate(t *testing.T) {
	v := getTestEngine(t)

	ctx := context.Background()

	// distribute 90% as the base rate,
	// so first we distribute some, then we get under the minimum value, and all the rest
	// is distributed
	require.NoError(t, v.OnRewardVestingBaseRateUpdate(ctx, num.MustDecimalFromString("0.9")))
	// this is multiplied by the quantum, so it will make it 100% of the quantum
	require.NoError(t, v.OnRewardVestingMinimumTransferUpdate(ctx, num.MustDecimalFromString("1")))

	require.NoError(t, v.OnBenefitTiersUpdate(ctx, &vegapb.VestingBenefitTiers{
		Tiers: []*vegapb.VestingBenefitTier{
			{
				MinimumQuantumBalance: "200",
				RewardMultiplier:      "1",
			},
			{
				MinimumQuantumBalance: "350",
				RewardMultiplier:      "2",
			},
			{
				MinimumQuantumBalance: "500",
				RewardMultiplier:      "3",
			},
		},
	}))

	v.asvm.EXPECT().GetRewardsVestingMultiplier(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1.1"))
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)

	party := "party1"
	vegaAsset := "VEGA"

	v.col.InitVestedBalance(party, vegaAsset, num.NewUint(300))

	epochSeq := uint64(1)

	t.Run("No vesting stats and summary when no reward is being vested", func(t *testing.T) {
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats:   []*eventspb.PartyVestingStats{},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
			Seq:    epochSeq,
		})
	})

	t.Run("Add a reward without epoch lock", func(t *testing.T) {
		v.AddReward(party, vegaAsset, num.NewUint(100), 0)
	})

	t.Run("First reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "1",
						QuantumBalance:        "300",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq: epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{
					{
						Party:               party,
						PartyLockedBalances: []*eventspb.PartyLockedBalance{},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "1",
							},
						},
					},
				},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("Second reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "2",
						QuantumBalance:        "399",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("No vesting stats and summary when no reward is being vested anymore", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats:   []*eventspb.PartyVestingStats{},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})
}

func TestDistributeMultipleAfterDelay(t *testing.T) {
	v := getTestEngine(t)

	ctx := context.Background()

	// distribute 90% as the base rate,
	// so first we distribute some, then we get under the minimum value, and all the rest
	// is distributed
	require.NoError(t, v.OnRewardVestingBaseRateUpdate(ctx, num.MustDecimalFromString("0.9")))
	// this is multiplied by the quantum, so it will make it 100% of the quantum
	require.NoError(t, v.OnRewardVestingMinimumTransferUpdate(ctx, num.MustDecimalFromString("1")))

	require.NoError(t, v.OnBenefitTiersUpdate(ctx, &vegapb.VestingBenefitTiers{
		Tiers: []*vegapb.VestingBenefitTier{
			{
				MinimumQuantumBalance: "200",
				RewardMultiplier:      "1",
			},
			{
				MinimumQuantumBalance: "350",
				RewardMultiplier:      "2",
			},
			{
				MinimumQuantumBalance: "500",
				RewardMultiplier:      "3",
			},
		},
	}))

	v.asvm.EXPECT().GetRewardsVestingMultiplier(gomock.Any()).AnyTimes().Return(num.MustDecimalFromString("1"))
	v.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(dummyAsset{quantum: 10}), nil)

	party := "party1"
	vegaAsset := "VEGA"

	v.col.InitVestedBalance(party, vegaAsset, num.NewUint(300))

	epochSeq := uint64(1)

	t.Run("No vesting stats and summary when no reward is being vested", func(t *testing.T) {
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats:   []*eventspb.PartyVestingStats{},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
			Seq:    epochSeq,
		})
	})

	t.Run("Add a reward locked for 2 epochs", func(t *testing.T) {
		v.AddReward(party, vegaAsset, num.NewUint(100), 2)
	})

	t.Run("Add another reward locked for 1 epoch", func(t *testing.T) {
		v.AddReward(party, vegaAsset, num.NewUint(100), 1)
	})

	t.Run("Wait for 1 epoch", func(t *testing.T) {
		epochSeq += 1

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "1",
						QuantumBalance:        "300",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq: epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{
					{
						Party: party,
						PartyLockedBalances: []*eventspb.PartyLockedBalance{
							{
								Asset:      vegaAsset,
								UntilEpoch: 3,
								Balance:    "100",
							},
							{
								Asset:      vegaAsset,
								UntilEpoch: 4,
								Balance:    "100",
							},
						},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{},
					},
				},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
			Seq:    epochSeq,
		})
	})

	t.Run("First reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "1",
						QuantumBalance:        "300",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq: epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{
					{
						Party: party,
						PartyLockedBalances: []*eventspb.PartyLockedBalance{
							{
								Asset:      vegaAsset,
								UntilEpoch: 4,
								Balance:    "100",
							},
						},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "10",
							},
						},
					},
				},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("Second reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "2",
						QuantumBalance:        "390",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq: epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{
					{
						Party:               party,
						PartyLockedBalances: []*eventspb.PartyLockedBalance{},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "11",
							},
						},
					},
				},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("Third reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "2",
						QuantumBalance:        "489",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq: epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{
					{
						Party:               party,
						PartyLockedBalances: []*eventspb.PartyLockedBalance{},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "1",
							},
						},
					},
				},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("Fourth reward payment", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:               party,
						RewardBonusMultiplier: "2",
						QuantumBalance:        "499",
					},
				},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.LedgerMovements)
			require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
			// LedgerMovements is the result of a mock, so it doesn't really make sense to verify data
			// consistency.
			assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})

	t.Run("No vesting stats and summary when no reward is being vested anymore", func(t *testing.T) {
		epochSeq += 1
		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats:   []*eventspb.PartyVestingStats{},
			}, e.Proto())
		}).Times(1)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingBalancesSummary)
			require.True(t, ok, "Event should be a VestingBalancesSummary, but is %T", evt)
			assert.Equal(t, eventspb.VestingBalancesSummary{
				EpochSeq:              epochSeq,
				PartiesVestingSummary: []*eventspb.PartyVestingSummary{},
			}, e.Proto())
		}).Times(1)

		v.OnEpochEvent(ctx, types.Epoch{
			Seq:    epochSeq,
			Action: vegapb.EpochAction_EPOCH_ACTION_END,
		})
	})
}
