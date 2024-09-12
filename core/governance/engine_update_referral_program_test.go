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

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestProposalForUpdateReferralProgram(t *testing.T) {
	t.Run("Submitting a proposal for referral program update succeeds", testSubmittingProposalForReferralProgramUpdateSucceeds)
	t.Run("Submitting a proposal for referral program update with too many tiers fails", testSubmittingProposalForReferralProgramUpdateWithTooManyTiersFails)
	t.Run("Submitting a proposal for referral program update with too high reward factor fails", testSubmittingProposalForReferralProgramUpdateWithTooHighRewardFactorFails)
	t.Run("Submitting a proposal for referral program update with too high discount factor fails", testSubmittingProposalForReferralProgramUpdateWithTooHighDiscountFactorFails)
	t.Run("Submitting a proposal for referral program that ends before it enacted fails", testSubmittingProposalForReferralProgramUpdateEndsBeforeEnactsFails)
}

func testSubmittingProposalForReferralProgramUpdateSucceeds(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinProposerBalance, "1000")

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, &types.ReferralProgramChanges{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.NewUint(1),
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
			}, {
				MinimumEpochs:                     num.NewUint(7),
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.005),
					Maker:     num.DecimalFromFloat(0.005),
					Liquidity: num.DecimalFromFloat(0.005),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.005),
					Maker:     num.DecimalFromFloat(0.005),
					Liquidity: num.DecimalFromFloat(0.005),
				},
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

// testSubmittingProposalForReferralProgramUpdateWithTooManyTiersFails covers 0095-HVMR-002.
func testSubmittingProposalForReferralProgramUpdateWithTooManyTiersFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinProposerBalance, "1000")

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "1")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "1"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, &types.ReferralProgramChanges{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.NewUint(1),
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
			}, {
				MinimumEpochs:                     num.NewUint(7),
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.005),
					Maker:     num.DecimalFromFloat(0.005),
					Liquidity: num.DecimalFromFloat(0.005),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.005),
					Maker:     num.DecimalFromFloat(0.005),
					Liquidity: num.DecimalFromFloat(0.005),
				},
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
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinProposerBalance, "1000")

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, &types.ReferralProgramChanges{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.NewUint(1),
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
			}, {
				MinimumEpochs:                     num.NewUint(7),
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.015),
					Maker:     num.DecimalFromFloat(0.015),
					Liquidity: num.DecimalFromFloat(0.015),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.005),
					Maker:     num.DecimalFromFloat(0.005),
					Liquidity: num.DecimalFromFloat(0.005),
				},
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
		"tier 2 defines a referral reward infrastructure factor higher than the maximum allowed by the network parameter \"referralProgram.maxReferralRewardFactor\": maximum is 0.01, but got 0.015",
	)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForReferralProgramUpdateWithTooHighDiscountFactorFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinProposerBalance, "1000")

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, &types.ReferralProgramChanges{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		BenefitTiers: []*types.BenefitTier{
			{
				MinimumEpochs:                     num.NewUint(1),
				MinimumRunningNotionalTakerVolume: num.NewUint(10000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.001),
					Maker:     num.DecimalFromFloat(0.001),
					Liquidity: num.DecimalFromFloat(0.001),
				},
			}, {
				MinimumEpochs:                     num.NewUint(7),
				MinimumRunningNotionalTakerVolume: num.NewUint(20000),
				ReferralRewardFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.01),
					Maker:     num.DecimalFromFloat(0.01),
					Liquidity: num.DecimalFromFloat(0.01),
				},
				ReferralDiscountFactors: types.Factors{
					Infra:     num.DecimalFromFloat(0.015),
					Maker:     num.DecimalFromFloat(0.015),
					Liquidity: num.DecimalFromFloat(0.015),
				},
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
		"tier 2 defines a referral discount infrastructure factor higher than the maximum allowed by the network parameter \"referralProgram.maxReferralDiscountFactor\": maximum is 0.01, but got 0.015",
	)
	require.Nil(t, toSubmit)
}

func testSubmittingProposalForReferralProgramUpdateEndsBeforeEnactsFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalReferralProgramMinProposerBalance, "1000")

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralTiers, "2"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralRewardFactor, "0.010"))

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.ReferralProgramMaxReferralDiscountFactor, "0.010"))

	// given
	proposer := vgrand.RandomStr(5)
	rp := &types.ProposalTermsUpdateReferralProgram{
		UpdateReferralProgram: &types.UpdateReferralProgram{
			Changes: &types.ReferralProgramChanges{
				EndOfProgramTimestamp: time.Time{}, // we will set this later
				WindowLength:          15,
				BenefitTiers: []*types.BenefitTier{
					{
						MinimumEpochs:                     num.NewUint(1),
						MinimumRunningNotionalTakerVolume: num.NewUint(10000),
						ReferralRewardFactors: types.Factors{
							Infra:     num.DecimalFromFloat(0.001),
							Maker:     num.DecimalFromFloat(0.001),
							Liquidity: num.DecimalFromFloat(0.001),
						},
						ReferralDiscountFactors: types.Factors{
							Infra:     num.DecimalFromFloat(0.001),
							Maker:     num.DecimalFromFloat(0.001),
							Liquidity: num.DecimalFromFloat(0.001),
						},
					}, {
						MinimumEpochs:                     num.NewUint(7),
						MinimumRunningNotionalTakerVolume: num.NewUint(20000),
						ReferralRewardFactors: types.Factors{
							Infra:     num.DecimalFromFloat(0.01),
							Maker:     num.DecimalFromFloat(0.01),
							Liquidity: num.DecimalFromFloat(0.01),
						},
						ReferralDiscountFactors: types.Factors{
							Infra:     num.DecimalFromFloat(0.015),
							Maker:     num.DecimalFromFloat(0.015),
							Liquidity: num.DecimalFromFloat(0.015),
						},
					},
				},
			},
		},
	}
	proposal := eng.newProposalForReferralProgramUpdate(proposer, now, rp.UpdateReferralProgram.Changes)
	rp.UpdateReferralProgram.Changes.EndOfProgramTimestamp = time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(-time.Second) // set to end before enacted
	proposal.Terms.Change = rp

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidReferralProgram)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.EqualError(t,
		err,
		"the proposal must be enacted before the referral program ends",
	)
	require.Nil(t, toSubmit)
}
