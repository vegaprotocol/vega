package governance_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	bmock "code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errStubbedAccountNotFound = errors.New("account not found")
)

type streamEvt interface {
	events.Event
	StreamMessage() *eventspb.BusEvent
}

type voteMatcher struct{}

type tstEngine struct {
	*governance.Engine
	ctrl            *gomock.Controller
	accounts        *mocks.MockAccounts
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

	t.Run("Validate market proposal commitment", testValidateProposalCommitment)
}

func testValidateProposalCommitment(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := eng.newValidPartyTimes("a-valid-party", 1, 10)

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)

	now := time.Now()
	prop := eng.newOpenProposal(party.Id, now)

	// first we test with no commitment
	prop.Terms.GetNewMarket().LiquidityCommitment = nil
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "market proposal is missing liquidity commitment")

	// Then no amount
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.CommitmentAmount = 0
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal commitment amount is 0 or missing")

	// Then empty fees
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Fee = ""
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid liquidity provision fee")

	// Then negative fees
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Fee = "-1"
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid liquidity provision fee")

	// Then empty shapes
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Buys = nil
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty SIDE_BUY shape")

	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Sells = nil
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty SIDE_SELL shape")

	// Then invalid shapes
	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Buys[0].Offset = 100
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order in buy side shape offset must be <= 0")

	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Buys[0].Reference = proto.PeggedReference_PEGGED_REFERENCE_BEST_ASK
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order in buy side shape with best ask price reference")

	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Sells[0].Offset = -100
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order in sell shape offset must be >= 0")

	prop.Terms.GetNewMarket().LiquidityCommitment = newMarketLiquidityCommitment()
	prop.Terms.GetNewMarket().LiquidityCommitment.Sells[0].Reference = proto.PeggedReference_PEGGED_REFERENCE_BEST_BID
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&prop), "proposal-id", party.Id)
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

	toSubmit, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, party.Id)
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
	toSubmit, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, party.Id)

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
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, party.Id)

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
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&original), original.Id, party.Id)

	// then
	assert.NoError(t, err)

	// given
	aCopy := original
	aCopy.Reference = "this-is-a-copy"

	// when
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&aCopy), aCopy.Id, party.Id)

	// then
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())

	// given
	aCopy = original
	aCopy.State = proto.Proposal_STATE_PASSED

	// when
	_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&aCopy), aCopy.Id, party.Id)

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
	eng.expectSendTxErrorProposalEvent(t, noAccountPartyID)
	eng.expectSendRejectedProposalEvent(t, noAccountPartyID)

	// when
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, noAccountPartyID)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, errStubbedAccountNotFound.Error())
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
	eng.expectSendTxErrorProposalEvent(t, emptyParty.Id)
	eng.expectSendRejectedProposalEvent(t, emptyParty.Id)

	// when
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, emptyParty.Id)

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
			eng.expectSendTxErrorProposalEvent(t, party.Id)
			eng.expectSendRejectedProposalEvent(t, party.Id)

			// when
			_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, party.Id)

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
			eng.expectSendTxErrorProposalEvent(t, party.Id)
			eng.expectSendRejectedProposalEvent(t, party.Id)

			// when
			_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, party.Id)

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
	proposal.Terms.GetNewMarket().Changes.RiskParameters = &proto.NewMarketConfiguration_LogNormal{
		LogNormal: &proto.LogNormalRiskModel{
			Params: nil, // it's nil by zero value, but eh, let's show that's what we test
		},
	}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)

	// when
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, party.Id)

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
	proposal.Terms.Change = &proto.ProposalTerms_NewAsset{}

	// setup
	eng.broker.EXPECT().Send(gomock.Any()).Times(2)

	// when
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, party.Id)

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
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)

	// when
	err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalId: proposal.Id,
	}, voter.Id)

	// then
	assert.NoError(t, err)
}

func testSubmittingVoteOnNonexistingProposalFails(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// when
	voter := eng.newValidPartyTimes("voter", 1, 0)
	voteSub := commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalId: "id-of-non-existent-proposal",
	}

	// setup
	eng.expectAnyAsset()
	eng.expectSendProposalNotFoundErrorEvent(t, voteSub)

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
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, proposer.Id)

	// then
	assert.NoError(t, err)

	// given
	voterNoAccount := "voter-no-account"
	vote := commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalId: proposal.Id,
	}

	// setup
	eng.expectNoAccountForParty(voterNoAccount)
	eng.expectSendAccountNotFoundErrorEvent(t, vote)

	// when
	err = eng.AddVote(context.Background(), vote, voterNoAccount)

	// then
	assert.Error(t, err)
	assert.EqualError(t, err, errStubbedAccountNotFound.Error())
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
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, proposer.Id)

	// then
	assert.NoError(t, err)

	// given
	voterWithEmptyAccount := eng.newValidParty("empty-account", 0)

	// setup
	eng.expectSendInsufficientTokensErrorEvent(t)

	// when
	err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalId: proposal.Id,
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
	eng.accounts.EXPECT().GetAssetTotalSupply(gomock.Any()).Times(1).Return(uint64(9), nil)
	eng.expectAnyAsset()
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter1, proposal)

	// then
	err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalId: proposal.Id,
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
		assert.Equal(t, proposal.Id, p.Id)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {

		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, uint64(7), v.TotalGovernanceTokenBalance())
	})

	// when
	eng.OnChainTimeUpdate(context.Background(), afterClosing)

	// setup
	eng.broker.EXPECT().Send(voteMatcher{}).Times(1)

	// when
	err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalId: proposal.Id,
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
	assert.Equal(t, proposal.Id, toBeEnacted[0].Proposal().Id)

	// setup
	eng.broker.EXPECT().Send(voteMatcher{}).Times(1)

	// when
	err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalId: proposal.Id,
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
	eng.accounts.EXPECT().GetAssetTotalSupply(gomock.Any()).Times(1).Return(uint64(800), nil)
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)
	// when
	err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalId: proposal.Id,
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
		assert.Equal(t, proposal.Id, p.Id)
		assert.Equal(t, proto.ProposalError_PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED, p.Reason)
	})

	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, uint64(100), v.TotalGovernanceTokenBalance())
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
	eng.accounts.EXPECT().GetAssetTotalSupply(gomock.Any()).Times(1).Return(uint64(200), nil)
	eng.expectSendOpenProposalEvent(t, proposer, proposal)

	// when
	_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&proposal), proposal.Id, proposer.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)

	// when
	err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_YES,
		ProposalId: proposal.Id,
	}, voter.Id)

	// then
	assert.NoError(t, err)

	// setup
	eng.expectSendVoteEvent(t, voter, proposal)

	// when
	err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
		Value:      proto.Vote_VALUE_NO,
		ProposalId: proposal.Id,
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
		assert.Equal(t, proposal.Id, p.Id)
		assert.Equal(t, proto.ProposalError_PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED, p.Reason)
	})
	eng.broker.EXPECT().SendBatch(gomock.Any()).Times(1).Do(func(evts []events.Event) {
		v, ok := evts[0].(*events.Vote)
		assert.True(t, ok)
		assert.Equal(t, "1", v.TotalGovernanceTokenWeight())
		assert.Equal(t, uint64(100), v.TotalGovernanceTokenBalance())
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
	eng.accounts.EXPECT().GetAssetTotalSupply(gomock.Any()).AnyTimes().Return(uint64(300), nil)
	accountA := types.Account{
		Id:      partyA + "-account",
		Owner:   partyA,
		Balance: 200,
		Asset:   "VOTE",
	}
	eng.accounts.EXPECT().GetPartyGeneralAccount(accountA.Owner, "VOTE").AnyTimes().Return(&accountA, nil)
	partyB := "party-B"
	accountB := types.Account{
		Id:      partyB + "-account",
		Owner:   partyB,
		Balance: 100,
		Asset:   "VOTE",
	}
	eng.accounts.EXPECT().GetPartyGeneralAccount(accountB.Owner, "VOTE").AnyTimes().Return(&accountB, nil)

	const howMany = 100
	now := time.Now()

	passed := map[string]*proto.Proposal{}
	declined := map[string]*proto.Proposal{}

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
		_, err := eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&toBePassed), toBePassed.Id, partyA)
		assert.NoError(t, err)
		passed[toBePassed.Id] = &toBePassed

		toBeDeclined := eng.newOpenProposal(partyB, now)
		_, err = eng.SubmitProposal(context.Background(), *commandspb.ProposalSubmissionFromProposal(&toBeDeclined), toBeDeclined.Id, partyB)
		assert.NoError(t, err)
		declined[toBeDeclined.Id] = &toBeDeclined

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
		err := eng.AddVote(context.Background(), commandspb.VoteSubmission{
			Value:      proto.Vote_VALUE_YES, // matters!
			ProposalId: id,
		}, partyA)
		assert.NoError(t, err)
		err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
			Value:      proto.Vote_VALUE_NO, // matters!
			ProposalId: id,
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
		err := eng.AddVote(context.Background(), commandspb.VoteSubmission{
			Value:      proto.Vote_VALUE_NO, // matters!
			ProposalId: id,
		}, partyA)
		assert.NoError(t, err)
		err = eng.AddVote(context.Background(), commandspb.VoteSubmission{
			Value:      proto.Vote_VALUE_YES, // matters!
			ProposalId: id,
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
		_, found := passed[toBeEnacted[i].Proposal().Id]
		assert.True(t, found)
	}
}

func getTestEngine(t *testing.T) *tstEngine {
	ctrl := gomock.NewController(t)
	cfg := governance.NewDefaultConfig()
	accounts := mocks.NewMockAccounts(ctrl)
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
	eng, err := governance.NewEngine(log, cfg, accounts, broker, assets, witness, netp, now) // started as a validator
	assert.NotNil(t, eng)
	assert.NoError(t, err)
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

func newValidMarketTerms() *proto.ProposalTerms_NewMarket {
	return &proto.ProposalTerms_NewMarket{
		NewMarket: &proto.NewMarket{
			Changes: &proto.NewMarketConfiguration{
				Instrument: &proto.InstrumentConfiguration{
					Name: "June 2020 GBP vs VUSD future",
					Code: "CRYPTO:GBPVUSD/JUN20",
					Product: &proto.InstrumentConfiguration_Future{
						Future: &proto.FutureProduct{
							Maturity:        "2030-06-30T22:59:59Z",
							SettlementAsset: "VUSD",
							QuoteName:       "VUSD",
							OracleSpec: &oraclesv1.OracleSpecConfiguration{
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
							OracleSpecBinding: &proto.OracleSpecToFutureBinding{
								SettlementPriceProperty: "prices.ETH.value",
							},
						},
					},
				},
				RiskParameters: &proto.NewMarketConfiguration_LogNormal{
					LogNormal: &proto.LogNormalRiskModel{
						RiskAversionParameter: 0.01,
						Tau:                   0.00011407711613050422,
						Params: &proto.LogNormalModelParams{
							Mu:    0,
							R:     0.016,
							Sigma: 0.09,
						},
					},
				},
				Metadata:      []string{"asset_class:fx/crypto", "product:futures"},
				DecimalPlaces: 5,
				TradingMode: &proto.NewMarketConfiguration_Continuous{
					Continuous: &proto.ContinuousTrading{
						TickSize: "0.1",
					},
				},
			},
			LiquidityCommitment: newMarketLiquidityCommitment(),
		},
	}
}

func newMarketLiquidityCommitment() *proto.NewMarketCommitment {
	return &proto.NewMarketCommitment{
		CommitmentAmount: 1000,
		Fee:              "0.5",
		Sells: []*proto.LiquidityOrder{
			{Reference: proto.PeggedReference_PEGGED_REFERENCE_BEST_ASK, Proportion: 1, Offset: 10},
		},
		Buys: []*proto.LiquidityOrder{
			{Reference: proto.PeggedReference_PEGGED_REFERENCE_BEST_BID, Proportion: 1, Offset: -10},
		},
	}
}

func (e *tstEngine) newValidPartyTimes(partyID string, balance uint64, times int) *proto.Party {
	account := types.Account{
		Id:      partyID + "-account",
		Owner:   partyID,
		Balance: balance,
		Asset:   "VOTE",
	}
	e.accounts.EXPECT().GetPartyGeneralAccount(partyID, "VOTE").Times(times).Return(&account, nil)
	return &proto.Party{Id: partyID}
}

func (e *tstEngine) newValidParty(partyID string, balance uint64) *proto.Party {
	return e.newValidPartyTimes(partyID, balance, 1)
}

func (e *tstEngine) newProposalID() string {
	e.proposalCounter++
	return fmt.Sprintf("proposal-id-%d", e.proposalCounter)
}

func (e *tstEngine) newOpenProposal(partyID string, now time.Time) proto.Proposal {
	id := e.newProposalID()
	return proto.Proposal{
		Id:        id,
		Reference: "ref-" + id,
		PartyId:   partyID,
		State:     proto.Proposal_STATE_OPEN,
		Terms: &proto.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newValidMarketTerms(), // TODO: add more variaty here (when available)
		},
	}
}

func (e *tstEngine) expectAnyAsset() {
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	e.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
}

func (e *tstEngine) expectAnyAssetTimes(times int) {
	e.assets.EXPECT().Get(gomock.Any()).Times(times).Return(nil, nil)
	e.assets.EXPECT().IsEnabled(gomock.Any()).Times(times).Return(true)
}

func (e *tstEngine) expectSendOpenProposalEvent(t *testing.T, party *proto.Party, proposal proto.Proposal) {
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(ev events.Event) {
		pe, ok := ev.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyId)
		assert.Equal(t, proposal.Id, p.Id)
	})
}

func (e *tstEngine) expectSendRejectedProposalEvent(t *testing.T, partyID string) {
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, proto.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, partyID, p.PartyId)
	})
}

func (e *tstEngine) expectSendTxErrorProposalEvent(t *testing.T, partyID string) {
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.TxErr)
		assert.True(t, ok)
		assert.True(t, pe.IsParty(partyID))
	})
}

func (e *tstEngine) expectSendProposalNotFoundErrorEvent(t *testing.T, vote commandspb.VoteSubmission) {
	e.broker.EXPECT().Send(voteMatcher{}).Times(1).Do(func(evt events.Event) {
		assert.Equal(t, events.TxErrEvent, evt.Type())
		se, ok := evt.(streamEvt)
		assert.True(t, ok)
		be := se.StreamMessage()
		assert.Equal(t, eventspb.BusEventType_BUS_EVENT_TYPE_TX_ERROR, be.Type)
		txErr := be.GetTxErrEvent()
		assert.NotNil(t, txErr)
		assert.Equal(t, governance.ErrProposalNotFound.Error(), txErr.ErrMsg)
		v := txErr.GetVoteSubmission()
		assert.NotNil(t, v)
		assert.Equal(t, vote, *v)
	})
}

func (e *tstEngine) expectSendAccountNotFoundErrorEvent(t *testing.T, vote commandspb.VoteSubmission) {
	e.broker.EXPECT().Send(voteMatcher{}).Times(1).Do(func(evt events.Event) {
		assert.Equal(t, events.TxErrEvent, evt.Type())
		se, ok := evt.(streamEvt)
		assert.True(t, ok)
		be := se.StreamMessage()
		assert.Equal(t, eventspb.BusEventType_BUS_EVENT_TYPE_TX_ERROR, be.Type)
		txErr := be.GetTxErrEvent()
		assert.NotNil(t, txErr)
		assert.Equal(t, errStubbedAccountNotFound.Error(), txErr.ErrMsg)
		v := txErr.GetVoteSubmission()
		assert.NotNil(t, v)
		assert.Equal(t, vote, *v)
	})
}

func (e *tstEngine) expectSendInsufficientTokensErrorEvent(t *testing.T) {
	e.broker.EXPECT().Send(voteMatcher{}).Times(1).Do(func(evt events.Event) {
		ve, ok := evt.(streamEvt)
		assert.True(t, ok)
		be := ve.StreamMessage()
		txErr := be.GetTxErrEvent()
		assert.NotNil(t, txErr)
		assert.Equal(t, governance.ErrVoterInsufficientTokens.Error(), txErr.ErrMsg)
	})
}

func (e *tstEngine) expectNoAccountForParty(partyID string) {
	e.accounts.EXPECT().GetPartyGeneralAccount(partyID, gomock.Any()).Times(1).Return(nil, errStubbedAccountNotFound)
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

func (e *tstEngine) expectSendVoteEvent(t *testing.T, party *proto.Party, proposal proto.Proposal) {
	e.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		ve, ok := e.(*events.Vote)
		assert.True(t, ok)
		vote := ve.Vote()
		assert.Equal(t, proposal.Id, vote.ProposalId)
		assert.Equal(t, party.Id, vote.PartyId)
	})
}

func (v voteMatcher) String() string {
	return "Vote TX error event"
}

func (v voteMatcher) Matches(x interface{}) bool {
	evt, ok := x.(streamEvt)
	if !ok {
		return false
	}
	if evt.Type() != events.TxErrEvent {
		return false
	}
	be := evt.StreamMessage()
	txErr := be.GetTxErrEvent()
	if txErr == nil {
		return false
	}
	if vote := txErr.GetVoteSubmission(); vote == nil {
		return false
	}
	return true
}
