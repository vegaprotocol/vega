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

func TestProposalForUpdateDiscountVolumeProgram(t *testing.T) {
	t.Run("Submitting a proposal for referral program update succeeds", testSubmittingProposalForVolumeDiscountProgramUpdateSucceeds)
	t.Run("Submitting a proposal for referral program update with too many tiers fails", testSubmittingProposalForVolumeDiscountProgramUpdateWithTooManyTiersFails)
	t.Run("Submitting a proposal for referral program update with too high discount factor fails", testSubmittingProposalForVolumeDiscountProgramUpdateWithTooHighDiscountFactorFails)
}

func testSubmittingProposalForVolumeDiscountProgramUpdateSucceeds(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.VolumeDiscountProgramMaxBenefitTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.VolumeDiscountProgramMaxBenefitTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForVolumeDiscountProgramUpdate(proposer, now, &types.VolumeDiscountProgram{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				VolumeDiscountFactor:              num.DecimalFromFloat(0.001),
			}, {
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				VolumeDiscountFactor:              num.DecimalFromFloat(0.005),
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

func testSubmittingProposalForVolumeDiscountProgramUpdateWithTooManyTiersFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.VolumeDiscountProgramMaxBenefitTiers, "1")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.VolumeDiscountProgramMaxBenefitTiers, "1"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForVolumeDiscountProgramUpdate(proposer, now, &types.VolumeDiscountProgram{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				VolumeDiscountFactor:              num.DecimalFromFloat(0.001),
			}, {
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				VolumeDiscountFactor:              num.DecimalFromFloat(0.005),
			},
		},
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidVolumeDiscountProgram)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForVolumeDiscountProgramUpdateWithTooHighDiscountFactorFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.VolumeDiscountProgramMaxBenefitTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.VolumeDiscountProgramMaxBenefitTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.VolumeDiscountProgramMaxVolumeDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForVolumeDiscountProgramUpdate(proposer, now, &types.VolumeDiscountProgram{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		VolumeBenefitTiers: []*types.VolumeBenefitTier{
			{
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				VolumeDiscountFactor:              num.DecimalFromFloat(0.001),
			}, {
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				VolumeDiscountFactor:              num.DecimalFromFloat(0.015),
			},
		},
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidVolumeDiscountProgram)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.EqualError(t,
		err,
		"tier 2 defines a volume discount factor higher than the maximum allowed by the network parameter \"volumeDiscountProgram.maxVolumeDiscountFactor\": maximum is 0.01, but got 0.015",
	)
	require.Nil(t, toSubmit)
}
