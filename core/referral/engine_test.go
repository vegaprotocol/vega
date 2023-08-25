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

package referral_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferralSet(t *testing.T) {
	te := newEngine(t)

	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	setID := vgrand.RandomStr(5)
	referrer := "referrer"
	referee1 := "referee1"

	t.Run("querying for a non existing set return false", func(t *testing.T) {
		require.False(t, te.engine.SetExists(setID))
	})

	t.Run("cannot join a non-existing set", func(t *testing.T) {
		err := te.engine.ApplyReferralCode(ctx, referee1, &commandspb.ApplyReferralCode{
			Id: setID,
		})

		assert.EqualError(t, err, referral.ErrInvalidReferralCode(setID).Error())
	})

	t.Run("can create a set for the first time", func(t *testing.T) {
		te.broker.EXPECT().Send(gomock.Any()).Times(1)
		te.timeSvc.EXPECT().GetTimeNow().Times(1)
		assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer, &commandspb.CreateReferralSet{}, setID))
	})

	t.Run("cannot create a set multiple times", func(t *testing.T) {
		assert.EqualError(t, te.engine.CreateReferralSet(ctx, referrer, &commandspb.CreateReferralSet{}, setID),
			referral.ErrIsAlreadyAReferrer(referrer).Error(),
		)
	})

	t.Run("can join an existing set", func(t *testing.T) {
		te.broker.EXPECT().Send(gomock.Any()).Times(1)
		te.timeSvc.EXPECT().GetTimeNow().Times(1)
		assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee1, &commandspb.ApplyReferralCode{Id: setID}))
	})

	t.Run("cannot create a team when being a referee", func(t *testing.T) {
		assert.EqualError(t, te.engine.CreateReferralSet(ctx, referee1, &commandspb.CreateReferralSet{}, setID),
			referral.ErrIsAlreadyAReferee(referee1).Error(),
		)
	})

	t.Run("cannot become a referee twice", func(t *testing.T) {
		assert.EqualError(t, te.engine.ApplyReferralCode(ctx, referee1, &commandspb.ApplyReferralCode{Id: setID}),
			referral.ErrIsAlreadyAReferee(referee1).Error(),
		)
	})
}

func TestUpdatingReferralProgramSucceeds(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	require.True(t, te.engine.HasProgramEnded(), "There is no program yet, so the engine should behave as a program ended.")

	program1 := &types.ReferralProgram{
		EndOfProgramTimestamp: time.Now().Add(24 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Set the first program.
	te.engine.UpdateProgram(program1)

	require.True(t, te.engine.HasProgramEnded(), "The program should start only on the next epoch")

	// Simulating end of epoch.
	// The program should be applied.
	expectReferralProgramStartedEvent(t, te)
	lastEpochStartTime := program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	require.False(t, te.engine.HasProgramEnded(), "The program should have started.")

	// Simulating end of epoch.
	// The program should have reached its end.
	expectReferralProgramEndedEvent(t, te)
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(1 * time.Second)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	require.True(t, te.engine.HasProgramEnded(), "The program should have reached its ending.")

	program2 := &types.ReferralProgram{
		EndOfProgramTimestamp: lastEpochStartTime.Add(10 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Set second the program.
	te.engine.UpdateProgram(program2)

	require.True(t, te.engine.HasProgramEnded(), "The program should start only on the next epoch")

	program3 := &types.ReferralProgram{
		// Ending the program before the second one to show the engine replace the
		// the previous program by this one
		EndOfProgramTimestamp: program2.EndOfProgramTimestamp.Add(-5 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Override the second program by a third.
	te.engine.UpdateProgram(program3)

	// Simulating end of epoch.
	// The third program should have started.
	expectReferralProgramStartedEvent(t, te)
	lastEpochStartTime = program3.EndOfProgramTimestamp.Add(-1 * time.Second)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	require.False(t, te.engine.HasProgramEnded(), "The program should have started.")

	program4 := &types.ReferralProgram{
		EndOfProgramTimestamp: lastEpochStartTime.Add(10 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Update to replace the third program by the fourth one.
	te.engine.UpdateProgram(program4)

	// Simulating end of epoch.
	// The current program should have been updated by the fourth one.
	expectReferralProgramUpdatedEvent(t, te)
	lastEpochStartTime = program4.EndOfProgramTimestamp.Add(-1 * time.Second)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	program5 := &types.ReferralProgram{
		EndOfProgramTimestamp: lastEpochStartTime.Add(10 * time.Hour),
		WindowLength:          10,
		BenefitTiers:          []*types.BenefitTier{},
	}

	// Update with new program.
	te.engine.UpdateProgram(program5)

	require.False(t, te.engine.HasProgramEnded(), "The fourth program should still be up")

	// Simulating end of epoch.
	// The fifth program should have ended before it even started.
	gomock.InOrder(
		expectReferralProgramUpdatedEvent(t, te),
		expectReferralProgramEndedEvent(t, te),
	)
	lastEpochStartTime = program5.EndOfProgramTimestamp.Add(1 * time.Second)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	require.True(t, te.engine.HasProgramEnded(), "The program should have ended before it even started")
}

func TestGettingRewardFactor(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	program1 := &types.ReferralProgram{
		EndOfProgramTimestamp: time.Now().Add(24 * time.Hour),
		WindowLength:          10,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.UintFromUint64(2),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(1000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.001),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.002),
			}, {
				MinimumEpochs:                     num.UintFromUint64(10),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(10000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.01),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.02),
			}, {
				MinimumEpochs:                     num.UintFromUint64(20),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(100000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.1),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.2),
			},
		},
	}

	// Set the first program.
	te.engine.UpdateProgram(program1)

	expectReferralProgramStartedEvent(t, te)
	lastEpochStartTime := program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Looking for reward factor for party without a team.
	loneWolfParty := newPartyID(t)
	// te.teamsEngine.EXPECT().IsTeamMember(loneWolfParty).Return(false)
	assert.Equal(t, num.DecimalZero(), te.engine.RewardsFactorForParty(loneWolfParty))

	// Looking for reward factor for party with a team, but not for long enough.
	noobParty := newPartyID(t)
	// te.teamsEngine.EXPECT().IsTeamMember(noobParty).Return(true)
	// te.teamsEngine.EXPECT().NumberOfEpochInTeamForParty(noobParty).Return(uint64(1))
	assert.Equal(t, num.DecimalZero(), te.engine.RewardsFactorForParty(noobParty))

	// FIXME: Re-enabled in a following PR.
	// // Looking for reward factor for party with a team, matching tier 2.
	// eligibleParty := newPartyID(t)
	// te.teamsEngine.EXPECT().IsTeamMember(eligibleParty).Return(true)
	// te.teamsEngine.EXPECT().NumberOfEpochInTeamForParty(eligibleParty).Return(uint64(13))
	// assert.Equal(t, num.DecimalFromFloat(0.01), te.engine.RewardsFactorForParty(eligibleParty))
}
