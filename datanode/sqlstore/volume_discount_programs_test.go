package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
					Id:      helpers.GenerateID(),
					BenefitTiers: []*vega.VolumeBenefitTier{
						{
							MinimumRunningNotionalTakerVolume: "1000",
							VolumeDiscountFactor:              "0.01",
						},
						{
							MinimumRunningNotionalTakerVolume: "10000",
							VolumeDiscountFactor:              "0.1",
						},
					},
					EndOfProgramTimestamp: endTime.Unix(),
					WindowLength:          100,
				},
			},
			{
				Program: &vega.VolumeDiscountProgram{
					Version: 1,
					Id:      helpers.GenerateID(),
					BenefitTiers: []*vega.VolumeBenefitTier{
						{
							MinimumRunningNotionalTakerVolume: "2000",
							VolumeDiscountFactor:              "0.02",
						},
						{
							MinimumRunningNotionalTakerVolume: "20000",
							VolumeDiscountFactor:              "0.2",
						},
					},
					EndOfProgramTimestamp: endTime2.Unix(),
					WindowLength:          200,
				},
			},
		}

		want := entities.VolumeDiscountProgramFromProto(programs[0].Program, block.VegaTime)
		err := rs.AddVolumeDiscountProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.VolumeDiscountProgram
		require.NoError(t, pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM volume_discount_programs"))
		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		want2 := entities.VolumeDiscountProgramFromProto(programs[1].Program, block2.VegaTime)
		err = rs.AddVolumeDiscountProgram(ctx, want2)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM volume_discount_programs")
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
			Id:      helpers.GenerateID(),
			BenefitTiers: []*vega.VolumeBenefitTier{
				{
					MinimumRunningNotionalTakerVolume: "1000",
					VolumeDiscountFactor:              "0.01",
				},
				{
					MinimumRunningNotionalTakerVolume: "10000",
					VolumeDiscountFactor:              "0.1",
				},
			},
			EndOfProgramTimestamp: endTime.Unix(),
			WindowLength:          100,
		},
	}

	updated := eventspb.VolumeDiscountProgramUpdated{
		Program: &vega.VolumeDiscountProgram{
			Version: 2,
			Id:      started.Program.Id,
			BenefitTiers: []*vega.VolumeBenefitTier{
				{
					MinimumRunningNotionalTakerVolume: "2000",
					VolumeDiscountFactor:              "0.02",
				},
				{
					MinimumRunningNotionalTakerVolume: "20000",
					VolumeDiscountFactor:              "0.2",
				},
			},
			EndOfProgramTimestamp: endTime.Unix(),
			WindowLength:          200,
		},
	}

	ended := eventspb.VolumeDiscountProgramEnded{
		Version: 3,
		Id:      started.Program.Id,
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
		want = entities.VolumeDiscountProgramFromProto(started.Program, block.VegaTime)
		err := rs.AddVolumeDiscountProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.VolumeDiscountProgram
		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM volume_discount_programs")
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		block = addTestBlock(t, ctx, bs)
		wantUpdated = entities.VolumeDiscountProgramFromProto(updated.Program, block.VegaTime)
		err = rs.UpdateVolumeDiscountProgram(ctx, wantUpdated)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM volume_discount_programs")
		require.NoError(t, err)

		require.Len(t, got, 2)

		wantAll := []entities.VolumeDiscountProgram{*want, *wantUpdated}
		assert.Equal(t, wantAll, got)
	})

	t.Run("The current_referral view should list the updated referral program record", func(t *testing.T) {
		var got []entities.VolumeDiscountProgram
		err := pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM current_volume_discount_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *wantUpdated, got[0])
	})
}

func TestVolumeDiscountPrograms_EndVolumeDiscountProgram(t *testing.T) {
	bs, rs := setupVolumeDiscountProgramTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	endTime := block.VegaTime.Add(time.Hour)
	startedEvent, updatedEvent, endedEvent := getVolumeDiscountEvents(t, endTime)
	t.Run("EndVolumeDiscountProgram should create a new referral program record with the data from the current referral program and set the ended_at timestamp", func(t *testing.T) {
		started := entities.VolumeDiscountProgramFromProto(startedEvent.Program, block.VegaTime)
		err := rs.AddVolumeDiscountProgram(ctx, started)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		updated := entities.VolumeDiscountProgramFromProto(updatedEvent.Program, block.VegaTime)
		err = rs.UpdateVolumeDiscountProgram(ctx, updated)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndVolumeDiscountProgram(ctx, entities.VolumeDiscountProgramID(endedEvent.Id), endedEvent.Version, block.VegaTime)
		require.NoError(t, err)

		ended := entities.VolumeDiscountProgramFromProto(updatedEvent.Program, block.VegaTime)
		ended.Version = endedEvent.Version
		ended.EndedAt = &block.VegaTime

		var got []entities.VolumeDiscountProgram
		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM volume_discount_programs order by vega_time")
		require.NoError(t, err)
		require.Len(t, got, 3)
		wantAll := []entities.VolumeDiscountProgram{*started, *updated, *ended}
		assert.Equal(t, wantAll, got)

		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM current_volume_discount_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *ended, got[0])
	})
}

func TestVolumeDiscountPrograms_GetCurrentVolumeDiscountProgram(t *testing.T) {
	bs, rs := setupVolumeDiscountProgramTest(t)
	ctx := tempTransaction(t)

	block := addTestBlock(t, ctx, bs)
	endTime := block.VegaTime.Add(time.Hour)
	startedEvent, updatedEvent, endedEvent := getVolumeDiscountEvents(t, endTime)

	t.Run("GetCurrentVolumeDiscountProgram should return the current referral program information", func(t *testing.T) {
		started := entities.VolumeDiscountProgramFromProto(startedEvent.Program, block.VegaTime)
		err := rs.AddVolumeDiscountProgram(ctx, started)
		require.NoError(t, err)

		got, err := rs.GetCurrentVolumeDiscountProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *started, got)

		block = addTestBlock(t, ctx, bs)
		updated := entities.VolumeDiscountProgramFromProto(updatedEvent.Program, block.VegaTime)
		err = rs.UpdateVolumeDiscountProgram(ctx, updated)
		require.NoError(t, err)

		got, err = rs.GetCurrentVolumeDiscountProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *updated, got)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndVolumeDiscountProgram(ctx, entities.VolumeDiscountProgramID(endedEvent.Id), endedEvent.Version, block.VegaTime)
		require.NoError(t, err)

		ended := entities.VolumeDiscountProgramFromProto(updatedEvent.Program, block.VegaTime)
		ended.Version = endedEvent.Version
		ended.EndedAt = &block.VegaTime

		got, err = rs.GetCurrentVolumeDiscountProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *ended, got)
	})
}
