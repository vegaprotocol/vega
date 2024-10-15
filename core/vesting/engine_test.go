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

	v.parties.EXPECT().RelatedKeys(party).Return(nil, nil).AnyTimes()

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
		v.AddReward(context.Background(), party, vegaAsset, num.NewUint(100), 3)
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
							PartyId:                     party,
							RewardBonusMultiplier:       "1",
							QuantumBalance:              "300",
							SummedRewardBonusMultiplier: "1",
							SummedQuantumBalance:        "300",
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "390",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "390",
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "400",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "400",
					},
				},
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

	v.parties.EXPECT().RelatedKeys(party).Return(nil, nil).AnyTimes()

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
		v.AddReward(context.Background(), party, vegaAsset, num.NewUint(100), 0)
	})

	t.Run("First reward payment", func(t *testing.T) {
		epochSeq += 1

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "390",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "390",
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "400",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "400",
					},
				},
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

	v.parties.EXPECT().RelatedKeys(party).Return(nil, nil).AnyTimes()

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
		v.AddReward(context.Background(), party, vegaAsset, num.NewUint(100), 0)
	})

	t.Run("First reward payment", func(t *testing.T) {
		epochSeq += 1

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "399",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "399",
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "400",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "400",
					},
				},
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

	v.parties.EXPECT().RelatedKeys(party).Return(nil, nil).AnyTimes()

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
		v.AddReward(context.Background(), party, vegaAsset, num.NewUint(100), 2)
	})

	t.Run("Add another reward locked for 1 epoch", func(t *testing.T) {
		v.AddReward(context.Background(), party, vegaAsset, num.NewUint(100), 1)
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
						PartyId:                     party,
						RewardBonusMultiplier:       "1",
						QuantumBalance:              "300",
						SummedRewardBonusMultiplier: "1",
						SummedQuantumBalance:        "300",
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "390",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "390",
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "489",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "489",
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "499",
						SummedRewardBonusMultiplier: "2",
						SummedQuantumBalance:        "499",
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "3",
						QuantumBalance:              "500",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "500",
					},
				},
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

func TestDistributeWithRelatedKeys(t *testing.T) {
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
	partyID := types.PartyID(party)
	vegaAsset := "VEGA"
	derivedKeys := []string{"derived1", "derived2", "derived3"}

	v.parties.EXPECT().RelatedKeys(party).Return(&partyID, derivedKeys).AnyTimes()

	v.col.InitVestedBalance(party, vegaAsset, num.NewUint(300))

	for _, key := range derivedKeys {
		v.col.InitVestedBalance(key, vegaAsset, num.NewUint(100))
		v.parties.EXPECT().RelatedKeys(key).Return(&partyID, derivedKeys).AnyTimes()
	}

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
		v.AddReward(context.Background(), party, vegaAsset, num.NewUint(100), 0)

		for _, key := range derivedKeys {
			v.AddReward(context.Background(), key, vegaAsset, num.NewUint(50), 0)
		}
	})

	t.Run("First reward payment", func(t *testing.T) {
		epochSeq += 1

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)

			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     derivedKeys[0],
						RewardBonusMultiplier:       "1",
						QuantumBalance:              "145",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "825",
					},
					{
						PartyId:                     derivedKeys[1],
						RewardBonusMultiplier:       "1",
						QuantumBalance:              "145",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "825",
					},
					{
						PartyId:                     derivedKeys[2],
						RewardBonusMultiplier:       "1",
						QuantumBalance:              "145",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "825",
					},
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "390",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "825",
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
						Party:               derivedKeys[0],
						PartyLockedBalances: []*eventspb.PartyLockedBalance{},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "5",
							},
						},
					},
					{
						Party:               derivedKeys[1],
						PartyLockedBalances: []*eventspb.PartyLockedBalance{},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "5",
							},
						},
					},
					{
						Party:               derivedKeys[2],
						PartyLockedBalances: []*eventspb.PartyLockedBalance{},
						PartyVestingBalances: []*eventspb.PartyVestingBalance{
							{
								Asset:   vegaAsset,
								Balance: "5",
							},
						},
					},
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

		expectLedgerMovements(t, v)

		v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
			e, ok := evt.(*events.VestingStatsUpdated)
			require.True(t, ok, "Event should be a VestingStatsUpdated, but is %T", evt)
			assert.Equal(t, eventspb.VestingStatsUpdated{
				AtEpoch: epochSeq,
				Stats: []*eventspb.PartyVestingStats{
					{
						PartyId:                     derivedKeys[0],
						RewardBonusMultiplier:       "1",
						QuantumBalance:              "150",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "850",
					},
					{
						PartyId:                     derivedKeys[1],
						RewardBonusMultiplier:       "1",
						QuantumBalance:              "150",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "850",
					},
					{
						PartyId:                     derivedKeys[2],
						RewardBonusMultiplier:       "1",
						QuantumBalance:              "150",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "850",
					},
					{
						PartyId:                     party,
						RewardBonusMultiplier:       "2",
						QuantumBalance:              "400",
						SummedRewardBonusMultiplier: "3",
						SummedQuantumBalance:        "850",
					},
				},
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

func TestGetRewardBonusMultiplier(t *testing.T) {
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
				MinimumQuantumBalance: "500",
				RewardMultiplier:      "2",
			},
			{
				MinimumQuantumBalance: "1200",
				RewardMultiplier:      "3",
			},
		},
	}))

	party := "party1"
	partyID := types.PartyID(party)
	vegaAsset := "VEGA"
	derivedKeys := []string{"derived1", "derived2", "derived3", "derived4"}

	v.parties.EXPECT().RelatedKeys(party).Return(&partyID, derivedKeys).AnyTimes()

	v.col.InitVestedBalance(party, vegaAsset, num.NewUint(500))

	for _, key := range derivedKeys {
		v.col.InitVestedBalance(key, vegaAsset, num.NewUint(250))
		v.parties.EXPECT().RelatedKeys(key).Return(&partyID, derivedKeys).AnyTimes()
	}

	for _, key := range append(derivedKeys, party) {
		_, summed := v.GetSingleAndSummedRewardBonusMultipliers(key)
		require.Equal(t, num.DecimalFromInt64(1500), summed.QuantumBalance)
		require.Equal(t, num.DecimalFromInt64(3), summed.Multiplier)
	}

	// check that we only called the GetVestingQuantumBalance once for each key
	// later calls should be cached
	require.Equal(t, 5, v.col.GetVestingQuantumBalanceCallCount())

	for _, key := range append(derivedKeys, party) {
		_, summed := v.GetSingleAndSummedRewardBonusMultipliers(key)
		require.Equal(t, num.DecimalFromInt64(1500), summed.QuantumBalance)
		require.Equal(t, num.DecimalFromInt64(3), summed.Multiplier)
	}

	// all the calls above should be served from cache
	require.Equal(t, 5, v.col.GetVestingQuantumBalanceCallCount())

	v.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// now we simulate the end of the epoch
	// it will reset cache for reward bonus multipliers
	v.OnEpochEvent(ctx, types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
		Seq:    1,
	})

	v.col.ResetVestingQuantumBalanceCallCount()

	for _, key := range append(derivedKeys, party) {
		_, summed := v.GetSingleAndSummedRewardBonusMultipliers(key)
		require.Equal(t, num.DecimalFromInt64(1500), summed.QuantumBalance)
		require.Equal(t, num.DecimalFromInt64(3), summed.Multiplier)
	}

	// now it's called 5 times again because the cache gets reset at the end of the epoch
	require.Equal(t, 5, v.col.GetVestingQuantumBalanceCallCount())

	v.OnEpochEvent(ctx, types.Epoch{
		Action: vegapb.EpochAction_EPOCH_ACTION_END,
		Seq:    1,
	})

	v.col.ResetVestingQuantumBalanceCallCount()

	for _, key := range append(derivedKeys, party) {
		single, summed := v.GetSingleAndSummedRewardBonusMultipliers(key)
		require.Equal(t, num.DecimalFromInt64(1500), summed.QuantumBalance)
		require.Equal(t, num.DecimalFromInt64(3), summed.Multiplier)

		if key == party {
			require.Equal(t, num.DecimalFromInt64(500), single.QuantumBalance)
			require.Equal(t, num.DecimalFromInt64(2), single.Multiplier)
		} else {
			require.Equal(t, num.DecimalFromInt64(250), single.QuantumBalance)
			require.Equal(t, num.DecimalFromInt64(1), single.Multiplier)
		}
	}

	// now it's called 5 times again because the cache gets reset at the end of the epoch
	require.Equal(t, 5, v.col.GetVestingQuantumBalanceCallCount())
}

// LedgerMovements is the result of a mock, so it doesn't really make sense to
// verify data consistency.
func expectLedgerMovements(t *testing.T, v *testEngine) {
	t.Helper()

	v.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		e, ok := evt.(*events.LedgerMovements)
		require.True(t, ok, "Event should be a LedgerMovements, but is %T", evt)
		assert.Equal(t, eventspb.LedgerMovements{LedgerMovements: []*vegapb.LedgerMovement{}}, e.Proto())
	}).Times(1)
}
