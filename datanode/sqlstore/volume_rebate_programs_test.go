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

func setupVolumeRebateProgramTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.VolumeRebatePrograms) {
	t.Helper()

	bs := sqlstore.NewBlocks(connectionSource)
	rs := sqlstore.NewVolumeRebatePrograms(connectionSource)

	return bs, rs
}

func TestVolumeRebatePrograms_AddVolumeRebateProgram(t *testing.T) {
	bs, rs := setupVolumeRebateProgramTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	block2 := addTestBlock(t, ctx, bs)
	one64 := uint64(1)

	t.Run("AddVolumeRebateProgram should create a new volume rebate program record", func(t *testing.T) {
		endTime := block.VegaTime.Add(time.Hour)
		endTime2 := block2.VegaTime.Add(time.Hour)

		programs := []*eventspb.VolumeRebateProgramStarted{
			{
				Program: &vega.VolumeRebateProgram{
					Version: 1,
					Id:      GenerateID(),
					BenefitTiers: []*vega.VolumeRebateBenefitTier{
						{
							MinimumPartyMakerVolumeFraction: "1",
							AdditionalMakerRebate:           "0.01",
							TierNumber:                      ptr.From(one64),
						},
						{
							MinimumPartyMakerVolumeFraction: "2",
							AdditionalMakerRebate:           "0.1",
							TierNumber:                      ptr.From(one64 * 2),
						},
					},
					EndOfProgramTimestamp: endTime.Unix(),
					WindowLength:          100,
				},
			},
			{
				Program: &vega.VolumeRebateProgram{
					Version: 1,
					Id:      GenerateID(),
					BenefitTiers: []*vega.VolumeRebateBenefitTier{
						{
							MinimumPartyMakerVolumeFraction: "3",
							AdditionalMakerRebate:           "0.02",
							TierNumber:                      ptr.From(one64),
						},
						{
							MinimumPartyMakerVolumeFraction: "4",
							AdditionalMakerRebate:           "0.2",
							TierNumber:                      ptr.From(one64 * 2),
						},
					},
					EndOfProgramTimestamp: endTime2.Unix(),
					WindowLength:          200,
				},
			},
		}

		want := entities.VolumeRebateProgramFromProto(programs[0].Program, block.VegaTime, 0)
		err := rs.AddVolumeRebateProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.VolumeRebateProgram
		require.NoError(t, pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_rebate_programs"))
		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		want2 := entities.VolumeRebateProgramFromProto(programs[1].Program, block2.VegaTime, 0)
		err = rs.AddVolumeRebateProgram(ctx, want2)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_rebate_programs")
		require.NoError(t, err)
		require.Len(t, got, 2)
		wantAll := []entities.VolumeRebateProgram{*want, *want2}
		assert.Equal(t, wantAll, got)
	})
}

func getVolumeRebateEvents(t *testing.T, endTime time.Time) (*eventspb.VolumeRebateProgramStarted,
	*eventspb.VolumeRebateProgramUpdated, *eventspb.VolumeRebateProgramEnded,
) {
	t.Helper()

	one64 := uint64(1)
	started := eventspb.VolumeRebateProgramStarted{
		Program: &vega.VolumeRebateProgram{
			Version: 1,
			Id:      GenerateID(),
			BenefitTiers: []*vega.VolumeRebateBenefitTier{
				{
					MinimumPartyMakerVolumeFraction: "1000",
					AdditionalMakerRebate:           "0.01",
					TierNumber:                      ptr.From(one64),
				},
				{
					MinimumPartyMakerVolumeFraction: "10000",
					AdditionalMakerRebate:           "0.1",
					TierNumber:                      ptr.From(one64 * 2),
				},
			},
			EndOfProgramTimestamp: endTime.Unix(),
			WindowLength:          100,
		},
		StartedAt: endTime.Add(-1 * time.Hour).UnixNano(),
		AtEpoch:   1,
	}

	updated := eventspb.VolumeRebateProgramUpdated{
		Program: &vega.VolumeRebateProgram{
			Version: 2,
			Id:      GenerateID(),
			BenefitTiers: []*vega.VolumeRebateBenefitTier{
				{
					MinimumPartyMakerVolumeFraction: "2000",
					AdditionalMakerRebate:           "0.02",
					TierNumber:                      ptr.From(one64),
				},
				{
					MinimumPartyMakerVolumeFraction: "20000",
					AdditionalMakerRebate:           "0.2",
					TierNumber:                      ptr.From(one64 * 2),
				},
			},
			EndOfProgramTimestamp: endTime.Unix(),
			WindowLength:          200,
		},
		UpdatedAt: endTime.Add(-30 * time.Minute).UnixNano(),
		AtEpoch:   2,
	}

	ended := eventspb.VolumeRebateProgramEnded{
		Version: 2,
		Id:      updated.Program.Id,
		EndedAt: endTime.UnixNano(),
		AtEpoch: 3,
	}

	return &started, &updated, &ended
}

func TestVolumeRebatePrograms_UpdateVolumeRebateProgram(t *testing.T) {
	bs, rs := setupVolumeRebateProgramTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	endTime := block.VegaTime.Add(time.Hour)
	started, updated, _ := getVolumeRebateEvents(t, endTime)

	var want, wantUpdated *entities.VolumeRebateProgram
	t.Run("UpdateVolumeRebateProgram should create a new referral program record with the updated data", func(t *testing.T) {
		want = entities.VolumeRebateProgramFromProto(started.Program, block.VegaTime, 0)
		err := rs.AddVolumeRebateProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.VolumeRebateProgram
		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_rebate_programs")
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		block = addTestBlock(t, ctx, bs)
		wantUpdated = entities.VolumeRebateProgramFromProto(updated.Program, block.VegaTime, 0)
		err = rs.UpdateVolumeRebateProgram(ctx, wantUpdated)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_rebate_programs")
		require.NoError(t, err)

		require.Len(t, got, 2)

		wantAll := []entities.VolumeRebateProgram{*want, *wantUpdated}
		assert.Equal(t, wantAll, got)
	})

	t.Run("The current_rebate view should list the updated rebate program record", func(t *testing.T) {
		var got []entities.VolumeRebateProgram
		err := pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM current_volume_rebate_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *wantUpdated, got[0])
	})
}

func TestVolumeRebatePrograms_EndVolumeRebateProgram(t *testing.T) {
	bs, rs := setupVolumeRebateProgramTest(t)
	ctx := tempTransaction(t)

	t.Run("EndVolumeRebateProgram should create a new rebate program record with the data from the current rebate program and set the ended_at timestamp", func(t *testing.T) {
		block := addTestBlock(t, ctx, bs)
		endTime := block.VegaTime.Add(time.Hour)
		startedEvent, updatedEvent, endedEvent := getVolumeRebateEvents(t, endTime)

		started := entities.VolumeRebateProgramFromProto(startedEvent.Program, block.VegaTime, 1)
		err := rs.AddVolumeRebateProgram(ctx, started)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		updated := entities.VolumeRebateProgramFromProto(updatedEvent.Program, block.VegaTime, 2)
		err = rs.UpdateVolumeRebateProgram(ctx, updated)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndVolumeRebateProgram(ctx, endedEvent.Version, endTime, block.VegaTime, 3)
		require.NoError(t, err)

		ended := entities.VolumeRebateProgramFromProto(updatedEvent.Program, block.VegaTime, 3)
		ended.Version = endedEvent.Version
		ended.EndedAt = &endTime

		var got []entities.VolumeRebateProgram
		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM volume_rebate_programs order by vega_time")
		require.NoError(t, err)
		require.Len(t, got, 3)
		wantAll := []entities.VolumeRebateProgram{*started, *updated, *ended}
		assert.Equal(t, wantAll, got)

		err = pgxscan.Select(ctx, connectionSource, &got, "SELECT * FROM current_volume_rebate_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *ended, got[0])
	})
}

func TestVolumeRebatePrograms_GetCurrentVolumeRebateProgram(t *testing.T) {
	bs, rs := setupVolumeRebateProgramTest(t)
	ctx := tempTransaction(t)

	t.Run("GetCurrentVolumeRebateProgram should return the current rebate program information", func(t *testing.T) {
		block := addTestBlock(t, ctx, bs)
		endTime := block.VegaTime.Add(time.Hour)
		startedEvent, updatedEvent, endedEvent := getVolumeRebateEvents(t, endTime)

		started := entities.VolumeRebateProgramFromProto(startedEvent.Program, block.VegaTime, 1)
		err := rs.AddVolumeRebateProgram(ctx, started)
		require.NoError(t, err)

		got, err := rs.GetCurrentVolumeRebateProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *started, got)

		block = addTestBlock(t, ctx, bs)
		updated := entities.VolumeRebateProgramFromProto(updatedEvent.Program, block.VegaTime, 2)
		err = rs.UpdateVolumeRebateProgram(ctx, updated)
		require.NoError(t, err)

		got, err = rs.GetCurrentVolumeRebateProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *updated, got)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndVolumeRebateProgram(ctx, endedEvent.Version, endTime, block.VegaTime, 3)
		require.NoError(t, err)

		ended := entities.VolumeRebateProgramFromProto(updatedEvent.Program, block.VegaTime, 3)
		ended.Version = endedEvent.Version
		ended.EndedAt = &endTime

		got, err = rs.GetCurrentVolumeRebateProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *ended, got)
	})
}
