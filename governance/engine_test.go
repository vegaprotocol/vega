package governance_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type tstEngine struct {
	*governance.Engine
	ctrl   *gomock.Controller
	accs   *mocks.MockAccounts
	broker *mocks.MockBroker
	erc    *mocks.MockExtResChecker
	assets *mocks.MockAssets
	// netp            *mocks.MockNetParams
	proposalCounter uint // to streamline proposal generation
}

func TestSubmitProposals(t *testing.T) {
	t.Run("Submit a valid proposal - success", testSubmitValidProposal)
	t.Run("Validate proposal state on submission", testProposalState)
	t.Run("Validate duplicate proposal", testProposalDuplicate)
	t.Run("Validate closing time", testClosingTime)
	t.Run("Validate enactment time", testEnactmentTime)
	t.Run("Validate timestamps", testValidateTimestamps)
	t.Run("Validate proposer stake", testProposerStake)
}

func testSubmitValidProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	var balance uint64 = 123456789
	party := eng.makeValidParty("a-valid-party", balance)

	// to check min required level
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Times(1).Return(true)
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(balance)
	// once proposal is validated, it is added to the buffer
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id")
	err := eng.SubmitProposal(ctx, eng.newOpenProposal(party.Id, time.Now()))
	assert.NoError(t, err)
}

func testProposalState(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	var tokens uint64 = 1000
	party := eng.makeValidParty("valid-party", tokens)

	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Times(1).Return(true)

	ctx := contextutil.WithCommandID(context.Background(), "proposal-id1")
	unspecified := eng.newOpenProposal(party.Id, time.Now())
	unspecified.State = types.Proposal_STATE_UNSPECIFIED
	err := eng.SubmitProposal(ctx, unspecified)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	ctx = contextutil.WithCommandID(context.Background(), "proposal-id2")
	failed := eng.newOpenProposal(party.Id, time.Now())
	failed.State = types.Proposal_STATE_FAILED
	err = eng.SubmitProposal(ctx, failed)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	ctx = contextutil.WithCommandID(context.Background(), "proposal-id3")
	passed := eng.newOpenProposal(party.Id, time.Now())
	passed.State = types.Proposal_STATE_PASSED
	err = eng.SubmitProposal(ctx, passed)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	ctx = contextutil.WithCommandID(context.Background(), "proposal-id4")
	rejected := eng.newOpenProposal(party.Id, time.Now())
	rejected.State = types.Proposal_STATE_REJECTED
	err = eng.SubmitProposal(ctx, rejected)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	ctx = contextutil.WithCommandID(context.Background(), "proposal-id5")
	declined := eng.newOpenProposal(party.Id, time.Now())
	declined.State = types.Proposal_STATE_DECLINED
	err = eng.SubmitProposal(ctx, declined)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	ctx = contextutil.WithCommandID(context.Background(), "proposal-id6")
	enacted := eng.newOpenProposal(party.Id, time.Now())
	enacted.State = types.Proposal_STATE_ENACTED
	err = eng.SubmitProposal(ctx, enacted)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(tokens)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	ctx = contextutil.WithCommandID(context.Background(), "proposal-id7")
	err = eng.SubmitProposal(ctx, eng.newOpenProposal(party.Id, time.Now()))
	assert.NoError(t, err)
}

func testProposalDuplicate(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Times(1).Return(true)

	var balance uint64 = 1000
	party := eng.makeValidParty("valid-party", balance)
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(balance)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})

	ctx := contextutil.WithCommandID(context.Background(), "proposal-id")
	original := eng.newOpenProposal(party.Id, time.Now())
	err := eng.SubmitProposal(ctx, original)
	assert.NoError(t, err)

	aCopy := original
	aCopy.Reference = "this-is-a-copy"
	ctx = contextutil.WithCommandID(context.Background(), "proposal-id")
	err = eng.SubmitProposal(ctx, aCopy)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())

	ctx = contextutil.WithCommandID(context.Background(), "proposal-id")
	aCopy = original
	aCopy.State = types.Proposal_STATE_PASSED
	err = eng.SubmitProposal(ctx, aCopy)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error(), "reject atempt to change state indirectly")
}

func testProposerStake(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	noAccountPartyID := "party"

	notFoundError := errors.New("account not found")
	eng.accs.EXPECT().GetPartyTokenAccount(noAccountPartyID).Times(1).Return(nil, notFoundError)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, noAccountPartyID, p.PartyID)
	})
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id")
	err := eng.SubmitProposal(ctx, eng.newOpenProposal(noAccountPartyID, time.Now()))
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	emptyParty := eng.makeValidParty("no-token-party", 0)
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(123456))
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, emptyParty.Id, p.PartyID)
	})
	ctx = contextutil.WithCommandID(context.Background(), "proposal-id1")
	err = eng.SubmitProposal(ctx, eng.newOpenProposal(emptyParty.Id, time.Now()))
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalInsufficientTokens.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(123456))
	poshParty := eng.makeValidParty("party-with-tokens", 123456-100)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, poshParty.Id, p.PartyID)
	})
	ctx = contextutil.WithCommandID(context.Background(), "proposal-id2")
	err = eng.SubmitProposal(ctx, eng.newOpenProposal(poshParty.Id, time.Now()))
	assert.NoError(t, err)
}

func testClosingTime(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	party := eng.makeValidParty("a-valid-party", 1)

	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})

	now := time.Now()
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id")
	tooEarly := eng.newOpenProposal(party.Id, now)
	tooEarly.Terms.ClosingTimestamp = now.Unix()
	err := eng.SubmitProposal(ctx, tooEarly)
	fmt.Printf("ERROR: %v\n", err)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalCloseTimeTooSoon.Error())

	ctx = contextutil.WithCommandID(context.Background(), "proposal-id2")
	tooLate := eng.newOpenProposal(party.Id, now)
	tooLate.Terms.ClosingTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()
	err = eng.SubmitProposal(ctx, tooLate)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalCloseTimeTooLate.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(1))
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	ctx = contextutil.WithCommandID(context.Background(), "proposal-id3")
	err = eng.SubmitProposal(ctx, eng.newOpenProposal(party.Id, now))
	assert.NoError(t, err)
}

func testEnactmentTime(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := eng.makeValidParty("a-valid-party", 1)

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})

	now := time.Now()
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id")
	beforeClosingTime := eng.newOpenProposal(party.Id, now)
	beforeClosingTime.Terms.EnactmentTimestamp = now.Unix()
	assert.Less(t, beforeClosingTime.Terms.EnactmentTimestamp, beforeClosingTime.Terms.ClosingTimestamp)
	err := eng.SubmitProposal(ctx, beforeClosingTime)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalEnactTimeTooSoon.Error())

	ctx = contextutil.WithCommandID(ctx, "proposal-id1")
	tooLate := eng.newOpenProposal(party.Id, now)
	tooLate.Terms.EnactmentTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()
	err = eng.SubmitProposal(ctx, tooLate)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalEnactTimeTooLate.Error())

	ctx = contextutil.WithCommandID(context.Background(), "proposal-id2")
	atClosingTime := eng.newOpenProposal(party.Id, now)
	atClosingTime.Terms.EnactmentTimestamp = atClosingTime.Terms.ClosingTimestamp
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(1))
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err = eng.SubmitProposal(ctx, atClosingTime)
	assert.NoError(t, err)
}

func testValidateTimestamps(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := eng.makeValidParty("a-valid-party", 1)
	// for some unknown reason this previous utilities expect a mock assertion while doing
	// nothing. basically this test utilities contaminates the other tests
	// this will need to be refactored
	eng.accs.GetPartyTokenAccount(party.Id)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)

	now := time.Now()
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id")
	prop := eng.newOpenProposal(party.Id, now)
	prop.Terms.ValidationTimestamp = prop.Terms.ClosingTimestamp + 10
	err := eng.SubmitProposal(ctx, prop)
	assert.EqualError(t, err, governance.ErrIncompatibleTimestamps.Error())
}

func TestVoteValidation(t *testing.T) {
	t.Run("Test proposal id on a vote", testVoteProposalID)
	t.Run("Test voter stake validation", testVoterStake)
	t.Run("Test voting on a declined proposal", testVotingDeclinedProposal)
	t.Run("Test voting on a passed proposal", testVotingPassedProposal)
	t.Run("Test proposal lifecycle - declined", testProposalDeclined)
	t.Run("Test proposal lifecycle - passed", testProposalPassed)
	t.Run("Test multiple proposal lifecycle", testMultipleProposalsLifecycle)
}

func testVoteProposalID(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	voter := eng.makeValidParty("voter", 1)

	err := eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "id-of-non-existent-proposal",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(2)) // 2 proposals + 1 valid vote

	emptyProposer := eng.makeValidParty("empty-proposer", 0)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, emptyProposer.Id, p.PartyID)
	})
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id")
	rejectedProposal := eng.newOpenProposal(emptyProposer.Id, time.Now())
	err = eng.SubmitProposal(ctx, rejectedProposal)
	assert.Error(t, err)

	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_NO, // does not matter
		ProposalID: rejectedProposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())

	goodProposer := eng.makeValidParty("proposer", 1)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, goodProposer.Id, p.PartyID)
	})
	ctx = contextutil.WithCommandID(context.Background(), "proposal-id1")
	openProposal := eng.newOpenProposal(goodProposer.Id, time.Now())
	err = eng.SubmitProposal(ctx, openProposal)
	assert.NoError(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		ve, ok := e.(*events.Vote)
		assert.True(t, ok)
		vote := ve.Vote()
		assert.Equal(t, "proposal-id1", vote.ProposalID)
		assert.Equal(t, voter.Id, vote.PartyID)
	})
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "proposal-id1",
	})
	assert.NoError(t, err)
}

func testVoterStake(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(2))

	proposer := eng.makeValidParty("proposer", 1)
	openProposal := eng.newOpenProposal(proposer.Id, time.Now())
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
	})
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id1")
	err := eng.SubmitProposal(ctx, openProposal)
	assert.NoError(t, err)

	voterNoAccount := "voter-no-account"
	notFoundError := errors.New("account not found")
	eng.accs.EXPECT().GetPartyTokenAccount(voterNoAccount).Times(1).Return(nil, notFoundError)
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voterNoAccount,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "proposal-id1",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	emptyAccount := eng.makeValidParty("empty-account", 0)
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    emptyAccount.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "proposal-id1",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrVoterInsufficientTokens.Error())

	validAccount := eng.makeValidParty("valid-account", 1)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		ve, ok := e.(*events.Vote)
		assert.True(t, ok)
		vote := ve.Vote()
		assert.Equal(t, "proposal-id1", vote.ProposalID)
		assert.Equal(t, validAccount.Id, vote.PartyID)
	})
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    validAccount.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "proposal-id1",
	})
	assert.NoError(t, err)
}

func testVotingDeclinedProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.accs.EXPECT().GetTotalTokens().Times(2).Return(uint64(2))

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	proposer := eng.makeValidParty("proposer", 1)
	declined := eng.newOpenProposal(proposer.Id, time.Now())
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
		assert.Equal(t, "proposal-id1", p.ID)
	})
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id1")
	err := eng.SubmitProposal(ctx, declined)
	assert.NoError(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_DECLINED, p.State)
		assert.Equal(t, "proposal-id1", p.ID)
	})
	afterClose := time.Unix(declined.Terms.ClosingTimestamp, 0).Add(time.Hour)
	accepted := eng.OnChainTimeUpdate(context.Background(), afterClose)
	assert.Empty(t, accepted) // nothing was accepted

	voter := eng.makeValidPartyTimes("voter", 1, 0)
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "proposal-id1",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testVotingPassedProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(uint64(9))

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	proposer := eng.makeValidParty("proposer", 1)
	passed := eng.newOpenProposal(proposer.Id, time.Now())
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
		assert.Equal(t, "proposal-id1", p.ID)
	})
	ctx := contextutil.WithCommandID(context.Background(), "proposal-id1")
	err := eng.SubmitProposal(ctx, passed)
	assert.NoError(t, err)

	voter1 := eng.makeValidPartyTimes("voter-1", 7, 2)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		ve, ok := e.(*events.Vote)
		assert.True(t, ok)
		vote := ve.Vote()
		assert.Equal(t, "proposal-id1", vote.ProposalID)
		assert.Equal(t, voter1.Id, vote.PartyID)
	})
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter1.Id,
		Value:      types.Vote_VALUE_YES, // matters!
		ProposalID: "proposal-id1",
	})
	assert.NoError(t, err)

	afterClosing := time.Unix(passed.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, "proposal-id1", p.ID)
	})
	eng.OnChainTimeUpdate(context.Background(), afterClosing)

	voter2 := eng.makeValidPartyTimes("voter2", 1, 0)
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter2.Id,
		Value:      types.Vote_VALUE_NO, // does not matter
		ProposalID: "proposal-id1",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalPassed.Error())

	afterEnactment := time.Unix(passed.Terms.EnactmentTimestamp, 0).Add(time.Second)
	// no calculations, no state change, simply removed from governance engine
	tobeEnacted := eng.OnChainTimeUpdate(context.Background(), afterEnactment)
	assert.Len(t, tobeEnacted, 1)
	assert.Equal(t, "proposal-id1", tobeEnacted[0].Proposal().ID)

	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter2.Id,
		Value:      types.Vote_VALUE_NO, // does not matter
		ProposalID: "proposal-id1",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := time.Now()

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(uint64(200))
	proposer := eng.makeValidParty("proposer", 100)
	voter := eng.makeValidPartyTimes("voter", 100, 3)

	proposal := eng.newOpenProposal(proposer.Id, now)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
		assert.Equal(t, proposal.ID, p.ID)
	})
	ctx := contextutil.WithCommandID(context.Background(), proposal.ID)
	err := eng.SubmitProposal(ctx, proposal)
	assert.NoError(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
		ve, ok := e.(*events.Vote)
		assert.True(t, ok)
		vote := ve.Vote()
		assert.Equal(t, proposal.ID, vote.ProposalID)
		assert.Equal(t, voter.Id, vote.PartyID)
	})
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES, // matters!
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)

	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_NO, // matters!
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)

	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_DECLINED, p.State)
		assert.Equal(t, proposal.ID, p.ID)
	})
	eng.OnChainTimeUpdate(context.Background(), afterClosing)

	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)
	tobeEnacted := eng.OnChainTimeUpdate(context.Background(), afterEnactment)
	assert.Empty(t, tobeEnacted)
}

func testProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := time.Now()

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(uint64(100))
	proposerVoter := eng.makeValidPartyTimes("proposer-and-voter", 100, 3)

	proposal := eng.newOpenProposal(proposerVoter.Id, now)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposerVoter.Id, p.PartyID)
		assert.Equal(t, proposal.ID, p.ID)
	})
	ctx := contextutil.WithCommandID(context.Background(), proposal.ID)
	err := eng.SubmitProposal(ctx, proposal)
	assert.NoError(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		ve, ok := e.(*events.Vote)
		assert.True(t, ok)
		vote := ve.Vote()
		assert.Equal(t, proposal.ID, vote.ProposalID)
		assert.Equal(t, proposerVoter.Id, vote.PartyID)
	})
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    proposerVoter.Id,
		Value:      types.Vote_VALUE_YES, // matters!
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)

	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, proposal.ID, p.ID)
	})
	eng.OnChainTimeUpdate(context.Background(), afterClosing)

	modified := proposal
	modified.State = types.Proposal_STATE_DECLINED
	ctx = contextutil.WithCommandID(context.Background(), proposal.ID)
	err = eng.SubmitProposal(ctx, proposal)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalIsDuplicate.Error())

	eng.makeValidPartyTimes(proposerVoter.Id, 0, 0) // effectively draining proposerVoter
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)
	tobeEnacted := eng.OnChainTimeUpdate(context.Background(), afterEnactment)
	assert.Len(t, tobeEnacted, 1)
	assert.Equal(t, proposal.ID, tobeEnacted[0].Proposal().ID)
}

func testMultipleProposalsLifecycle(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)

	partyA := "party-A"
	eng.accs.EXPECT().GetTotalTokens().AnyTimes().Return(uint64(300))
	accountA := types.Account{
		Id:      partyA + "-account",
		Owner:   partyA,
		Balance: 200,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(accountA.Owner).AnyTimes().Return(&accountA, nil)
	partyB := "party-B"
	accountB := types.Account{
		Id:      partyB + "-account",
		Owner:   partyB,
		Balance: 100,
		Asset:   collateral.TokenAsset,
	}
	eng.accs.EXPECT().GetPartyTokenAccount(accountB.Owner).AnyTimes().Return(&accountB, nil)

	const howMany = 100
	now := time.Now()

	passed := map[string]*types.Proposal{}
	declined := map[string]*types.Proposal{}

	var afterClosing time.Time
	var afterEnactment time.Time

	for i := 0; i < howMany; i++ {
		eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
			pe, ok := e.(*events.Proposal)
			assert.True(t, ok)
			p := pe.Proposal()
			assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		})
		toBePassed := eng.newOpenProposal(partyA, now)
		ctx := contextutil.WithCommandID(context.Background(), toBePassed.ID)
		err := eng.SubmitProposal(ctx, toBePassed)
		assert.NoError(t, err)
		passed[toBePassed.ID] = &toBePassed

		toBeDeclined := eng.newOpenProposal(partyB, now)
		ctx1 := contextutil.WithCommandID(context.Background(), toBeDeclined.ID)
		err = eng.SubmitProposal(ctx1, toBeDeclined)
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

	for id, _ := range passed {
		eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
			ve, ok := e.(*events.Vote)
			assert.True(t, ok)
			vote := ve.Vote()
			assert.Equal(t, id, vote.ProposalID)
		})
		err := eng.AddVote(context.Background(), types.Vote{
			PartyID:    partyA,
			Value:      types.Vote_VALUE_YES, // matters!
			ProposalID: id,
		})
		assert.NoError(t, err)
		err = eng.AddVote(context.Background(), types.Vote{
			PartyID:    partyB,
			Value:      types.Vote_VALUE_NO, // matters!
			ProposalID: id,
		})
		assert.NoError(t, err)
	}
	for id, _ := range declined {
		eng.broker.EXPECT().Send(gomock.Any()).Times(2).Do(func(e events.Event) {
			ve, ok := e.(*events.Vote)
			assert.True(t, ok)
			vote := ve.Vote()
			assert.Equal(t, id, vote.ProposalID)
		})
		err := eng.AddVote(context.Background(), types.Vote{
			PartyID:    partyA,
			Value:      types.Vote_VALUE_NO, // matters!
			ProposalID: id,
		})
		assert.NoError(t, err)
		err = eng.AddVote(context.Background(), types.Vote{
			PartyID:    partyB,
			Value:      types.Vote_VALUE_YES, // matters!
			ProposalID: id,
		})
		assert.NoError(t, err)
	}

	var howManyPassed, howManyDeclined int
	eng.broker.EXPECT().Send(gomock.Any()).Times(howMany * 2).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		if p.State == types.Proposal_STATE_PASSED {
			_, found := passed[p.ID]
			assert.True(t, found, "passed proposal is in the passed collection")
			howManyPassed++
		} else if p.State == types.Proposal_STATE_DECLINED {
			_, found := declined[p.ID]
			assert.True(t, found, "declined proposal is in the declined collection")
			howManyDeclined++
		} else {
			assert.FailNow(t, "unexpected proposal state")
		}
	})
	eng.OnChainTimeUpdate(context.Background(), afterClosing)
	assert.Equal(t, howMany, howManyPassed)
	assert.Equal(t, howMany, howManyDeclined)

	tobeEnacted := eng.OnChainTimeUpdate(context.Background(), afterEnactment)
	assert.Len(t, tobeEnacted, howMany)
	for i := 0; i < howMany; i++ {
		_, found := passed[tobeEnacted[i].Proposal().ID]
		assert.True(t, found)
	}
}

func getTestEngine(t *testing.T) *tstEngine {
	ctrl := gomock.NewController(t)
	cfg := governance.NewDefaultConfig()
	accs := mocks.NewMockAccounts(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	erc := mocks.NewMockExtResChecker(ctrl)

	log := logging.NewTestLogger()
	netp := netparams.New(log, netparams.NewDefaultConfig(), broker)
	eng, err := governance.NewEngine(log, cfg, accs, broker, assets, erc, netp, time.Now()) // started as a validator
	assert.NotNil(t, eng)
	assert.NoError(t, err)
	return &tstEngine{
		Engine: eng,
		ctrl:   ctrl,
		accs:   accs,
		broker: broker,
		assets: assets,
		erc:    erc,
		// netp:   netp,
	}
}

type testVegaWallet struct {
	chain string
	key   []byte
	sig   []byte
}

func (w testVegaWallet) Chain() string { return w.chain }
func (w testVegaWallet) Sign([]byte) ([]byte, error) {
	return w.sig, nil
}
func (w testVegaWallet) PubKeyOrAddress() []byte {
	return w.key
}

func newValidMarketTerms() *types.ProposalTerms_NewMarket {
	return &types.ProposalTerms_NewMarket{
		NewMarket: &types.NewMarket{
			Changes: &types.NewMarketConfiguration{
				Instrument: &types.InstrumentConfiguration{
					Name:      "June 2020 GBP vs VUSD future",
					Code:      "CRYPTO:GBPVUSD/JUN20",
					BaseName:  "GBP",
					QuoteName: "VUSD",
					Product: &types.InstrumentConfiguration_Future{
						Future: &types.FutureProduct{
							Maturity: "2030-06-30T22:59:59Z",
							Asset:    "VUSD",
						},
					},
				},
				RiskParameters: &types.NewMarketConfiguration_LogNormal{
					LogNormal: &types.LogNormalRiskModel{
						RiskAversionParameter: 0.01,
						Tau:                   0.00011407711613050422,
						Params: &types.LogNormalModelParams{
							Mu:    0,
							R:     0.016,
							Sigma: 0.09,
						},
					},
				},
				Metadata:               []string{"asset_class:fx/crypto", "product:futures"},
				DecimalPlaces:          5,
				OpeningAuctionDuration: 30 * 60, // 30 minutes
				TradingMode: &types.NewMarketConfiguration_Continuous{
					Continuous: &types.ContinuousTrading{
						TickSize: "0.1",
					},
				},
				PriceMonitoringSettings: &types.PriceMonitoringSettings{
					UpdateFrequency: 10,
				},
			},
		},
	}
}

func (e *tstEngine) makeValidPartyTimes(partyID string, balance uint64, times int) *types.Party {
	account := types.Account{
		Id:      partyID + "-account",
		Owner:   partyID,
		Balance: balance,
		Asset:   collateral.TokenAsset,
	}
	e.accs.EXPECT().GetPartyTokenAccount(partyID).Times(times).Return(&account, nil)
	return &types.Party{Id: partyID}
}

func (e *tstEngine) makeValidParty(partyID string, balance uint64) *types.Party {
	return e.makeValidPartyTimes(partyID, balance, 1)
}

func (e *tstEngine) newProposalID(partyID string) string {
	e.proposalCounter++
	return fmt.Sprintf("proposal-id-%d", e.proposalCounter)
}

func (e *tstEngine) newOpenProposal(partyID string, now time.Time) types.Proposal {
	id := e.newProposalID(partyID)
	return types.Proposal{
		ID:        id,
		Reference: "ref-" + id,
		PartyID:   partyID,
		State:     types.Proposal_STATE_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp:  now.Add(2 * 48 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(1 * time.Hour).Unix(),
			Change:              newValidMarketTerms(), //TODO: add more variaty here (when available)
		},
	}
}
