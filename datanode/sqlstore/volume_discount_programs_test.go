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
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupVolumeDiscountProgramTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.VolumeDiscountPrograms) {
	t.Helper()

	bs := sqlstore.NewBlocks(connectionSource)
	rs := sqlstore.NewVolumeDiscountPrograms(connectionSource)

	return bs, rs
}

func TestVolumeDiscountPrograms_AddVolumeDiscountProgram(t *testing.T) {
	bs, rs := setupVolumeDiscountProgramTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	block2 := addTestBlock(t, ctx, bs)

	t.Run("AddVolumeDiscountProgram should create a new referral program record", func(t *testing.T) {
		endTime := block.VegaTime.Add(time.Hour)
		endTime2 := block2.VegaTime.Add(time.Hour)

		programs := []*eventspb.VolumeDiscountProgramStarted{
			{
				Program: &vega.VolumeDiscountProgram{
					Version: 1,
					Id:      GenerateID(),
					BenefitTiers: []*vega.VolumeBenefitTier{
						{
							MinimumRunningNotionalTakerVolume: "1000",
							VolumeDiscountFactor:              "0.01",
							VolumeDiscountFactors: &vega.DiscountFactors{
								InfrastructureDiscountFactor: "0.004",
								LiquidityDiscountFactor:      "0.002",
								MakerDiscountFactor:          "0.004",
							},
						},
						{
							MinimumRunningNotionalTakerVolume: "10000",
							VolumeDiscountFactor:              "0.1",
							VolumeDiscountFactors: &vega.DiscountFactors{
								InfrastructureDiscountFactor: "0.04",
								LiquidityDiscountFactor:      "0.02",
								MakerDiscountFactor:          "0.04",
							},
						},
					},
					EndOfProgramTimestamp: endTime.Unix(),
					WindowLength:          100,
				},
			},
			{
				Program: &vega.VolumeDiscountProgram{
					Version: 1,
					Id:      GenerateID(),
					BenefitTiers: []*vega.VolumeBenefitTier{
						{
							MinimumRunningNotionalTakerVolume: "2000",
							VolumeDiscountFactor:              "0.02",
							VolumeDiscountFactors: &vega.DiscountFactors{
								InfrastructureDiscountFactor: "0.008",
								LiquidityDiscountFactor:      "0.002",
								MakerDiscountFactor:          "0.004",
							},
						},
						{
							MinimumRunningNotionalTakerVolume: "20000",
							VolumeDiscountFactor:              "0.2",
							VolumeDiscountFactors: &vega.DiscountFactors{
								InfrastructureDiscountFactor: "0.08",
								LiquidityDiscountFactor:      "0.04",
								MakerDiscountFactor:          "0.08",
							},
						},
					},
					EndOfProgramTimestamp: endTime2.Unix(),
					WindowLength:          200,
				},
			},
		}

		want := entities.VolumeDiscountProgramFromProto(programs[0].Program, block.VegaTime, 0)
		err := rs.AddVolumeDiscountProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.VolumeDiscountProgram
		require.NoError(t, pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_discount_programs"))
		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		want2 := entities.VolumeDiscountProgramFromProto(programs[1].Program, block2.VegaTime, 0)
		err = rs.AddVolumeDiscountProgram(ctx, want2)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_discount_programs")
		require.NoError(t, err)
		require.Len(t, got, 2)
		wantAll := []entities.VolumeDiscountProgram{*want, *want2}
		assert.Equal(t, wantAll, got)
	})
}

func getVolumeDiscountEvents(t *testing.T, endTime time.Time) (*eventspb.VolumeDiscountProgramStarted,
	*eventspb.VolumeDiscountProgramUpdated, *eventspb.VolumeDiscountProgramEnded,
) {
	t.Helper()

	started := eventspb.VolumeDiscountProgramStarted{
		Program: &vega.VolumeDiscountProgram{
			Version: 1,
			Id:      GenerateID(),
			BenefitTiers: []*vega.VolumeBenefitTier{
				{
					MinimumRunningNotionalTakerVolume: "1000",
					VolumeDiscountFactor:              "0.01",
					VolumeDiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.004",
						LiquidityDiscountFactor:      "0.002",
						MakerDiscountFactor:          "0.004",
					},
				},
				{
					MinimumRunningNotionalTakerVolume: "10000",
					VolumeDiscountFactor:              "0.1",
					VolumeDiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.04",
						LiquidityDiscountFactor:      "0.02",
						MakerDiscountFactor:          "0.04",
					},
				},
			},
			EndOfProgramTimestamp: endTime.Unix(),
			WindowLength:          100,
		},
		StartedAt: endTime.Add(-1 * time.Hour).UnixNano(),
		AtEpoch:   1,
	}

	updated := eventspb.VolumeDiscountProgramUpdated{
		Program: &vega.VolumeDiscountProgram{
			Version: 2,
			Id:      GenerateID(),
			BenefitTiers: []*vega.VolumeBenefitTier{
				{
					MinimumRunningNotionalTakerVolume: "2000",
					VolumeDiscountFactor:              "0.02",
					VolumeDiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.008",
						LiquidityDiscountFactor:      "0.004",
						MakerDiscountFactor:          "0.008",
					},
				},
				{
					MinimumRunningNotionalTakerVolume: "20000",
					VolumeDiscountFactor:              "0.2",
					VolumeDiscountFactors: &vega.DiscountFactors{
						InfrastructureDiscountFactor: "0.08",
						LiquidityDiscountFactor:      "0.04",
						MakerDiscountFactor:          "0.08",
					},
				},
			},
			EndOfProgramTimestamp: endTime.Unix(),
			WindowLength:          200,
		},
		UpdatedAt: endTime.Add(-30 * time.Minute).UnixNano(),
		AtEpoch:   2,
	}

	ended := eventspb.VolumeDiscountProgramEnded{
		Version: 2,
		Id:      updated.Program.Id,
		EndedAt: endTime.UnixNano(),
		AtEpoch: 3,
	}

	return &started, &updated, &ended
}

func TestVolumeDiscountPrograms_UpdateVolumeDiscountProgram(t *testing.T) {
	bs, rs := setupVolumeDiscountProgramTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	endTime := block.VegaTime.Add(time.Hour)
	started, updated, _ := getVolumeDiscountEvents(t, endTime)

	var want, wantUpdated *entities.VolumeDiscountProgram
	t.Run("UpdateVolumeDiscountProgram should create a new referral program record with the updated data", func(t *testing.T) {
		want = entities.VolumeDiscountProgramFromProto(started.Program, block.VegaTime, 0)
		err := rs.AddVolumeDiscountProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.VolumeDiscountProgram
		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_discount_programs")
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		block = addTestBlock(t, ctx, bs)
		wantUpdated = entities.VolumeDiscountProgramFromProto(updated.Program, block.VegaTime, 0)
		err = rs.UpdateVolumeDiscountProgram(ctx, wantUpdated)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_discount_programs")
		require.NoError(t, err)

		require.Len(t, got, 2)

		wantAll := []entities.VolumeDiscountProgram{*want, *wantUpdated}
		assert.Equal(t, wantAll, got)
	})

	t.Run("The current_referral view should list the updated referral program record", func(t *testing.T) {
		var got []entities.VolumeDiscountProgram
		err := pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM current_volume_discount_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *wantUpdated, got[0])
	})
}

func TestVolumeDiscountPrograms_EndVolumeDiscountProgram(t *testing.T) {
	bs, rs := setupVolumeDiscountProgramTest(t)
	ctx := tempTransaction(t)

	t.Run("EndVolumeDiscountProgram should create a new referral program record with the data from the current referral program and set the ended_at timestamp", func(t *testing.T) {
		block := addTestBlock(t, ctx, bs)
		endTime := block.VegaTime.Add(time.Hour)
		startedEvent, updatedEvent, endedEvent := getVolumeDiscountEvents(t, endTime)

		started := entities.VolumeDiscountProgramFromProto(startedEvent.Program, block.VegaTime, 1)
		err := rs.AddVolumeDiscountProgram(ctx, started)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		updated := entities.VolumeDiscountProgramFromProto(updatedEvent.Program, block.VegaTime, 2)
		err = rs.UpdateVolumeDiscountProgram(ctx, updated)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndVolumeDiscountProgram(ctx, endedEvent.Version, endTime, block.VegaTime, 3)
		require.NoError(t, err)

		ended := entities.VolumeDiscountProgramFromProto(updatedEvent.Program, block.VegaTime, 3)
		ended.Version = endedEvent.Version
		ended.EndedAt = &endTime

		var got []entities.VolumeDiscountProgram
		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_discount_programs order by vega_time")
		require.NoError(t, err)
		require.Len(t, got, 3)
		wantAll := []entities.VolumeDiscountProgram{*started, *updated, *ended}
		assert.Equal(t, wantAll, got)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM current_volume_discount_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *ended, got[0])
	})
}

func TestVolumeDiscountPrograms_GetCurrentVolumeDiscountProgram(t *testing.T) {
	bs, rs := setupVolumeDiscountProgramTest(t)
	ctx := tempTransaction(t)

	t.Run("GetCurrentVolumeDiscountProgram should return the current referral program information", func(t *testing.T) {
		block := addTestBlock(t, ctx, bs)
		endTime := block.VegaTime.Add(time.Hour)
		startedEvent, updatedEvent, endedEvent := getVolumeDiscountEvents(t, endTime)

		started := entities.VolumeDiscountProgramFromProto(startedEvent.Program, block.VegaTime, 1)
		err := rs.AddVolumeDiscountProgram(ctx, started)
		require.NoError(t, err)

		got, err := rs.GetCurrentVolumeDiscountProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *started, got)

		block = addTestBlock(t, ctx, bs)
		updated := entities.VolumeDiscountProgramFromProto(updatedEvent.Program, block.VegaTime, 2)
		err = rs.UpdateVolumeDiscountProgram(ctx, updated)
		require.NoError(t, err)

		got, err = rs.GetCurrentVolumeDiscountProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *updated, got)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndVolumeDiscountProgram(ctx, endedEvent.Version, endTime, block.VegaTime, 3)
		require.NoError(t, err)

		ended := entities.VolumeDiscountProgramFromProto(updatedEvent.Program, block.VegaTime, 3)
		ended.Version = endedEvent.Version
		ended.EndedAt = &endTime

		got, err = rs.GetCurrentVolumeDiscountProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *ended, got)
	})
}
