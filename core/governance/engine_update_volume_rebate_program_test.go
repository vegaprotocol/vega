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

func TestProposalForUpdateVolumeRebateProgram(t *testing.T) {
	t.Run("Submitting a proposal for referral program update succeeds", testSubmittingProposalForVolumeRebateProgramUpdateSucceeds)
	t.Run("Submitting a proposal for referral program update with too many tiers fails", testSubmittingProposalForVolumeRebateProgramUpdateWithTooManyTiersFails)
}

func testSubmittingProposalForVolumeRebateProgramUpdateSucceeds(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalVolumeRebateProgramMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalVolumeRebateProgramMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalVolumeRebateProgramMinProposerBalance, "1000")

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.VolumeRebateProgramMaxBenefitTiers, "2")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.VolumeRebateProgramMaxBenefitTiers, "2"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForVolumeRebateProgramUpdate(proposer, now, &types.VolumeRebateProgramChanges{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		VolumeRebateBenefitTiers: []*types.VolumeRebateBenefitTier{
			{
				MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.1),
				AdditionalMakerRebate:           num.DecimalFromFloat(0.00001),
			}, {
				MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.2),
				AdditionalMakerRebate:           num.DecimalFromFloat(0.00002),
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

func testSubmittingProposalForVolumeRebateProgramUpdateWithTooManyTiersFails(t *testing.T) {
	now := time.Now()
	ctx := vgtest.VegaContext(vgrand.RandomStr(5), vgtest.RandomPositiveI64())
	eng := getTestEngine(t, now)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(3)
	eng.netp.Update(ctx, netparams.GovernanceProposalVolumeRebateProgramMinClose, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalVolumeRebateProgramMinEnact, "48h")
	eng.netp.Update(ctx, netparams.GovernanceProposalVolumeRebateProgramMinProposerBalance, "1000")

	eng.broker.EXPECT().Send(events.NewNetworkParameterEvent(ctx, netparams.VolumeRebateProgramMaxBenefitTiers, "1")).Times(1)
	require.NoError(t, eng.netp.Update(ctx, netparams.VolumeRebateProgramMaxBenefitTiers, "1"))

	// given
	proposer := vgrand.RandomStr(5)
	proposal := eng.newProposalForVolumeRebateProgramUpdate(proposer, now, &types.VolumeRebateProgramChanges{
		EndOfProgramTimestamp: now.Add(4 * 48 * time.Hour),
		WindowLength:          15,
		VolumeRebateBenefitTiers: []*types.VolumeRebateBenefitTier{
			{
				MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.1),
				AdditionalMakerRebate:           num.DecimalFromFloat(0.001),
			}, {
				MinimumPartyMakerVolumeFraction: num.DecimalFromFloat(0.2),
				AdditionalMakerRebate:           num.DecimalFromFloat(0.002),
			},
		},
	})

	// setup
	eng.ensureTokenBalanceForParty(t, proposer, 1000)

	// expect
	eng.expectRejectedProposalEvent(t, proposer, proposal.ID, types.ProposalErrorInvalidVolumeRebateProgram)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.Error(t, err)
	require.Nil(t, toSubmit)
}
