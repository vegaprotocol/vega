package governance_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	oraclesv1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/builtin"
	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/validators"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errNoBalanceForParty = errors.New("no balance for party")

type tstEngine struct {
	*governance.Engine
	ctrl            *gomock.Controller
	accounts        *mocks.MockStakingAccounts
	broker          *bmock.MockBroker
	witness         *mocks.MockWitness
	assets          *mocks.MockAssets
	netp            *netparams.Store
	proposalCounter uint // to streamline proposal generation
}

func TestSubmitProposals(t *testing.T) {
	t.Run("Submitting a valid proposal succeeds", testSubmittingValidProposalSucceeds)
	t.Run("Submitting duplicated proposal fails", testSubmittingDuplicatedProposalFails)
	t.Run("Submitting a proposal with bad closing time fails", testSubmittingProposalWithBadClosingTimeFails)
	t.Run("Submitting a proposal with bad enactment time fails", testSubmittingProposalWithBadEnactmentTimeFails)
	t.Run("Submitting a proposal with closing time before validation time fails", testSubmittingProposalWithClosingTimeBeforeValidationTimeFails)
	t.Run("Submitting a proposal with bad risk parameter fail", testSubmittingProposalWithBadRiskParameter)
	t.Run("Submitting a proposal with non-existing account fails", testSubmittingProposalWithNonexistingAccountFails)
	t.Run("Submitting a proposal without enough stake fails", testSubmittingProposalWithoutEnoughStakeFails)
	t.Run("Submit valid market proposal return a market to submit", testNewValidMarketProposalReturnsAMarketToSubmit)
	t.Run("Can reject proposal", testCanRejectProposal)

	t.Run("Submitting a valid vote on existing proposal succeeds", testSubmittingValidVoteOnExistingProposalSucceeds)
	t.Run("Submitting a vote on non-existing proposal fails", testSubmittingVoteOnNonexistingProposalFails)
	t.Run("Submitting a vote with non-existing account fails", testSubmittingVoteWithNonexistingAccountFails)
	t.Run("Submitting a vote without token fails", testSubmittingVoteWithoutTokenFails)
	t.Run("Submitting a majority of yes vote makes the proposal passed", testSubmittingMajorityOfYesVoteMakesProposalPassed)
	t.Run("Submitting a majority of no vote makes the proposal declined", testSubmittingMajorityOfNoVoteMakesProposalDeclined)
	t.Run("Submitting a majority of yes votes below participation threshold marks proposal as declined", testSubmittingMajorityOfInsuccifientParticipationMakesProposalDeclined)
	t.Run("Test multiple proposal lifecycle", testMultipleProposalsLifecycle)
	t.Run("Test withdrawing asset before proposal end", testSubmittingVoteAndWithdrawingFundsDeclined)
	t.Run("Validate market proposal commitment", testValidateProposalCommitment)

	t.Run("Valid freeform proposal", testValidFreeformProposal)
	t.Run("Invalid freeform proposal", testInvalidFreeformProposal)
	t.Run("Freeform proposal does not wait for enactment timestamp", testFreeformProposalDoesNotWaitToEnact)

	t.Run("Can vote during validation period - proposal passed", testSubmittingMajorityOfYesVoteDuringValidationMakesProposalPassed)

	t.Run("test hash", testGovernanceHash)
	t.Run("key rotation test", testKeyRotated)
}

func testKeyRotated(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	proposer := eng.newValidParty("proposer", 1)
	voter := eng.newValidParty("voter", 1)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)
	assert.NoError(t, err)
	eng.expectSendVoteEvent(t, voter, proposal)
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter.Id)
	assert.NoError(t, err)

	eng.expectSendVoteEvent(t, &proto.Party{Id: "newVoter"}, proposal)
	eng.ValidatorKeyChanged(context.Background(), "voter", "newVoter")
}

func testValidateProposalCommitment(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := eng.newValidPartyTimes("a-valid-party", 1, 8)

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.expectAnyAsset()

	now := time.Now()
	prop := eng.newOpenProposal(party.Id, now)

	// first we test with no commitment
	prop.Terms.GetNewMarket().LiquidityCommitment = nil
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "market proposal is missing liquidity commitment")

	// Then no amount
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.CommitmentAmount = num.Zero()
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal commitment amount is 0 or missing")

	// Then empty fees
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Fee = num.DecimalZero()
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid liquidity provision fee")

	// Then negative fees
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Fee = num.DecimalFromFloat(-1)
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid liquidity provision fee")

	// Then empty shapes
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Buys = nil
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty SIDE_BUY shape")

	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Sells = nil
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty SIDE_SELL shape")

	// Then invalid shapes
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Buys[0].Reference = proto.PeggedReference_PEGGED_REFERENCE_BEST_ASK
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order in buy side shape with best ask price reference")

	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Sells[0].Reference = proto.PeggedReference_PEGGED_REFERENCE_BEST_BID
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order in sell side shape with best bid price reference")
}

func testCanRejectProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// to check min required level
	eng.expectAnyAssetTimes(2)

	// once proposal is validated, it is added to the buffer
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newOpenProposal(party.Id, time.Now())
	eng.expectSendOpenProposalEvent(t, party, proposal)

	toSubmit, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)
	assert.NoError(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// now we try to reject to reject
	err = eng.RejectProposal(context.Background(), toSubmit.Proposal(), proto.ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET, errors.New("failure"))
	assert.NoError(t, err)

	// just one more to make sure it was rejected...
	err = eng.RejectProposal(context.Background(), toSubmit.Proposal(), proto.ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET, errors.New("failure"))
	assert.EqualError(t, err, governance.ErrProposalDoesNotExists.Error())
}

func testNewValidMarketProposalReturnsAMarketToSubmit(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newOpenProposal(party.Id, time.Now())

	// setup
	eng.expectAnyAssetTimes(2)
	eng.expectSendOpenProposalEvent(t, party, proposal)

	// when
	toSubmit, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)

	// then
	assert.NoError(t, err)
	assert.NotNil(t, toSubmit)
	assert.True(t, toSubmit.IsNewMarket())
	assert.NotNil(t, toSubmit.NewMarket().Market())
	assert.NotNil(t, toSubmit.NewMarket().LiquidityProvisionSubmission())
}

func testSubmittingValidProposalSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newOpenProposal(party.Id, time.Now())

	// setup
	eng.expectAnyAssetTimes(2)
	eng.expectSendOpenProposalEvent(t, party, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)

	// then
	assert.NoError(t, err)
}

func testSubmittingDuplicatedProposalFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("valid-party", 1000)
	original := eng.newOpenProposal(party.Id, time.Now())

	// setup
	eng.expectAnyAssetTimes(2)
	eng.expectSendOpenProposalEvent(t, party, original)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&original), original.ID, party.Id)

	// then
	assert.NoError(t, err)

	// given
	aCopy := original
	aCopy.Reference = "this-is-a-copy"

	// when
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&aCopy), aCopy.ID, party.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())

	// given
	aCopy = original
	aCopy.State = proto.Proposal_STATE_PASSED

	// when
	_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&aCopy), aCopy.ID, party.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error(), "reject attempt to change state indirectly")
}

func testSubmittingProposalWithNonexistingAccountFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	noAccountPartyID := "party"
	proposal := eng.newOpenProposal(noAccountPartyID, time.Now())

	// setup
	eng.expectAnyAsset()
	eng.expectNoAccountForParty(noAccountPartyID)
	eng.expectSendRejectedProposalEvent(t, noAccountPartyID)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, noAccountPartyID)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, errNoBalanceForParty.Error())
}

func testSubmittingProposalWithoutEnoughStakeFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	emptyParty := eng.newValidParty("no-token-party", 0)
	proposal := eng.newOpenProposal(emptyParty.Id, time.Now())

	// setup
	eng.setMinProposerBalance("10000")
	eng.expectAnyAsset()
	eng.expectSendRejectedProposalEvent(t, emptyParty.Id)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, emptyParty.Id)

	// then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposer have insufficient governance token, expected >=")
}

func testSubmittingProposalWithBadClosingTimeFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := time.Now()

	cases := []struct {
		msg              string
		closingTimestamp int64
		error            string
	}{
		{
			msg:              "proposal closing time cannot be earlier than expected",
			closingTimestamp: now.Unix(),
			error:            "proposal closing time too soon, expected >",
		},
		{
			msg:              "proposal closing time cannot be later than expected",
			closingTimestamp: now.Add(3 * 365 * 24 * time.Hour).Unix(),
			error:            "proposal closing time too late, expected <",
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			party := eng.newValidPartyTimes("party", 1000, 0)
			proposal := eng.newOpenProposal(party.Id, now)
			proposal.Terms.ClosingTimestamp = c.closingTimestamp

			// setup
			eng.expectAnyAsset()
			eng.expectSendRejectedProposalEvent(t, party.Id)

			// when
			_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)

			// then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), c.error)
		})
	}
}

func testSubmittingProposalWithBadEnactmentTimeFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := time.Now()

	cases := []struct {
		msg                string
		enactmentTimestamp int64
		error              string
	}{
		{
			msg:                "proposal enactment time cannot be earlier than expected",
			enactmentTimestamp: now.Unix(),
			error:              "proposal enactment time too soon, expected >",
		},
		{
			msg:                "proposal enactment time cannot be later than expected",
			enactmentTimestamp: now.Add(3 * 365 * 24 * time.Hour).Unix(),
			error:              "proposal enactment time too late, expected <",
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			// given
			party := eng.newValidPartyTimes("party", 1000, 0)
			proposal := eng.newOpenProposal(party.Id, now)
			proposal.Terms.EnactmentTimestamp = c.enactmentTimestamp

			// setup
			eng.expectAnyAsset()
			eng.expectSendRejectedProposalEvent(t, party.Id)

			// when
			_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)

			// then
			assert.Error(t, err)
			assert.Contains(t, err.Error(), c.error)
		})
	}
}

func testSubmittingProposalWithBadRiskParameter(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	now := time.Now()
	party := eng.newValidPartyTimes("a-valid-party", 1, 1)
	eng.expectAnyAsset()

	proposal := eng.newOpenProposal(party.Id, now)
	proposal.Terms.GetNewMarket().Changes.RiskParameters = &types.NewMarketConfiguration_LogNormal{
		LogNormal: &types.LogNormalRiskModel{
			Params: nil, // it's nil by zero value, but eh, let's show that's what we test
		},
	}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)

	// then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid risk parameter")
}

func testSubmittingProposalWithClosingTimeBeforeValidationTimeFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	now := time.Now()
	party := eng.newValidPartyTimes("a-valid-party", 1, 0)
	proposal := eng.newOpenProposal(party.Id, now)
	proposal.Terms.ValidationTimestamp = proposal.Terms.ClosingTimestamp + 10
	proposal.Terms.Change = &types.ProposalTerms_NewAsset{}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)

	// then
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal closing time cannot be before validation time, expected >")
}

func testSubmittingValidVoteOnExistingProposalSucceeds(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := eng.newValidParty("proposer", 1)
	voter := eng.newValidParty("voter", 1)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())

	// setup
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter.Id)

	// then
	assert.NoError(t, err)
}

func testSubmittingVoteOnNonexistingProposalFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	voter := eng.newValidPartyTimes("voter", 1, 0)
	voteSub := types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: "id-of-non-existent-proposal",
	}

	// setup
	eng.expectAnyAsset()

	// when
	err := eng.AddVote(context.Background(), voteSub, voter.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testSubmittingVoteWithNonexistingAccountFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := eng.newValidParty("proposer", 1)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())

	// setup
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// given
	voterNoAccount := "voter-no-account"
	vote := types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}

	// setup
	eng.expectNoAccountForParty(voterNoAccount)

	// when
	err = eng.AddVote(context.Background(), vote, voterNoAccount)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, errNoBalanceForParty.Error())
}

func testSubmittingVoteWithoutTokenFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	proposer := eng.newValidParty("proposer", 1)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())

	// setup
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// given
	voterWithEmptyAccount := eng.newValidParty("empty-account", 0)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voterWithEmptyAccount.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrVoterInsufficientTokens.Error())
}

func testSubmittingMajorityOfYesVoteMakesProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	voter2 := eng.newValidPartyTimes("voter2", 1, 0)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())

	// setup
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(9))
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter1, proposal)

	// then
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter1.Id)

	// then
	assert.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, proposal.ID, p.Id)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "7", v.TotalGovernanceTokenBalance())
	})

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalID: proposal.ID,
	}, voter2.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalPassed.Error())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterEnactment)

	// then
	assert.Len(t, toBeEnacted, 1)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalID: proposal.ID,
	}, voter2.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testSubmittingMajorityOfYesVoteDuringValidationMakesProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	voter2 := eng.newValidPartyTimes("voter2", 1, 0)

	now := time.Now()

	id := eng.newProposalID()
	proposal := types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     proposer.Id,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newValidAssetTerms(), // TODO: add more variaty here (when available)
		},
	}

	var bAsset *assets.Asset

	// setup
	var fcheck func(interface{}, bool)
	var rescheck validators.Resource
	eng.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(ref string, assetDetails *types.AssetDetails) (string, error) {
		bAsset = assets.NewAsset(builtin.New(ref, assetDetails))
		return ref, nil
	})
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).DoAndReturn(func(id string) (*assets.Asset, error) {
		return bAsset, nil
	})
	eng.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Do(func(r validators.Resource, f func(interface{}, bool), _ time.Time) error {
		fcheck = f
		rescheck = r
		return nil
	})
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(9))
	eng.expectAnyAsset()
	eng.expectSendWaitingForNodeVoteProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter1, proposal)

	// then
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter1.Id)

	// call success on the validation
	fcheck(rescheck, true)

	// then
	assert.NoError(t, err)
	afterValidation := time.Unix(proposal.Terms.ValidationTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposal.ID, p.Id)
	})

	// when
	eng.OnChainTimeUpdate(context.Background(), afterValidation)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, proposal.ID, p.Id)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "7", v.TotalGovernanceTokenBalance())
	})

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalID: proposal.ID,
	}, voter2.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalPassed.Error())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterEnactment)

	// then
	assert.Len(t, toBeEnacted, 1)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalID: proposal.ID,
	}, voter2.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testSubmittingMajorityOfInsuccifientParticipationMakesProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	now := time.Now()
	proposer := eng.newValidParty("proposer", 100)
	// voter := eng.newValidPartyTimes("voter", 100, 3)
	voter := eng.newValidPartyTimes("voter", 100, 2)
	proposal := eng.newOpenProposal(proposer.Id, now)

	// setup
	eng.expectAnyAsset()
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(800))
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)
	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter.Id)

	// then
	assert.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_DECLINED, p.State, p.State.String())
		assert.Equal(t, proposal.ID, p.Id)
		assert.Equal(t, proto.ProposalError_PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED, p.Reason)
	})

	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "100", v.TotalGovernanceTokenBalance())
	})

	// when
	_, voteClosed := eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// then
	assert.Len(t, voteClosed, 1)
	vc := voteClosed[0]
	assert.NotNil(t, vc.NewMarket())
	assert.True(t, vc.NewMarket().Rejected())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterEnactment)

	// then
	assert.Empty(t, toBeEnacted)
}

func testSubmittingMajorityOfNoVoteMakesProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	now := time.Now()
	proposer := eng.newValidParty("proposer", 100)
	voter := eng.newValidPartyTimes("voter", 100, 3)
	proposal := eng.newOpenProposal(proposer.Id, now)

	// setup
	eng.expectAnyAsset()
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(200))
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalID: proposal.ID,
	}, voter.Id)

	// then
	assert.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_DECLINED, p.State)
		assert.Equal(t, proposal.ID, p.Id)
		assert.Equal(t, proto.ProposalError_PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED, p.Reason)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "100", v.TotalGovernanceTokenBalance())
	})

	// when
	_, voteClosed := eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// then
	assert.Len(t, voteClosed, 1)
	vc := voteClosed[0]
	assert.NotNil(t, vc.NewMarket())
	assert.True(t, vc.NewMarket().Rejected())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterEnactment)

	// then
	assert.Empty(t, toBeEnacted)
}

func testMultipleProposalsLifecycle(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.expectAnyAsset()

	partyA := "party-A"
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().AnyTimes().
		Return(num.NewUint(300))
	accountA := types.Account{
		ID:      partyA + "-account",
		Owner:   partyA,
		Balance: num.NewUint(200),
		Asset:   "VOTE",
	}
	eng.accounts.EXPECT().GetAvailableBalance(accountA.Owner).AnyTimes().Return(accountA.Balance, nil)
	partyB := "party-B"
	accountB := types.Account{
		ID:      partyB + "-account",
		Owner:   partyB,
		Balance: num.NewUint(100),
		Asset:   "VOTE",
	}
	eng.accounts.EXPECT().GetAvailableBalance(accountB.Owner).AnyTimes().Return(accountB.Balance, nil)

	const howMany = 100
	now := time.Now()

	passed := map[string]*types.Proposal{}
	declined := map[string]*types.Proposal{}

	var afterClosing time.Time
	var afterEnactment time.Time

	for i := 0; i < howMany; i++ {
		toBePassed := eng.newOpenProposal(partyA, now)
		eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
			pe, ok := e.(*events.Proposal)
			assert.True(t, ok)
			p := pe.Proposal()
			assert.Equal(t, proto.Proposal_STATE_OPEN, p.State)
		})
		_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&toBePassed), toBePassed.ID, partyA)
		assert.NoError(t, err)
		passed[toBePassed.ID] = &toBePassed

		toBeDeclined := eng.newOpenProposal(partyB, now)
		_, err = eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&toBeDeclined), toBeDeclined.ID, partyB)
		assert.NoError(t, err)
		declined[toBeDeclined.ID] = &toBeDeclined

		if i == 0 {
			// all proposal terms are expected to be equal
			afterClosing = time.Unix(toBePassed.Terms.ClosingTimestamp, 0).Add(time.Second)
			afterEnactment = time.Unix(toBePassed.Terms.EnactmentTimestamp, 0).Add(time.Second)
		}
	}
	assert.Len(t, passed, howMany)
	assert.Len(t, declined, howMany)

	for id := range passed {
		eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
			ve, ok := e.(*events.Vote)
			assert.True(t, ok)
			vote := ve.Vote()
			assert.Equal(t, id, vote.ProposalId)
		})
		err := eng.AddVote(context.Background(), types.VoteSubmission{
			Value:      proto.Vote_VALUE_YES, // matters!
			ProposalID: id,
		}, partyA)
		assert.NoError(t, err)
		err = eng.AddVote(context.Background(), types.VoteSubmission{
			Value:      proto.Vote_VALUE_NO, // matters!
			ProposalID: id,
		}, partyB)
		assert.NoError(t, err)
	}
	for id := range declined {
		eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
			ve, ok := e.(*events.Vote)
			assert.True(t, ok)
			vote := ve.Vote()
			assert.Equal(t, id, vote.ProposalId)
		})
		err := eng.AddVote(context.Background(), types.VoteSubmission{
			Value:      proto.Vote_VALUE_NO, // matters!
			ProposalID: id,
		}, partyA)
		assert.NoError(t, err)
		err = eng.AddVote(context.Background(), types.VoteSubmission{
			Value:      proto.Vote_VALUE_YES, // matters!
			ProposalID: id,
		}, partyB)
		assert.NoError(t, err)
	}

	var howManyPassed, howManyDeclined int
	eng.broker.EXPECT().Send(gomock.Any()).Times(howMany * 2).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		if p.State == proto.Proposal_STATE_PASSED {
			_, found := passed[p.Id]
			assert.True(t, found, "passed proposal is in the passed collection")
			howManyPassed++
		} else if p.State == proto.Proposal_STATE_DECLINED {
			_, found := declined[p.Id]
			assert.True(t, found, "declined proposal is in the declined collection")
			howManyDeclined++
		} else {
			assert.FailNow(t, "unexpected proposal state")
		}
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(howMany * 2)
	eng.OnChainTimeUpdate(context.Background(), afterClosing)
	assert.Equal(t, howMany, howManyPassed)
	assert.Equal(t, howMany, howManyDeclined)

	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterEnactment)
	assert.Len(t, toBeEnacted, howMany)
	for i := 0; i < howMany; i++ {
		_, found := passed[toBeEnacted[i].Proposal().ID]
		assert.True(t, found)
	}
}

func testSubmittingVoteAndWithdrawingFundsDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	now := time.Now()
	proposer := eng.newValidParty("proposer", 100)
	voter := eng.newValidPartyTimes("voter", 100, 2)
	proposal := eng.newOpenProposal(proposer.Id, now)

	// setup
	eng.expectAnyAsset()
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(200))
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalID: proposal.ID,
	}, voter.Id)

	// then
	assert.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_DECLINED, p.State)
		assert.Equal(t, proposal.ID, p.Id)
		assert.Equal(t, proto.ProposalError_PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED.String(), p.Reason.String())
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "0", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "0", v.TotalGovernanceTokenBalance())
	})

	// when

	// we set the call to the balance to return 0
	account := types.Account{
		ID:      "voter" + "-account",
		Owner:   "voter",
		Balance: num.Zero(),
		Asset:   "VOTE",
	}

	eng.accounts.EXPECT().GetAvailableBalance("voter").Times(1).Return(account.Balance, nil)

	_, voteClosed := eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// then
	assert.Len(t, voteClosed, 1)
	vc := voteClosed[0]
	assert.NotNil(t, vc.NewMarket())
	assert.True(t, vc.NewMarket().Rejected())

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterEnactment)

	// then
	assert.Empty(t, toBeEnacted)
}

func testGovernanceHash(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	require.Equal(t,
		"a1292c11ccdb876535c6699e8217e1a1294190d83e4233ecc490d32df17a4116",
		hex.EncodeToString(eng.Hash()),
		"hash is not deterministic",
	)

	// when
	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	proposal := eng.newOpenProposal(proposer.Id, time.Now())

	// setup
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(9))
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter1, proposal)

	// then
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter1.Id)

	// then
	assert.NoError(t, err)

	// test hash before enactement
	require.Equal(t,
		"d43f721a8e28c5bad0e78ab7052b8990be753044bb355056519fab76e8de50a7",
		hex.EncodeToString(eng.Hash()),
		"hash is not deterministic",
	)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, proposal.ID, p.Id)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "7", v.TotalGovernanceTokenBalance())
	})

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// given
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)

	// when
	// no calculations, no state change, simply removed from governance engine
	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterEnactment)

	// then
	assert.Len(t, toBeEnacted, 1)

	require.Equal(t,
		"fbf86f159b135501153cda0fc333751df764290a3ae61c3f45f19f9c19445563",
		hex.EncodeToString(eng.Hash()),
		"hash is not deterministic",
	)
}

func testValidFreeformProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := eng.newOpenFreeformProposal(party.Id, time.Now())

	// setup
	eng.expectSendOpenProposalEvent(t, party, proposal)

	// when
	toSubmit, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)

	// then
	assert.NoError(t, err)
	assert.NotNil(t, toSubmit)
}

func testFreeformProposalDoesNotWaitToEnact(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	proposer := eng.newValidParty("proposer", 1)
	voter1 := eng.newValidPartyTimes("voter-1", 7, 2)
	voter2 := eng.newValidPartyTimes("voter2", 1, 0)
	proposal := eng.newOpenFreeformProposal(proposer.Id, time.Now())

	// setup
	eng.accounts.EXPECT().GetStakingAssetTotalSupply().Times(1).
		Return(num.NewUint(9))
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter1, proposal)

	// then
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalID: proposal.ID,
	}, voter1.Id)

	// then
	assert.NoError(t, err)

	// given
	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		pe, ok := evt.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, proposal.ID, p.Id)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, "7", v.TotalGovernanceTokenBalance())
	})

	// when the proposal is closed, it is enacted immediately
	toBeEnacted, _ := eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// then
	assert.Len(t, toBeEnacted, 1)
	assert.Equal(t, proposal.ID, toBeEnacted[0].Proposal().ID)

	// when
	err = eng.AddVote(context.Background(), types.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalID: proposal.ID,
	}, voter2.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testInvalidFreeformProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// given
	id := eng.newProposalID()
	now := time.Now()
	d := "I am much too long I am much too long I am much too long I am much too long I am much too long"
	party := eng.newValidParty("a-valid-party", 123456789)
	proposal := types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     "a-valid-party",
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change: &types.ProposalTerms_NewFreeform{
				NewFreeform: &types.NewFreeform{
					URL:         "https://example.com",
					Description: d + d + d,
					Hash:        "2fb572edea4af9154edeff680e23689ed076d08934c60f8a4c1f5743a614954e",
				},
			},
		},
	}

	// setup
	eng.expectSendRejectedProposalEvent(t, party.Id)

	// when
	toSubmit, err := eng.SubmitProposal(context.Background(), *types.ProposalSubmissionFromProposal(&proposal), proposal.ID, party.Id)

	// then
	assert.ErrorIs(t, err, governance.ErrFreeformDescriptionTooLong)
	assert.Nil(t, toSubmit)
}

func getTestEngine(t *testing.T) *tstEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	cfg := governance.NewDefaultConfig()
	accounts := mocks.NewMockStakingAccounts(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	broker := bmock.NewMockBroker(ctrl)
	witness := mocks.NewMockWitness(ctrl)

	log := logging.NewTestLogger()
	broker.EXPECT().Send(gomock.Any()).Times(2)
	netp := netparams.New(log, netparams.NewDefaultConfig(), broker)
	_ = netp.Update(context.Background(), netparams.GovernanceProposalMarketMinVoterBalance, "1")
	require.NoError(t, netp.Update(context.Background(), netparams.GovernanceProposalMarketRequiredParticipation, "0.5"))
	now := time.Now()
	now = now.Truncate(time.Second)
	eng := governance.NewEngine(log, cfg, accounts, broker, assets, witness, netp, now) // started as a validator
	assert.NotNil(t, eng)
	return &tstEngine{
		Engine:   eng,
		ctrl:     ctrl,
		accounts: accounts,
		broker:   broker,
		assets:   assets,
		witness:  witness,
		netp:     netp,
	}
}

func newValidFreeformTerms() *types.ProposalTerms_NewFreeform {
	return &types.ProposalTerms_NewFreeform{
		NewFreeform: &types.NewFreeform{
			URL:         "https://example.com",
			Description: "Test my freeform proposal",
			Hash:        "2fb572edea4af9154edeff680e23689ed076d08934c60f8a4c1f5743a614954e",
		},
	}
}

func newValidAssetTerms() *types.ProposalTerms_NewAsset {
	return &types.ProposalTerms_NewAsset{
		NewAsset: &types.NewAsset{
			Changes: &types.AssetDetails{
				Name:        "token",
				Symbol:      "TKN",
				TotalSupply: num.NewUint(10000),
				Decimals:    18,
				MinLpStake:  num.NewUint(1),
				Source: &types.AssetDetailsBuiltinAsset{
					BuiltinAsset: &types.BuiltinAsset{
						MaxFaucetAmountMint: num.NewUint(1),
					},
				},
			},
		},
	}
}

func newValidMarketTerms() *types.ProposalTerms_NewMarket {
	return &types.ProposalTerms_NewMarket{
		NewMarket: &types.NewMarket{
			Changes: &types.NewMarketConfiguration{
				Instrument: &types.InstrumentConfiguration{
					Name: "June 2020 GBP vs VUSD future",
					Code: "CRYPTO:GBPVUSD/JUN20",
					Product: &types.InstrumentConfiguration_Future{
						Future: &types.FutureProduct{
							Maturity:        "2030-06-30T22:59:59Z",
							SettlementAsset: "VUSD",
							QuoteName:       "VUSD",
							OracleSpecForSettlementPrice: &oraclesv1.OracleSpecConfiguration{
								PubKeys: []string{"0xDEADBEEF"},
								Filters: []*oraclesv1.Filter{
									{
										Key: &oraclesv1.PropertyKey{
											Name: "prices.ETH.value",
											Type: oraclesv1.PropertyKey_TYPE_INTEGER,
										},
										Conditions: []*oraclesv1.Condition{},
									},
								},
							},
							OracleSpecForTradingTermination: &oraclesv1.OracleSpecConfiguration{
								PubKeys: []string{"0xDEADBEEF"},
								Filters: []*oraclesv1.Filter{
									{
										Key: &oraclesv1.PropertyKey{
											Name: "trading.terminated",
											Type: oraclesv1.PropertyKey_TYPE_BOOLEAN,
										},
										Conditions: []*oraclesv1.Condition{},
									},
								},
							},
							OracleSpecBinding: &types.OracleSpecToFutureBinding{
								SettlementPriceProperty:    "prices.ETH.value",
								TradingTerminationProperty: "trading.terminated",
							},
						},
					},
				},
				RiskParameters: &types.NewMarketConfiguration_LogNormal{
					LogNormal: &types.LogNormalRiskModel{
						RiskAversionParameter: num.DecimalFromFloat(0.01),
						Tau:                   num.DecimalFromFloat(0.00011407711613050422),
						Params: &types.LogNormalModelParams{
							Mu:    num.DecimalZero(),
							R:     num.DecimalFromFloat(0.016),
							Sigma: num.DecimalFromFloat(0.09),
						},
					},
				},
				Metadata:      []string{"asset_class:fx/crypto", "product:futures"},
				DecimalPlaces: 0,
				TradingMode: &types.NewMarketConfiguration_Continuous{
					Continuous: &types.ContinuousTrading{
						TickSize: "0.1",
					},
				},
			},
			LiquidityCommitment: newMarketLiquidityCommitment(),
		},
	}
}

func newMarketLiquidityCommitment() *types.NewMarketCommitment {
	return &types.NewMarketCommitment{
		CommitmentAmount: num.NewUint(1000),
		Fee:              num.DecimalFromFloat(0.5),
		Sells: []*types.LiquidityOrder{
			{Reference: proto.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: num.NewUint(10)},
		},
		Buys: []*types.LiquidityOrder{
			{Reference: proto.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: num.NewUint(10)},
		},
	}
}

func (e *tstEngine) newValidPartyTimes(partyID string, balance uint64, times int) *proto.Party {
	account := types.Account{
		ID:      partyID + "-account",
		Owner:   partyID,
		Balance: num.NewUint(balance),
		Asset:   "VOTE",
	}
	e.accounts.EXPECT().GetAvailableBalance(partyID).Times(times).Return(account.Balance, nil)
	return &proto.Party{Id: partyID}
}

func (e *tstEngine) newValidParty(partyID string, balance uint64) *proto.Party {
	return e.newValidPartyTimes(partyID, balance, 1)
}

func (e *tstEngine) newProposalID() string {
	e.proposalCounter++
	return fmt.Sprintf("proposal-id-%d", e.proposalCounter)
}

func (e *tstEngine) newOpenProposal(partyID string, now time.Time) types.Proposal {
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
			Change:              newValidMarketTerms(), // TODO: add more variaty here (when available)
		},
	}
}

func (e *tstEngine) newOpenAssetProposal(partyID string, now time.Time) types.Proposal {
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
			Change:              newValidAssetTerms(),
		},
	}
}

func (e *tstEngine) newOpenFreeformProposal(partyID string, now time.Time) types.Proposal {
	id := e.newProposalID()
	return types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		Party:     partyID,
		State:     types.ProposalStateOpen,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newValidFreeformTerms(),
		},
	}
}

func (e *tstEngine) expectAnyAsset() {
	details := newValidAssetTerms()
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(id string) (*assets.Asset, error) {
		ret := assets.NewAsset(builtin.New(id, details.NewAsset.Changes))
		return ret, nil
	})
	e.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
}

func (e *tstEngine) expectAnyAssetTimes(times int) {
	details := newValidAssetTerms()
	e.assets.EXPECT().Get(gomock.Any()).Times(times).DoAndReturn(func(id string) (*assets.Asset, error) {
		ret := assets.NewAsset(builtin.New(id, details.NewAsset.Changes))
		return ret, nil
	})
	e.assets.EXPECT().IsEnabled(gomock.Any()).Times(times).Return(true)
}

func (e *tstEngine) expectSendOpenProposalEvent(t *testing.T, party *proto.Party, proposal types.Proposal) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(ev events.Event) {
		pe, ok := ev.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyId)
		assert.Equal(t, proposal.ID, p.Id)
	})
}

func (e *tstEngine) expectSendWaitingForNodeVoteProposalEvent(t *testing.T, party *proto.Party, proposal types.Proposal) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(ev events.Event) {
		pe, ok := ev.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		fmt.Printf("PROPOSAL: %v\n", p.String())
		assert.Equal(t, proto.Proposal_STATE_WAITING_FOR_NODE_VOTE, p.State)
		assert.Equal(t, party.Id, p.PartyId)
		assert.Equal(t, proposal.ID, p.Id)
	})
}

func (e *tstEngine) expectSendRejectedProposalEvent(t *testing.T, partyID string) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, partyID, p.PartyId)
	})
}

func (e *tstEngine) expectNoAccountForParty(partyID string) {
	e.accounts.EXPECT().GetAvailableBalance(partyID).Times(1).Return(nil, errNoBalanceForParty)
}

func (e *tstEngine) setMinProposerBalance(balance string) {
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	if err := e.netp.Update(
		context.Background(),
		netparams.GovernanceProposalMarketMinProposerBalance,
		balance,
	); err != nil {
		panic(fmt.Errorf("failed to set GovernanceProposalMarketMinProposerBalance parameter: %v", err))
	}
}

func (e *tstEngine) expectSendVoteEvent(t *testing.T, party *proto.Party, proposal types.Proposal) {
	t.Helper()
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		ve, ok := e.(*events.Vote)
		assert.True(t, ok)
		vote := ve.Vote()
		assert.Equal(t, proposal.ID, vote.ProposalId)
		assert.Equal(t, party.Id, vote.PartyId)
	})
}
