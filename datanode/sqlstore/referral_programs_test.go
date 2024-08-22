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

package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupReferralProgramTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.ReferralPrograms) {
	t.Helper()

	bs := sqlstore.NewBlocks(connectionSource)
	rs := sqlstore.NewReferralPrograms(connectionSource)

	return bs, rs
}

func TestReferralPrograms_AddReferralProgram(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	block2 := addTestBlock(t, ctx, bs)

	t.Run("AddReferralProgram should create a new referral program record", func(t *testing.T) {
		endTime := block.VegaTime.Add(time.Hour)
		endTime2 := block2.VegaTime.Add(time.Hour)

		programs := []*eventspb.ReferralProgramStarted{
			{
				Program: &vega.ReferralProgram{
					Version: 1,
					Id:      GenerateID(),
					BenefitTiers: []*vega.BenefitTier{
						{
							MinimumRunningNotionalTakerVolume: "1000",
							MinimumEpochs:                     "10",
							ReferralRewardFactor:              "0.0001",
							ReferralDiscountFactor:            "0.0001",
							ReferralRewardFactors: &vega.RewardFactors{
								InfrastructureRewardFactor: "0.00002",
								LiquidityRewardFactor:      "0.00004",
								MakerRewardFactor:          "0.00004",
							},
							ReferralDiscountFactors: &vega.DiscountFactors{
								InfrastructureDiscountFactor: "0.00002",
								LiquidityDiscountFactor:      "0.00004",
								MakerDiscountFactor:          "0.00004",
							},
							TierNumber: ptr.From(uint64(1)),
						},
						{
							MinimumRunningNotionalTakerVolume: "10000",
							MinimumEpochs:                     "100",
							ReferralRewardFactor:              "0.001",
							ReferralDiscountFactor:            "0.001",
							ReferralRewardFactors: &vega.RewardFactors{
								InfrastructureRewardFactor: "0.0002",
								LiquidityRewardFactor:      "0.0004",
								MakerRewardFactor:          "0.0004",
							},
							ReferralDiscountFactors: &vega.DiscountFactors{
								InfrastructureDiscountFactor: "0.0002",
								LiquidityDiscountFactor:      "0.0004",
								MakerDiscountFactor:          "0.0004",
							},
							TierNumber: ptr.From(uint64(2)),
						},
					},
					EndOfProgramTimestamp: endTime.Unix(),
					WindowLength:          100,
					StakingTiers: []*vega.StakingTier{
						{
							MinimumStakedTokens:      "1000",
							ReferralRewardMultiplier: "1.0",
						},
						{
							MinimumStakedTokens:      "10000",
							ReferralRewardMultiplier: "1.1",
						},
					},
				},
			},
			{
				Program: &vega.ReferralProgram{
					Version: 1,
					Id:      GenerateID(),
					BenefitTiers: []*vega.BenefitTier{
						{
							MinimumRunningNotionalTakerVolume: "2000",
							MinimumEpochs:                     "20",
							ReferralRewardFactor:              "0.0002",
							ReferralDiscountFactor:            "0.0002",
							ReferralRewardFactors: &vega.RewardFactors{
								InfrastructureRewardFactor: "0.00004",
								LiquidityRewardFactor:      "0.00008",
								MakerRewardFactor:          "0.00008",
							},
							ReferralDiscountFactors: &vega.DiscountFactors{
								InfrastructureDiscountFactor: "0.00004",
								LiquidityDiscountFactor:      "0.00008",
								MakerDiscountFactor:          "0.00008",
							},
							TierNumber: ptr.From(uint64(1)),
						},
						{
							MinimumRunningNotionalTakerVolume: "20000",
							MinimumEpochs:                     "200",
							ReferralRewardFactor:              "0.002",
							ReferralDiscountFactor:            "0.002",
							ReferralRewardFactors: &vega.RewardFactors{
								InfrastructureRewardFactor: "0.0004",
								LiquidityRewardFactor:      "0.0008",
								MakerRewardFactor:          "0.0008",
							},
							ReferralDiscountFactors: &vega.DiscountFactors{
								InfrastructureDiscountFactor: "0.0004",
								LiquidityDiscountFactor:      "0.0008",
								MakerDiscountFactor:          "0.0008",
							},
							TierNumber: ptr.From(uint64(2)),
						},
					},
					EndOfProgramTimestamp: endTime2.Unix(),
					WindowLength:          200,
					StakingTiers: []*vega.StakingTier{
						{
							MinimumStakedTokens:      "1000",
							ReferralRewardMultiplier: "1.0",
						},
						{
							MinimumStakedTokens:      "10000",
							ReferralRewardMultiplier: "1.1",
						},
					},
				},
			},
		}

		want := entities.ReferralProgramFromProto(programs[0].Program, block.VegaTime, 0)
		err := rs.AddReferralProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.ReferralProgram
		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM referral_programs")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		want2 := entities.ReferralProgramFromProto(programs[1].Program, block2.VegaTime, 0)
		err = rs.AddReferralProgram(ctx, want2)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM referral_programs")
		require.NoError(t, err)
		require.Len(t, got, 2)
		wantAll := []entities.ReferralProgram{*want, *want2}
		assert.Equal(t, wantAll, got)
	})
}

func getReferralEvents(t *testing.T, endTime time.Time) (*eventspb.ReferralProgramStarted,
	*eventspb.ReferralProgramUpdated, *eventspb.ReferralProgramEnded,
) {
	t.Helper()

	started := eventspb.ReferralProgramStarted{
		Program: &vega.ReferralProgram{
			Version: 1,
			Id:      GenerateID(),
			BenefitTiers: []*vega.BenefitTier{
				{
					MinimumRunningNotionalTakerVolume: "1000",
					MinimumEpochs:                     "10",
					ReferralRewardFactor:              "0.0001",
					ReferralDiscountFactor:            "0.0001",
					ReferralRewardFactors: &vega.RewardFactors{
						InfrastructureRewardFactor: "0.00002",
						LiquidityRewardFactor:      "0.00004",
						MakerRewardFactor:          "0.00004",
					},
					ReferralDiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.00002",
						LiquidityDiscountFactor:      "0.00004",
						MakerDiscountFactor:          "0.00004",
					},
					TierNumber: ptr.From(uint64(1)),
				},
				{
					MinimumRunningNotionalTakerVolume: "10000",
					MinimumEpochs:                     "100",
					ReferralRewardFactor:              "0.001",
					ReferralDiscountFactor:            "0.001",
					ReferralRewardFactors: &vega.RewardFactors{
						InfrastructureRewardFactor: "0.0002",
						LiquidityRewardFactor:      "0.0004",
						MakerRewardFactor:          "0.0004",
					},
					ReferralDiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.0002",
						LiquidityDiscountFactor:      "0.0004",
						MakerDiscountFactor:          "0.0004",
					},
					TierNumber: ptr.From(uint64(2)),
				},
			},
			EndOfProgramTimestamp: endTime.Unix(),
			WindowLength:          100,
			StakingTiers: []*vega.StakingTier{
				{
					MinimumStakedTokens:      "1000",
					ReferralRewardMultiplier: "1.0",
				},
				{
					MinimumStakedTokens:      "10000",
					ReferralRewardMultiplier: "1.1",
				},
			},
		},
		StartedAt: endTime.Add(-1 * time.Hour).UnixNano(),
		AtEpoch:   40,
	}

	updated := eventspb.ReferralProgramUpdated{
		Program: &vega.ReferralProgram{
			Version: 2,
			Id:      started.Program.Id,
			BenefitTiers: []*vega.BenefitTier{
				{
					MinimumRunningNotionalTakerVolume: "2000",
					MinimumEpochs:                     "20",
					ReferralRewardFactor:              "0.0002",
					ReferralDiscountFactor:            "0.0002",
					ReferralRewardFactors: &vega.RewardFactors{
						InfrastructureRewardFactor: "0.0004",
						LiquidityRewardFactor:      "0.0008",
						MakerRewardFactor:          "0.0008",
					},
					ReferralDiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.0004",
						LiquidityDiscountFactor:      "0.0008",
						MakerDiscountFactor:          "0.0008",
					},
					TierNumber: ptr.From(uint64(1)),
				},
				{
					MinimumRunningNotionalTakerVolume: "20000",
					MinimumEpochs:                     "200",
					ReferralRewardFactor:              "0.002",
					ReferralDiscountFactor:            "0.002",
					ReferralRewardFactors: &vega.RewardFactors{
						InfrastructureRewardFactor: "0.004",
						LiquidityRewardFactor:      "0.008",
						MakerRewardFactor:          "0.008",
					},
					ReferralDiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.004",
						LiquidityDiscountFactor:      "0.008",
						MakerDiscountFactor:          "0.008",
					},
					TierNumber: ptr.From(uint64(2)),
				},
			},
			EndOfProgramTimestamp: endTime.Unix(),
			WindowLength:          200,
			StakingTiers: []*vega.StakingTier{
				{
					MinimumStakedTokens:      "1000",
					ReferralRewardMultiplier: "1.0",
				},
				{
					MinimumStakedTokens:      "10000",
					ReferralRewardMultiplier: "1.1",
				},
			},
		},
		UpdatedAt: endTime.Add(-30 * time.Minute).UnixNano(),
		AtEpoch:   41,
	}

	ended := eventspb.ReferralProgramEnded{
		Version: 2,
		Id:      started.Program.Id,
		EndedAt: endTime.UnixNano(),
		AtEpoch: 42,
	}

	return &started, &updated, &ended
}

func TestReferralPrograms_UpdateReferralProgram(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	endTime := block.VegaTime.Add(time.Hour)
	started, updated, _ := getReferralEvents(t, endTime)

	var want, wantUpdated *entities.ReferralProgram
	t.Run("UpdateReferralProgram should create a new referral program record with the updated data", func(t *testing.T) {
		want = entities.ReferralProgramFromProto(started.Program, block.VegaTime, 0)
		err := rs.AddReferralProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.ReferralProgram
		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM referral_programs")
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		block = addTestBlock(t, ctx, bs)
		wantUpdated = entities.ReferralProgramFromProto(updated.Program, block.VegaTime, 0)
		err = rs.UpdateReferralProgram(ctx, wantUpdated)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM referral_programs")
		require.NoError(t, err)

		require.Len(t, got, 2)

		wantAll := []entities.ReferralProgram{*want, *wantUpdated}
		assert.Equal(t, wantAll, got)
	})

	t.Run("The current_referral view should list the updated referral program record", func(t *testing.T) {
		var got []entities.ReferralProgram
		err := pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM current_referral_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *wantUpdated, got[0])
	})
}

func TestReferralPrograms_EndReferralProgram(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx := tempTransaction(t)

	t.Run("EndReferralProgram should create a new referral program record with the data from the current referral program and set the ended_at timestamp", func(t *testing.T) {
		block := addTestBlock(t, ctx, bs)
		endTime := block.VegaTime.Add(time.Hour)
		startedEvent, updatedEvent, endedEvent := getReferralEvents(t, endTime)

		started := entities.ReferralProgramFromProto(startedEvent.Program, block.VegaTime, 1)
		err := rs.AddReferralProgram(ctx, started)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		updated := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime, 2)
		err = rs.UpdateReferralProgram(ctx, updated)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndReferralProgram(ctx, endedEvent.Version, endTime, block.VegaTime, 3)
		require.NoError(t, err)

		ended := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime, 3)
		ended.Version = endedEvent.Version
		ended.EndedAt = &endTime

		var got []entities.ReferralProgram
		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM referral_programs order by vega_time")
		require.NoError(t, err)
		require.Len(t, got, 3)
		wantAll := []entities.ReferralProgram{*started, *updated, *ended}
		assert.Equal(t, wantAll, got)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM current_referral_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *ended, got[0])
	})
}

func TestReferralPrograms_GetCurrentReferralProgram(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx := tempTransaction(t)

	t.Run("GetCurrentReferralProgram should return the current referral program information", func(t *testing.T) {
		block := addTestBlock(t, ctx, bs)
		endTime := block.VegaTime.Add(time.Hour)
		startedEvent, updatedEvent, endedEvent := getReferralEvents(t, endTime)

		started := entities.ReferralProgramFromProto(startedEvent.Program, block.VegaTime, 1)
		err := rs.AddReferralProgram(ctx, started)
		require.NoError(t, err)

		got, err := rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *started, got)

		block = addTestBlock(t, ctx, bs)
		updated := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime, 2)
		err = rs.UpdateReferralProgram(ctx, updated)
		require.NoError(t, err)

		got, err = rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *updated, got)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndReferralProgram(ctx, endedEvent.Version, endTime, block.VegaTime, 3)
		require.NoError(t, err)

		updatedProgram := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime, 3)
		updatedProgram.Version = endedEvent.Version
		updatedProgram.EndedAt = &endTime

		got, err = rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *updatedProgram, got)
	})
}

func TestReferralPrograms_StartAndEndInSameBlock(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx := tempTransaction(t)

	t.Run("Data node should allow a referral program to be started and ended in the same block", func(t *testing.T) {
		block := addTestBlock(t, ctx, bs)
		endTime := block.VegaTime
		startedEvent, updatedEvent, endedEvent := getReferralEvents(t, endTime)

		started := entities.ReferralProgramFromProto(startedEvent.Program, block.VegaTime, 1)
		err := rs.AddReferralProgram(ctx, started)
		require.NoError(t, err)

		got, err := rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *started, got)

		updated := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime, 2)
		err = rs.UpdateReferralProgram(ctx, updated)
		require.NoError(t, err)

		got, err = rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *updated, got)

		err = rs.EndReferralProgram(ctx, endedEvent.Version, endTime, block.VegaTime, 3)
		require.NoError(t, err)

		ended := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime, 3)
		ended.Version = endedEvent.Version
		ended.EndedAt = &endTime

		got, err = rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *ended, got)
	})
}
