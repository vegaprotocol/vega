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
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	checkpointpb "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpoint(t *testing.T) {
	t.Run("Basic test -> get checkpoints at various points in time, load checkpoint", testCheckpointSuccess)
	t.Run("Loading with missing rationale shouldn't be a problem", testCheckpointLoadingWithMissingRationaleShouldNotBeProblem)
}

func testCheckpointSuccess(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	voter2 := eng.newValidPartyTimes("voter2", 1, 0)

	now := eng.tsvc.GetTimeNow()
	termTimeAfterEnact := now.Add(4 * 48 * time.Hour).Add(1 * time.Second)
	filter, binding := produceTimeTriggeredOracleSpec(termTimeAfterEnact)
	proposal := eng.newProposalForNewMarket(proposer.Id, eng.tsvc.GetTimeNow(), filter, binding)
	ctx := context.Background()

	// setup
	eng.ensureStakingAssetTotalSupply(t, 9)
	eng.ensureAllAssetEnabled(t)
	eng.expectOpenProposalEvent(t, proposer.Id, proposal.ID)

	// when
	_, err := eng.submitProposal(t, proposal)

	// then
	require.NoError(t, err)

	// setup
	eng.expectVoteEvent(t, voter1.Id, proposal.ID)

	// then
	err = eng.addYesVote(t, voter1.Id, proposal.ID)

	// then
	require.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.expectPassedProposalEvent(t, proposal.ID)
	eng.expectTotalGovernanceTokenFromVoteEvents(t, "1", "7")

	// checkpoint should be empty at this point
	data, err := eng.Checkpoint()
	require.NoError(t, err)
	require.Empty(t, data)

	eng.expectGetMarketState(t, proposal.ID)
	// when
	eng.OnTick(ctx, afterClosing)

	// the proposal should already be in the snapshot
	data, err = eng.Checkpoint()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// when
	err = eng.addNoVote(t, voter2.Id, proposal.ID)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotOpenForVotes.Error())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	toBeEnacted, closed := eng.OnTick(ctx, afterEnactment)

	// then
	require.NotEmpty(t, toBeEnacted)
	require.Empty(t, closed)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// Now take the checkpoint
	data, err = eng.Checkpoint()
	require.NoError(t, err)
	require.NotEmpty(t, data)

	eng2 := getTestEngine(t)
	defer eng2.ctrl.Finish()

	eng2.broker.EXPECT().SendBatch(gomock.Any()).Times(1)

	eng2.assets.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(id string) (*assets.Asset, error) {
		ret := assets.NewAsset(builtin.New(id, &types.AssetDetails{}))
		return ret, nil
	})
	eng2.assets.EXPECT().IsEnabled(gomock.Any()).Return(true).AnyTimes()
	eng2.markets.EXPECT().RestoreMarketWithLiquidityProvision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	eng2.markets.EXPECT().StartOpeningAuction(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	// Load checkpoint
	require.NoError(t, eng2.Load(ctx, data))

	// check that it matches what we took before in eng1
	cp2, err := eng2.Checkpoint()
	require.NoError(t, err)
	require.True(t, bytes.Equal(cp2, data))

	data = append(data, []byte("foo")...)
	require.Error(t, eng2.Load(ctx, data))
}

func testCheckpointLoadingWithMissingRationaleShouldNotBeProblem(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := eng.tsvc.GetTimeNow()
	// given
	proposalWithoutRationale := &vegapb.Proposal{
		Id:        vgrand.RandomStr(5),
		Reference: vgrand.RandomStr(5),
		PartyId:   vgrand.RandomStr(5),
		State:     types.ProposalStateEnacted,
		Timestamp: 123456789,
		Terms: &vegapb.ProposalTerms{
			ClosingTimestamp:    now.Add(10 * time.Minute).Unix(),
			EnactmentTimestamp:  now.Add(30 * time.Minute).Unix(),
			ValidationTimestamp: 0,
			Change:              &vegapb.ProposalTerms_NewFreeform{},
		},
		Reason:       0,
		ErrorDetails: "",
		Rationale:    nil,
	}
	data := marshalProposal(t, proposalWithoutRationale)

	// setup
	eng.expectRestoredProposals(t, []string{proposalWithoutRationale.Id})

	// when
	err := eng.Load(context.Background(), data)

	// then
	require.NoError(t, err)
}

func marshalProposal(t *testing.T, proposal *vegapb.Proposal) []byte {
	t.Helper()
	proposals := &checkpointpb.Proposals{
		Proposals: []*vegapb.Proposal{proposal},
	}

	data, err := proto.Marshal(proposals)
	if err != nil {
		t.Fatalf("couldn't marshal proposals for tests: %v", err)
	}

	return data
}
