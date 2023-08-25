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
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferralSet(t *testing.T) {
	te := newEngine(t)

	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	setID := newSetID(t)
	referrer := newPartyID(t)
	referee1 := newPartyID(t)

	t.Run("querying for a non existing set return false", func(t *testing.T) {
		require.False(t, te.engine.SetExists(setID))
	})

	t.Run("cannot join a non-existing set", func(t *testing.T) {
		err := te.engine.ApplyReferralCode(ctx, referee1, setID)
		assert.EqualError(t, err, referral.ErrUnknownReferralCode(setID).Error())
	})

	t.Run("can create a set for the first time", func(t *testing.T) {
		te.broker.EXPECT().Send(gomock.Any()).Times(1)
		te.timeSvc.EXPECT().GetTimeNow().Times(1)
		assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer, setID))
	})

	t.Run("cannot create a set multiple times", func(t *testing.T) {
		assert.EqualError(t, te.engine.CreateReferralSet(ctx, referrer, setID),
			referral.ErrIsAlreadyAReferrer(referrer).Error(),
		)
	})

	t.Run("can join an existing set", func(t *testing.T) {
		te.broker.EXPECT().Send(gomock.Any()).Times(1)
		te.timeSvc.EXPECT().GetTimeNow().Times(1)
		assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee1, setID))
	})

	t.Run("cannot create a set when being a referee", func(t *testing.T) {
		assert.EqualError(t, te.engine.CreateReferralSet(ctx, referee1, setID),
			referral.ErrIsAlreadyAReferee(referee1).Error(),
		)
	})

	t.Run("cannot become a referee twice", func(t *testing.T) {
		assert.EqualError(t, te.engine.ApplyReferralCode(ctx, referee1, setID),
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

func TestGettingRewardAndDiscountFactors(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)

	setID1 := newSetID(t)
	setID2 := newSetID(t)
	referrer1 := newPartyID(t)
	referrer2 := newPartyID(t)
	referee1 := newPartyID(t)
	referee2 := newPartyID(t)
	maxVolumeParams := num.UintFromUint64(2000)

	// Cap the notional volume.
	require.NoError(t, te.engine.OnReferralProgramMaxPartyNotionalVolumeByQuantumPerEpochUpdate(ctx, maxVolumeParams))

	program1 := &types.ReferralProgram{
		EndOfProgramTimestamp: time.Now().Add(24 * time.Hour),
		WindowLength:          2,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.UintFromUint64(2),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(1000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.001),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.002),
			}, {
				MinimumEpochs:                     num.UintFromUint64(3),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(3000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.01),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.02),
			}, {
				MinimumEpochs:                     num.UintFromUint64(4),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(4000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.1),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.2),
			},
		},
	}

	// Set the first program.
	te.engine.UpdateProgram(program1)

	te.broker.EXPECT().Send(gomock.Any()).Times(3)
	te.timeSvc.EXPECT().GetTimeNow().Return(time.Now()).Times(3)
	assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer1, setID1))
	assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer2, setID2))
	assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee1, setID1))

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(800)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(20000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(100)).Times(1)

	// When the epoch starts, the new program should start.
	expectReferralProgramStartedEvent(t, te)
	lastEpochStartTime := program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Looking for reward factor for party without a set.
	loneWolfParty := newPartyID(t)
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(loneWolfParty).String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.DiscountFactorForParty(loneWolfParty).String())

	// Looking for reward factor for referrer 1.
	// His set has not enough to notional volume to match tier 1.
	// => No reward factor.
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer1).String())
	// Only referees are eligible to discount factors.
	assert.Equal(t, num.DecimalZero().String(), te.engine.DiscountFactorForParty(referrer1).String())

	// Looking for reward factor for referee 1.
	// He is not a member for long enough.
	// His set has not enough to notional volume to match tier 1.
	// => No discount factor.
	assert.Equal(t, num.DecimalZero().String(), te.engine.DiscountFactorForParty(referee1).String())
	// Only referrers are eligible to reward factors.
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referee1).String())

	// Looking for reward factor for referrer 2.
	// His set has not enough notional volume to match tier 2, because the volume is
	// capped at 2000, so it matches tier 1.
	// => Tier 1 reward factor.
	assert.Equal(t, num.DecimalFromFloat(0.001).String(), te.engine.RewardsFactorForParty(referrer2).String())
	// Only referees are eligible to discount factors.
	assert.Equal(t, num.DecimalZero().String(), te.engine.DiscountFactorForParty(referrer2).String())

	// Adding a new referee.
	te.broker.EXPECT().Send(gomock.Any()).Times(1)
	te.timeSvc.EXPECT().GetTimeNow().Return(time.Now()).Times(1)
	assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee2, setID2))

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(1900)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(1000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(1000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee2)).Return(num.UintFromUint64(1000)).Times(1)

	// When the epoch starts, the new program should start.
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-1 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Looking for reward factor for referrer 1.
	// His set has enough to notional volume to match tier 2.
	// => Tier 2 reward factor.
	assert.Equal(t, num.DecimalFromFloat(0.01).String(), te.engine.RewardsFactorForParty(referrer1).String())

	// Looking for reward factor for referee 1.
	// With set, and member for long enough to match tier 2.
	// His set has enough notional volume to match tier 3.
	// => Tier 2 discount factor.
	assert.Equal(t, num.DecimalFromFloat(0.002).String(), te.engine.DiscountFactorForParty(referee1).String())

	// Looking for reward factor for referrer 2.
	// His set has enough notional volume to match tier 3.
	// => Tier 3 reward factor.
	assert.Equal(t, num.DecimalFromFloat(0.1).String(), te.engine.RewardsFactorForParty(referrer2).String())

	// Looking for reward factor for referee 2.
	// With set, but not member for long enough to match tier 1.
	// His set has enough notional volume to match tier 3.
	// => No discount factor.
	assert.Equal(t, num.DecimalZero().String(), te.engine.DiscountFactorForParty(referee2).String())

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(10)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(10)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(500)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee2)).Return(num.UintFromUint64(500)).Times(1)

	// When the epoch starts, the new program should start.
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-1 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Because the window length is set to 2, the first notional volumes are now
	// ignored in the running volume computation.

	// Looking for reward factor for referrer 1.
	// His set has enough to notional volume to match tier 2
	// => Tier 1 reward factor.
	assert.Equal(t, num.DecimalFromFloat(0.01).String(), te.engine.RewardsFactorForParty(referrer1).String())

	// Looking for reward factor for referee 1.
	// With set, and member for long enough to match tier 2.
	// His set has enough notional volume to match tier 2.
	// => Tier 1 discount factor.
	assert.Equal(t, num.DecimalFromFloat(0.02).String(), te.engine.DiscountFactorForParty(referee1).String())

	// Looking for reward factor for referrer 2.
	// His set has enough notional volume to match tier 3.
	// => Tier 3 reward factor.
	assert.Equal(t, num.DecimalFromFloat(0.1).String(), te.engine.RewardsFactorForParty(referrer2).String())

	// Looking for reward factor for referee 2.
	// With set, and member for long enough to match tier 1.
	// His set has enough notional volume to match tier 3.
	// => Tier 1 discount factor.
	assert.Equal(t, num.DecimalFromFloat(0.002).String(), te.engine.DiscountFactorForParty(referee2).String())

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee2)).Return(num.UintFromUint64(10000)).Times(1)

	// When the epoch starts, the new program should start.
	expectReferralProgramEndedEvent(t, te)
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(1 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Program has ended, no more reward and discount factors.
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer1).String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.DiscountFactorForParty(referee1).String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer2).String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.DiscountFactorForParty(referee2).String())
}
