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

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// TestProposalForNetParamInvalidOnProposal verifies that an invalid value with respect to current values (i.e. crossing with the other current value) fails at submission time.
func TestProposalForNetParamInvalidOnProposal(t *testing.T) {
	now := time.Now()

	eng := getTestEngine(t, now)

	party := eng.newValidParty("a-valid-party", 1)

	// propose a min time that is greater than the max time, this should be invalid upon proposal and should be immediately rejected
	p1 := eng.newProposalForNetParam(party.Id, netparams.MarketAuctionMinimumDuration, "169h", now.Add(48*time.Hour))

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	_, err := eng.submitProposal(t, p1)

	// then
	require.Equal(t, "unable to validate market.auction.minimumDuration: expect <= 24h0m0s got 169h0m0s, expect < 168h0m0s (market.auction.maximumDuration) got 169h0m0s", err.Error())
}

// TestProposalForNetParamCrossingOnUpdate submits two crossing proposals with the same enactment time. In this case the proposals pass the initial validation but also
// pass the enactment validation and only fail at the last step of updating the param. The proposal will be marked as faild and only one of the values will go through.
func TestProposalForNetParamCrossingOnUpdate(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	party1 := eng.newValidParty("a-valid-party", 1)
	party2 := eng.newValidParty("another-valid-party", 1)

	now := eng.tsvc.GetTimeNow()
	date1 := now.Add(5 * 24 * time.Hour)

	// lower the max first, this should go through
	p1 := eng.newProposalForNetParam(party1.Id, netparams.MarketAuctionMaximumDuration, "10h", date1)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	_, err := eng.submitProposal(t, p1)
	require.NoError(t, err)

	// now increase the min above the max, this should be valid at the time of proposal
	p2 := eng.newProposalForNetParam(party2.Id, netparams.MarketAuctionMinimumDuration, "11h", date1)

	_, err = eng.submitProposal(t, p2)
	require.NoError(t, err)

	// now get them enacted in order, the expected result is that the one enacted first gets through and the latter gets rejected
	// i.e. min = 5s, max = 10h

	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(2).Return(num.NewUint(9))
	eng.ensureTokenBalanceForParty(t, party1.Id, 1)
	eng.ensureTokenBalanceForParty(t, party2.Id, 1)

	// vote for first proposal
	voter1 := vgrand.RandomStr(5)
	eng.ensureTokenBalanceForParty(t, voter1, 7)
	err = eng.addYesVote(t, voter1, p1.ID)
	require.NoError(t, err)

	// vote for second proposal
	voter2 := vgrand.RandomStr(5)
	eng.ensureTokenBalanceForParty(t, voter2, 7)
	err = eng.addYesVote(t, voter2, p2.ID)
	require.NoError(t, err)

	// given
	afterEnactment := time.Unix(p1.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// time to enact, expect both of them to be enacted because at the time of check none has updated yet
	enacted, _ := eng.OnTick(context.Background(), afterEnactment)
	require.Equal(t, 2, len(enacted))

	// update enacted - first is expected to pass
	err = eng.netp.Update(context.Background(), enacted[0].UpdateNetworkParameter().Key, enacted[0].UpdateNetworkParameter().Value)
	require.NoError(t, err)

	// second expected to fail the validation as the first has already been updated
	err = eng.netp.Update(context.Background(), enacted[1].UpdateNetworkParameter().Key, enacted[1].UpdateNetworkParameter().Value)
	require.Equal(t, "unable to update market.auction.minimumDuration: expect < 10h0m0s (market.auction.maximumDuration) got 11h0m0s", err.Error())

	max, err := eng.netp.Get(netparams.MarketAuctionMaximumDuration)
	require.Equal(t, "10h", max)
	require.NoError(t, err)

	min, err := eng.netp.Get(netparams.MarketAuctionMinimumDuration)
	require.Equal(t, "30m0s", min)
	require.NoError(t, err)
}

// TestProposalForNetParamCrossingAtEnactment submits two crossing proposals such that one goes in before the other. In this case the first proposal passes and will get updated while
// the second proposal will fail pre-enactment.
func TestProposalForNetParamCrossingAtEnactment(t *testing.T) {
	eng := getTestEngine(t, time.Now())

	party1 := eng.newValidParty("a-valid-party", 1)
	party2 := eng.newValidParty("another-valid-party", 1)

	now := eng.tsvc.GetTimeNow()

	date1 := now.Add(5 * 24 * time.Hour)
	date2 := date1.Add(time.Hour)

	// lower the max first, this should go through
	p1 := eng.newProposalForNetParam(party1.Id, netparams.MarketAuctionMaximumDuration, "10h", date1)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	_, err := eng.submitProposal(t, p1)
	require.NoError(t, err)

	// now increase the min above the max, this should be valid at the time of proposal
	p2 := eng.newProposalForNetParam(party2.Id, netparams.MarketAuctionMinimumDuration, "11h", date2)

	_, err = eng.submitProposal(t, p2)
	require.NoError(t, err)

	// now get them enacted in order, the expected result is that the one enacted first gets through and the latter gets rejected
	// i.e. min = 5s, max = 10h

	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(2).Return(num.NewUint(9))
	eng.ensureTokenBalanceForParty(t, party1.Id, 1)
	eng.ensureTokenBalanceForParty(t, party2.Id, 1)

	// vote for first proposal
	voter1 := vgrand.RandomStr(5)
	eng.ensureTokenBalanceForParty(t, voter1, 7)
	err = eng.addYesVote(t, voter1, p1.ID)
	require.NoError(t, err)

	// vote for second proposal
	voter2 := vgrand.RandomStr(5)
	eng.ensureTokenBalanceForParty(t, voter2, 7)
	err = eng.addYesVote(t, voter2, p2.ID)
	require.NoError(t, err)

	// move time to after the enactment time of the first proposal
	afterEnactment1 := time.Unix(p1.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// time to enact, expect both of them to be enacted because at the time of check none has updated yet
	enacted, _ := eng.OnTick(context.Background(), afterEnactment1)
	require.Equal(t, 1, len(enacted))

	// update enacted - first is expected to pass
	err = eng.netp.Update(context.Background(), enacted[0].UpdateNetworkParameter().Key, enacted[0].UpdateNetworkParameter().Value)
	require.NoError(t, err)

	// move time to after the enactment time of the second proposal
	afterEnactment2 := time.Unix(p2.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// time to enact, expect both of them to be enacted because at the time of check none has updated yet
	enacted, _ = eng.OnTick(context.Background(), afterEnactment2)
	require.Equal(t, 0, len(enacted))

	max, err := eng.netp.Get(netparams.MarketAuctionMaximumDuration)
	require.Equal(t, "10h", max)
	require.NoError(t, err)

	min, err := eng.netp.Get(netparams.MarketAuctionMinimumDuration)
	require.Equal(t, "30m0s", min)
	require.NoError(t, err)

	eng.OnTick(context.Background(), afterEnactment1.Add(1*time.Second))
}

func (e *tstEngine) newProposalForNetParam(partyID, key, value string, now time.Time) types.Proposal {
	id := e.newProposalID()
	return types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     partyID,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newNetParamTerms(key, value),
		},
		Rationale: &types.ProposalRationale{
			Description: "some description",
		},
	}
}
