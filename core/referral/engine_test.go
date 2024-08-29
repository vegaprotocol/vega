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

package referral_test

import (
	"context"
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
	setID2 := newSetID(t)
	referrer := newPartyID(t)
	referrer2 := newPartyID(t)
	referee1 := newPartyID(t)

	require.NoError(t, te.engine.OnReferralProgramMinStakedVegaTokensUpdate(context.Background(), num.NewUint(100)))

	t.Run("querying for a non existing set return false", func(t *testing.T) {
		require.ErrorIs(t, referral.ErrUnknownSetID, te.engine.PartyOwnsReferralSet(referrer, setID))
	})

	t.Run("cannot join a non-existing set", func(t *testing.T) {
		err := te.engine.ApplyReferralCode(ctx, referee1, setID)
		assert.EqualError(t, err, referral.ErrUnknownReferralCode(setID).Error())
	})

	t.Run("can create a set for the first time", func(t *testing.T) {
		te.staking.EXPECT().GetAvailableBalance(string(referrer)).Return(num.NewUint(10001), nil).Times(1)
		te.broker.EXPECT().Send(gomock.Any()).Times(1)
		te.timeSvc.EXPECT().GetTimeNow().Times(1)

		assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer, setID))

		// check ownership query
		require.NoError(t, te.engine.PartyOwnsReferralSet(referrer, setID))
		require.Error(t, referral.ErrPartyDoesNotOwnReferralSet(referrer2), te.engine.PartyOwnsReferralSet(referrer2, setID))
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

	t.Run("cannot become a referee twice for the same set", func(t *testing.T) {
		assert.EqualError(t, te.engine.ApplyReferralCode(ctx, referee1, setID),
			referral.ErrIsAlreadyAReferee(referee1).Error(),
		)
	})

	t.Run("can create a second referrer", func(t *testing.T) {
		te.staking.EXPECT().GetAvailableBalance(string(referrer2)).Return(num.NewUint(10001), nil).Times(1)
		te.broker.EXPECT().Send(gomock.Any()).Times(1)
		te.timeSvc.EXPECT().GetTimeNow().Times(1)
		assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer2, setID2))
	})

	t.Run("cannot switch set if initial set still have an OK staking balance", func(t *testing.T) {
		te.staking.EXPECT().GetAvailableBalance(string(referrer)).Times(1).Return(num.NewUint(100), nil)
		assert.EqualError(t, te.engine.ApplyReferralCode(ctx, referee1, setID2),
			referral.ErrIsAlreadyAReferee(referee1).Error(),
		)
	})

	t.Run("can switch set if initial set have an insufficient staking balance", func(t *testing.T) {
		te.staking.EXPECT().GetAvailableBalance(string(referrer)).Times(1).Return(num.NewUint(99), nil)
		te.broker.EXPECT().Send(gomock.Any()).Times(1)
		te.timeSvc.EXPECT().GetTimeNow().Times(1)
		assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee1, setID2))
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
		StakingTiers:          []*types.StakingTier{},
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

func TestGettingRewardMultiplier(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)
	require.NoError(t, te.engine.OnReferralProgramMinStakedVegaTokensUpdate(context.Background(), num.NewUint(100)))
	require.NoError(t, te.engine.OnReferralProgramMaxReferralRewardProportionUpdate(context.Background(), num.MustDecimalFromString("0.5")))
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
				ReferralRewardFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.001),
					Infra:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
				ReferralDiscountFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.002),
					Infra:     num.DecimalFromFloat(0.002),
					Liquidity: num.DecimalFromFloat(0.002),
				},
			},
			{
				MinimumEpochs:                     num.UintFromUint64(1),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(1000),
				ReferralRewardFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.0005),
					Infra:     num.DecimalFromFloat(0.0005),
					Liquidity: num.DecimalFromFloat(0.0005),
				},
				ReferralDiscountFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.001),
					Infra:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
			},
			{
				MinimumEpochs:                     num.UintFromUint64(3),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(1000),
				ReferralRewardFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.0007),
					Infra:     num.DecimalFromFloat(0.0007),
					Liquidity: num.DecimalFromFloat(0.0007),
				},
				ReferralDiscountFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.002),
					Infra:     num.DecimalFromFloat(0.002),
					Liquidity: num.DecimalFromFloat(0.002),
				},
			},
			{
				MinimumEpochs:                     num.UintFromUint64(3),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(3000),
				ReferralRewardFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.01),
					Infra:     num.DecimalFromFloat(0.01),
					Liquidity: num.DecimalFromFloat(0.01),
				},
				ReferralDiscountFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.02),
					Infra:     num.DecimalFromFloat(0.02),
					Liquidity: num.DecimalFromFloat(0.02),
				},
			},
			{
				MinimumEpochs:                     num.UintFromUint64(4),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(4000),
				ReferralRewardFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.1),
					Infra:     num.DecimalFromFloat(0.1),
					Liquidity: num.DecimalFromFloat(0.1),
				},
				ReferralDiscountFactors: types.Factors{
					Maker:     num.DecimalFromFloat(0.2),
					Infra:     num.DecimalFromFloat(0.2),
					Liquidity: num.DecimalFromFloat(0.2),
				},
			},
		},
		StakingTiers: []*types.StakingTier{
			{
				MinimumStakedTokens:      num.NewUint(1000),
				ReferralRewardMultiplier: num.MustDecimalFromString("1.5"),
			},
			{
				MinimumStakedTokens:      num.NewUint(10000),
				ReferralRewardMultiplier: num.MustDecimalFromString("2"),
			},
			{
				MinimumStakedTokens:      num.NewUint(100000),
				ReferralRewardMultiplier: num.MustDecimalFromString("2.5"),
			},
		},
	}

	// Set the first program.
	te.engine.UpdateProgram(program1)

	setID1 := newSetID(t)
	referrer1 := newPartyID(t)
	referee1 := newPartyID(t)

	// When the epoch starts, the new program should start.
	expectReferralProgramStartedEvent(t, te)
	lastEpochStartTime := program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	te.broker.EXPECT().Send(gomock.Any()).Times(2)
	te.timeSvc.EXPECT().GetTimeNow().Return(time.Now()).Times(2)

	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Return(num.NewUint(10001), nil).Times(1)

	assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer1, setID1))
	assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee1, setID1))

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Return(num.NewUint(10001), nil).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(1500)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(2000)).Times(1)

	// When the epoch starts, the new program should start.
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	expectReferralSetStatsUpdatedEvent(t, te, 1)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	assert.Equal(t, "0.01", te.engine.RewardsFactorForParty(referee1).Infra.String())
	assert.Equal(t, "2", te.engine.RewardsMultiplierForParty(referee1).String())
	assert.Equal(t, "0.02", te.engine.RewardsFactorsMultiplierAppliedForParty(referee1).Infra.String())

	// When the epoch ends, the running volume for set members should be
	// computed.
	// Makes the set not eligible for rewards anymore by simulating staking balance
	// at 0.
	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Return(num.NewUint(0), nil).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(1500)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(2000)).Times(1)

	// When the epoch starts, the new program should start.
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-1 * time.Hour)
	expectReferralSetStatsUpdatedEvent(t, te, 1)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	assert.Equal(t, "0", te.engine.RewardsFactorForParty(referee1).Infra.String())
	assert.Equal(t, "1", te.engine.RewardsMultiplierForParty(referee1).String())
	assert.Equal(t, "0", te.engine.RewardsFactorsMultiplierAppliedForParty(referee1).Infra.String())
}

func TestGettingRewardAndDiscountFactors(t *testing.T) {
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomI64())

	te := newEngine(t)
	require.NoError(t, te.engine.OnReferralProgramMinStakedVegaTokensUpdate(context.Background(), num.NewUint(100)))

	setID1 := newSetID(t)
	setID2 := newSetID(t)
	referrer1 := newPartyID(t)
	referrer2 := newPartyID(t)
	referee1 := newPartyID(t)
	referee2 := newPartyID(t)
	referee3 := newPartyID(t)
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
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.002),
					Maker:     num.DecimalFromFloat(0.002),
					Liquidity: num.DecimalFromFloat(0.002),
				},
			}, {
				MinimumEpochs:                     num.UintFromUint64(3),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(3000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.01),
					Maker:     num.DecimalFromFloat(0.01),
					Liquidity: num.DecimalFromFloat(0.01),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.02),
					Maker:     num.DecimalFromFloat(0.02),
					Liquidity: num.DecimalFromFloat(0.02),
				},
			}, {
				MinimumEpochs:                     num.UintFromUint64(4),
				MinimumRunningNotionalTakerVolume: num.UintFromUint64(4000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.1),
					Maker:     num.DecimalFromFloat(0.1),
					Liquidity: num.DecimalFromFloat(0.1),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.2),
					Maker:     num.DecimalFromFloat(0.2),
					Liquidity: num.DecimalFromFloat(0.2),
				},
			},
		},
		StakingTiers: []*types.StakingTier{},
	}

	// Set the first program.
	te.engine.UpdateProgram(program1)

	// When the epoch starts, the new program should start.
	expectReferralProgramStartedEvent(t, te)
	lastEpochStartTime := program1.EndOfProgramTimestamp.Add(-2 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Setting up the referral sets.
	te.broker.EXPECT().Send(gomock.Any()).Times(4)
	te.timeSvc.EXPECT().GetTimeNow().Return(time.Now()).Times(4)

	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Times(1).Return(num.NewUint(100), nil)
	assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer1, setID1))

	te.staking.EXPECT().GetAvailableBalance(string(referrer2)).Times(1).Return(num.NewUint(100), nil)
	assert.NoError(t, te.engine.CreateReferralSet(ctx, referrer2, setID2))

	assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee1, setID1))
	assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee3, setID2))

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(800)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(100)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee3)).Return(num.UintFromUint64(0)).Times(1)
	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Times(1).Return(num.NewUint(100), nil)
	te.staking.EXPECT().GetAvailableBalance(string(referrer2)).Times(1).Return(num.NewUint(100), nil)

	expectReferralSetStatsUpdatedEvent(t, te, 2)
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-1*time.Hour - 50*time.Minute)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Looking for factors for party without a set.
	// => No reward nor discount factor.
	loneWolfParty := newPartyID(t)
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(loneWolfParty).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(loneWolfParty).Infra.String())

	// Looking for factors for referrer 1.
	// Factors only apply to referees' trades.
	// => No reward nor discount factor.
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer1).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referrer1).Infra.String())

	// Looking for factors for referrer 2.
	// Factors only apply to referees' trades.
	// => No reward nor discount factor.
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer2).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referrer2).Infra.String())

	// Looking for rewards factor for referee 1.
	// His set has not enough notional volume to reach tier 1.
	// He is not a member for long enough.
	// => No reward nor discount factor.
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referee1).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee1).Infra.String())

	// Looking for reward factor for referee 3.
	// His set has enough notional volume to reach tier 1.
	// He is not a member for long enough.
	// => Tier 1 reward factor.
	// => No discount factor.
	assert.Equal(t, "0.001", te.engine.RewardsFactorForParty(referee3).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee3).Infra.String())

	// Adding a new referee.
	te.broker.EXPECT().Send(gomock.Any()).Times(1)
	te.timeSvc.EXPECT().GetTimeNow().Return(time.Now()).Times(1)
	assert.NoError(t, te.engine.ApplyReferralCode(ctx, referee2, setID2))

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(1900)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(1000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(1000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee3)).Return(num.UintFromUint64(0)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee2)).Return(num.UintFromUint64(1000)).Times(1)
	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Times(1).Return(num.NewUint(100), nil)
	te.staking.EXPECT().GetAvailableBalance(string(referrer2)).Times(1).Return(num.NewUint(100), nil)

	// When the epoch starts, the new program should start.
	expectReferralSetStatsUpdatedEvent(t, te, 2)
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-1 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Looking for rewards factor for referee 1.
	// His set has enough notional volume to reach tier 2.
	// He is a member for long enough to reach tier 1.
	// => Tier 2 reward factor.
	// => Tier 1 discount factor.
	assert.Equal(t, "0.01", te.engine.RewardsFactorForParty(referee1).Infra.String())
	assert.Equal(t, "0.002", te.engine.ReferralDiscountFactorsForParty(referee1).Infra.String())

	// Looking for reward factor for referee 2.
	// His set has enough notional volume to reach tier 3.
	// He is not a member for long enough.
	// => Tier 3 reward factor.
	// => No discount factor.
	assert.Equal(t, "0.1", te.engine.RewardsFactorForParty(referee2).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee2).Infra.String())

	// Looking for reward factor for referee 3.
	// His set has enough notional volume to reach tier 3.
	// He is a member for long enough to reach tier 1.
	// => Tier 3 reward factor.
	// => Tier 1 discount factor.
	assert.Equal(t, "0.1", te.engine.RewardsFactorForParty(referee3).Infra.String())
	assert.Equal(t, "0.002", te.engine.ReferralDiscountFactorsForParty(referee3).Infra.String())

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(10)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(10)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(500)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee3)).Return(num.UintFromUint64(0)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee2)).Return(num.UintFromUint64(500)).Times(1)
	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Times(1).Return(num.NewUint(100), nil)
	te.staking.EXPECT().GetAvailableBalance(string(referrer2)).Times(1).Return(num.NewUint(100), nil)

	// When the epoch starts, the new program should start.
	expectReferralSetStatsUpdatedEvent(t, te, 2)
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-1 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Because the window length is set to 2, the first notional volumes are now
	// ignored in the running volume computation.

	// Looking for rewards factor for referee 1.
	// His set has enough notional volume to reach tier 1.
	// He is a member for long enough to reach tier 1.
	// => Tier 1 reward factor.
	// => Tier 1 discount factor.
	assert.Equal(t, "0.001", te.engine.RewardsFactorForParty(referee1).Infra.String())
	assert.Equal(t, "0.002", te.engine.ReferralDiscountFactorsForParty(referee1).Infra.String())

	// Looking for reward factor for referee 2.
	// His set has enough notional volume to reach tier 3.
	// He is a member for long enough to reach tier 1.
	// => Tier 2 reward factor.
	// => Tier 1 discount factor.
	assert.Equal(t, "0.01", te.engine.RewardsFactorForParty(referee2).Infra.String())
	assert.Equal(t, "0.002", te.engine.ReferralDiscountFactorsForParty(referee2).Infra.String())

	// Looking for reward factor for referee 3.
	// His set has enough notional volume to reach tier 2.
	// He is a member for long enough to reach tier 2.
	// => Tier 2 reward factor.
	// => Tier 2 discount factor.
	assert.Equal(t, "0.01", te.engine.RewardsFactorForParty(referee3).Infra.String())
	assert.Equal(t, "0.02", te.engine.ReferralDiscountFactorsForParty(referee3).Infra.String())

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee3)).Return(num.UintFromUint64(0)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee2)).Return(num.UintFromUint64(10000)).Times(1)
	// But, the sets are not eligible anymore.
	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Times(1).Return(num.NewUint(0), nil)
	te.staking.EXPECT().GetAvailableBalance(string(referrer2)).Times(1).Return(num.NewUint(0), nil)

	expectReferralSetStatsUpdatedEvent(t, te, 2)
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-45 * time.Minute)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// The sets are not eligible anymore, no more reward and discount factors.
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer1).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee1).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer2).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee3).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee2).Infra.String())

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee3)).Return(num.UintFromUint64(0)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee2)).Return(num.UintFromUint64(10000)).Times(1)
	// And the sets are eligible once again.
	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Times(1).Return(num.NewUint(100), nil)
	te.staking.EXPECT().GetAvailableBalance(string(referrer2)).Times(1).Return(num.NewUint(100), nil)

	expectReferralSetStatsUpdatedEvent(t, te, 2)
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(-30 * time.Minute)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Looking for rewards factor for referee 1.
	// His set has enough notional volume to reach tier 3.
	// He is a member for long enough to reach tier 3.
	// => Tier 3 reward factor.
	// => Tier 3 discount factor.
	assert.Equal(t, "0.1", te.engine.RewardsFactorForParty(referee1).Infra.String())
	assert.Equal(t, "0.2", te.engine.ReferralDiscountFactorsForParty(referee1).Infra.String())

	// Looking for reward factor for referee 2.
	// His set has enough notional volume to reach tier 3.
	// He is a member for long enough to reach tier 3.
	// => Tier 3 reward factor.
	// => Tier 3 discount factor.
	assert.Equal(t, "0.1", te.engine.RewardsFactorForParty(referee2).Infra.String())
	assert.Equal(t, "0.2", te.engine.ReferralDiscountFactorsForParty(referee2).Infra.String())

	// Looking for reward factor for referee 3.
	// His set has enough notional volume to reach tier 3.
	// He is a member for long enough to reach tier 3.
	// => Tier 3 reward factor.
	// => Tier 3 discount factor.
	assert.Equal(t, "0.1", te.engine.RewardsFactorForParty(referee3).Infra.String())
	assert.Equal(t, "0.2", te.engine.ReferralDiscountFactorsForParty(referee3).Infra.String())

	// When the epoch ends, the running volume for set members should be
	// computed.
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer1)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee1)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referrer2)).Return(num.UintFromUint64(10000)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee3)).Return(num.UintFromUint64(0)).Times(1)
	te.marketActivityTracker.EXPECT().NotionalTakerVolumeForParty(string(referee2)).Return(num.UintFromUint64(10000)).Times(1)
	te.staking.EXPECT().GetAvailableBalance(string(referrer1)).Times(1).Return(num.NewUint(100), nil)
	te.staking.EXPECT().GetAvailableBalance(string(referrer2)).Times(1).Return(num.NewUint(100), nil)

	// When the epoch starts, the current program should end.
	gomock.InOrder(
		expectReferralSetStatsUpdatedEvent(t, te, 2),
		expectReferralProgramEndedEvent(t, te),
	)
	lastEpochStartTime = program1.EndOfProgramTimestamp.Add(1 * time.Hour)
	nextEpoch(t, ctx, te, lastEpochStartTime)

	// Program has ended, no more reward and discount factors.
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer1).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee1).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.RewardsFactorForParty(referrer2).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee3).Infra.String())
	assert.Equal(t, num.DecimalZero().String(), te.engine.ReferralDiscountFactorsForParty(referee2).Infra.String())
}
