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

func setupReferralProgramTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.ReferralPrograms) {
	t.Helper()

	bs := sqlstore.NewBlocks(connectionSource)
	rs := sqlstore.NewReferralPrograms(connectionSource)

	return bs, rs
}

func TestReferralPrograms_AddReferralProgram(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()

	block := addTestBlock(t, ctx, bs)
	block2 := addTestBlock(t, ctx, bs)

	t.Run("AddReferralProgram should create a new referral program record", func(t *testing.T) {
		endTime := block.VegaTime.Add(time.Hour)
		endTime2 := block2.VegaTime.Add(time.Hour)

		programs := []*eventspb.ReferralProgramStarted{
			{
				Program: &vega.ReferralProgram{
					Version: 1,
					Id:      helpers.GenerateID(),
					BenefitTiers: []*vega.BenefitTier{
						{
							MinimumRunningNotionalTakerVolume: "1000",
							MinimumEpochs:                     "10",
							ReferralRewardFactor:              "0.0001",
							ReferralDiscountFactor:            "0.0001",
						},
						{
							MinimumRunningNotionalTakerVolume: "10000",
							MinimumEpochs:                     "100",
							ReferralRewardFactor:              "0.001",
							ReferralDiscountFactor:            "0.001",
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
					Id:      helpers.GenerateID(),
					BenefitTiers: []*vega.BenefitTier{
						{
							MinimumRunningNotionalTakerVolume: "2000",
							MinimumEpochs:                     "20",
							ReferralRewardFactor:              "0.0002",
							ReferralDiscountFactor:            "0.0002",
						},
						{
							MinimumRunningNotionalTakerVolume: "20000",
							MinimumEpochs:                     "200",
							ReferralRewardFactor:              "0.002",
							ReferralDiscountFactor:            "0.002",
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

		want := entities.ReferralProgramFromProto(programs[0].Program, block.VegaTime)
		err := rs.AddReferralProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.ReferralProgram
		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM referral_programs")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		want2 := entities.ReferralProgramFromProto(programs[1].Program, block2.VegaTime)
		err = rs.AddReferralProgram(ctx, want2)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM referral_programs")
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
			Id:      helpers.GenerateID(),
			BenefitTiers: []*vega.BenefitTier{
				{
					MinimumRunningNotionalTakerVolume: "1000",
					MinimumEpochs:                     "10",
					ReferralRewardFactor:              "0.0001",
					ReferralDiscountFactor:            "0.0001",
				},
				{
					MinimumRunningNotionalTakerVolume: "10000",
					MinimumEpochs:                     "100",
					ReferralRewardFactor:              "0.001",
					ReferralDiscountFactor:            "0.001",
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
				},
				{
					MinimumRunningNotionalTakerVolume: "20000",
					MinimumEpochs:                     "200",
					ReferralRewardFactor:              "0.002",
					ReferralDiscountFactor:            "0.002",
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
	}

	ended := eventspb.ReferralProgramEnded{
		Version: 3,
		Id:      started.Program.Id,
	}

	return &started, &updated, &ended
}

func TestReferralPrograms_UpdateReferralProgram(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()

	block := addTestBlock(t, ctx, bs)
	endTime := block.VegaTime.Add(time.Hour)
	started, updated, _ := getReferralEvents(t, endTime)

	var want, wantUpdated *entities.ReferralProgram
	t.Run("UpdateReferralProgram should create a new referral program record with the updated data", func(t *testing.T) {
		want = entities.ReferralProgramFromProto(started.Program, block.VegaTime)
		err := rs.AddReferralProgram(ctx, want)
		require.NoError(t, err)

		var got []entities.ReferralProgram
		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM referral_programs")
		require.NoError(t, err)

		require.Len(t, got, 1)
		assert.Equal(t, *want, got[0])

		block = addTestBlock(t, ctx, bs)
		wantUpdated = entities.ReferralProgramFromProto(updated.Program, block.VegaTime)
		err = rs.UpdateReferralProgram(ctx, wantUpdated)
		require.NoError(t, err)

		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM referral_programs")
		require.NoError(t, err)

		require.Len(t, got, 2)

		wantAll := []entities.ReferralProgram{*want, *wantUpdated}
		assert.Equal(t, wantAll, got)
	})

	t.Run("The current_referral view should list the updated referral program record", func(t *testing.T) {
		var got []entities.ReferralProgram
		err := pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM current_referral_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *wantUpdated, got[0])
	})
}

func TestReferralPrograms_EndReferralProgram(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()

	block := addTestBlock(t, ctx, bs)
	endTime := block.VegaTime.Add(time.Hour)
	startedEvent, updatedEvent, endedEvent := getReferralEvents(t, endTime)
	t.Run("EndReferralProgram should create a new referral program record with the data from the current referral program and set the ended_at timestamp", func(t *testing.T) {
		started := entities.ReferralProgramFromProto(startedEvent.Program, block.VegaTime)
		err := rs.AddReferralProgram(ctx, started)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		updated := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime)
		err = rs.UpdateReferralProgram(ctx, updated)
		require.NoError(t, err)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndReferralProgram(ctx, entities.ReferralProgramID(endedEvent.Id), endedEvent.Version, block.VegaTime)
		require.NoError(t, err)

		ended := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime)
		ended.Version = endedEvent.Version
		ended.EndedAt = &block.VegaTime

		var got []entities.ReferralProgram
		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM referral_programs order by vega_time")
		require.NoError(t, err)
		require.Len(t, got, 3)
		wantAll := []entities.ReferralProgram{*started, *updated, *ended}
		assert.Equal(t, wantAll, got)

		err = pgxscan.Select(ctx, connectionSource.Connection, &got, "SELECT * FROM current_referral_program")
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, *ended, got[0])
	})
}

func TestReferralPrograms_GetCurrentReferralProgram(t *testing.T) {
	bs, rs := setupReferralProgramTest(t)
	ctx, rollback := tempTransaction(t)
	defer rollback()

	block := addTestBlock(t, ctx, bs)
	endTime := block.VegaTime.Add(time.Hour)
	startedEvent, updatedEvent, endedEvent := getReferralEvents(t, endTime)

	t.Run("GetCurrentReferralProgram should return the current referral program information", func(t *testing.T) {
		started := entities.ReferralProgramFromProto(startedEvent.Program, block.VegaTime)
		err := rs.AddReferralProgram(ctx, started)
		require.NoError(t, err)

		got, err := rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *started, got)

		block = addTestBlock(t, ctx, bs)
		updated := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime)
		err = rs.UpdateReferralProgram(ctx, updated)
		require.NoError(t, err)

		got, err = rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *updated, got)

		block = addTestBlock(t, ctx, bs)
		err = rs.EndReferralProgram(ctx, entities.ReferralProgramID(endedEvent.Id), endedEvent.Version, block.VegaTime)
		require.NoError(t, err)

		ended := entities.ReferralProgramFromProto(updatedEvent.Program, block.VegaTime)
		ended.Version = endedEvent.Version
		ended.EndedAt = &block.VegaTime

		got, err = rs.GetCurrentReferralProgram(ctx)
		require.NoError(t, err)
		assert.Equal(t, *ended, got)
	})
}
