// Copyright (c) 2022 Gobalsky Labs Limited
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

package governance_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/governance"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFreeformProposal(t *testing.T) {
	t.Run("Submitting a freeform proposal succeeds", testSubmittingFreeformProposalSucceeds)
	t.Run("Freeform proposal does not wait for enactment timestamp", testFreeformProposalDoesNotWaitToEnact)
}

func testSubmittingFreeformProposalSucceeds(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newFreeformProposal(party.Id, eng.tsvc.GetTimeNow().Add(48*time.Hour))

	// setup
	eng.expectOpenProposalEvent(t, party.Id, proposal.ID)

	// when
	toSubmit, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)
	require.NotNil(t, toSubmit)
}

func testFreeformProposalDoesNotWaitToEnact(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	// when
	proposer := vgrand.RandomStr(5)
	proposal := eng.newFreeformProposal(proposer, eng.tsvc.GetTimeNow().Add(48*time.Hour))

	// setup
	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)
	eng.ensureTokenBalanceForParty(t, proposer, 1)

	// expect
	eng.expectOpenProposalEvent(t, proposer, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// given
	voter1 := vgrand.RandomStr(5)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectVoteEvent(t, voter1, proposal.ID)

	// then
	err = eng.addYesVote(t, voter1, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.ensureTokenBalanceForParty(t, voter1, 7)

	// expect
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")
	eng.expectGetMarketState(t, proposal.ID)

	// when the proposal is closed, it is enacted immediately
	toBeEnacted, _ := eng.OnTick(context.Background(), afterClosing)

	// then
	require.Len(t, toBeEnacted, 1)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// given
	voter2 := vgrand.RandomStr(5)

	// when
	err = eng.addNoVote(t, voter2, proposal.ID)

	// then
	require.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalDoesNotExist.Error())
}
