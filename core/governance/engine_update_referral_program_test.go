package governance_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"github.com/stretchr/testify/require"
)

func TestProposalForUpdateReferralProgram(t *testing.T) {
	t.Run("Submitting a proposal for referral program update succeeds", testSubmittingProposalForReferralProgramUpdateSucceeds)
	t.Run("Submitting a proposal for referral program update with too many tiers fails", testSubmittingProposalForReferralProgramUpdateWithTooManyTiersFails)
	t.Run("Submitting a proposal for referral program update with too high reward factor fails", testSubmittingProposalForReferralProgramUpdateWithTooHighRewardFactorFails)
	t.Run("Submitting a proposal for referral program update with too high discount factor fails", testSubmittingProposalForReferralProgramUpdateWithTooHighDiscountFactorFails)
}

func testSubmittingProposalForReferralProgramUpdateSucceeds(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, &types.ReferralProgram{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.NewUint(1),
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.001),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.001),
			}, {
				MinimumEpochs:                     num.NewUint(7),
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.005),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.005),
			},
		},
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
}

func testSubmittingProposalForReferralProgramUpdateWithTooManyTiersFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "1")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "1"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, &types.ReferralProgram{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.NewUint(1),
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.001),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.001),
			}, {
				MinimumEpochs:                     num.NewUint(7),
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.005),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.005),
			},
		},
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidReferralProgram)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForReferralProgramUpdateWithTooHighRewardFactorFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, &types.ReferralProgram{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.NewUint(1),
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.001),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.001),
			}, {
				MinimumEpochs:                     num.NewUint(7),
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.015),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.005),
			},
		},
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidReferralProgram)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.EqualError(t,
		err,
		"tier 2 defines a referral reward factor higher than the maximum allowed by the network parameter \"referralProgram.maxReferralRewardFactor\": maximum is 0.01, but got 0.015",
	)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForReferralProgramUpdateWithTooHighDiscountFactorFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, &types.ReferralProgram{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.NewUint(1),
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.001),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.001),
			}, {
				MinimumEpochs:                     num.NewUint(7),
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				ReferralRewardFactor:              num.DecimalFromFloat(0.010),
				ReferralDiscountFactor:            num.DecimalFromFloat(0.015),
			},
		},
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidReferralProgram)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.EqualError(t,
		err,
		"tier 2 defines a referral discount factor higher than the maximum allowed by the network parameter \"referralProgram.maxReferralDiscountFactor\": maximum is 0.01, but got 0.015",
	)
	require.Nil(t, toSubmit)
}
