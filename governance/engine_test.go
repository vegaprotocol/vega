package governance_test

import (
	"context"
	"fmt"
	"testing"
	"time"

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

type streamEvt interface {
	events.Event
	StreamMessage() *types.BusEvent
}

type voteMatcher struct{}

type tstEngine struct {
	*governance.Engine
	ctrl            *gomock.Controller
	accs            *mocks.MockAccounts
	broker          *mocks.MockBroker
	erc             *mocks.MockExtResChecker
	assets          *mocks.MockAssets
	netp            *netparams.Store
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
	// once proposal is validated, it is added to the buffer
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err := eng.SubmitProposal(context.Background(), eng.newOpenProposal(party.Id, time.Now()), "proposal-id")
	assert.NoError(t, err)
}

func testProposalState(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	var tokens uint64 = 1000
	party := eng.makeValidParty("valid-party", tokens)

	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Times(1).Return(true)

	unspecified := eng.newOpenProposal(party.Id, time.Now())
	unspecified.State = types.Proposal_STATE_UNSPECIFIED
	err := eng.SubmitProposal(context.Background(), unspecified, "proposal-id1")
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	failed := eng.newOpenProposal(party.Id, time.Now())
	failed.State = types.Proposal_STATE_FAILED
	err = eng.SubmitProposal(context.Background(), failed, "proposal-id2")
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	passed := eng.newOpenProposal(party.Id, time.Now())
	passed.State = types.Proposal_STATE_PASSED
	err = eng.SubmitProposal(context.Background(), passed, "proposal-id3")
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	rejected := eng.newOpenProposal(party.Id, time.Now())
	rejected.State = types.Proposal_STATE_REJECTED
	err = eng.SubmitProposal(context.Background(), rejected, "proposal-id4")
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	declined := eng.newOpenProposal(party.Id, time.Now())
	declined.State = types.Proposal_STATE_DECLINED
	err = eng.SubmitProposal(context.Background(), declined, "proposal-id5")
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	enacted := eng.newOpenProposal(party.Id, time.Now())
	enacted.State = types.Proposal_STATE_ENACTED
	err = eng.SubmitProposal(context.Background(), enacted, "proposal-id6")
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err = eng.SubmitProposal(context.Background(), eng.newOpenProposal(party.Id, time.Now()), "proposal-id7")
	assert.NoError(t, err)
}

func testProposalDuplicate(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).Times(1).Return(true)

	var balance uint64 = 1000
	party := eng.makeValidParty("valid-party", balance)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})

	original := eng.newOpenProposal(party.Id, time.Now())
	err := eng.SubmitProposal(context.Background(), original, "proposal-id")
	assert.NoError(t, err)

	aCopy := original
	aCopy.Reference = "this-is-a-copy"
	err = eng.SubmitProposal(context.Background(), aCopy, "proposal-id")
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())

	aCopy = original
	aCopy.State = types.Proposal_STATE_PASSED
	err = eng.SubmitProposal(context.Background(), aCopy, "proposal-id")
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error(), "reject atempt to change state indirectly")
}

func testProposerStake(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	// default is 0
	// let'sset it up to more so it can fail
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.netp.Update(
		context.Background(),
		netparams.GovernanceProposalMarketMinProposerBalance,
		"10000")

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
	err := eng.SubmitProposal(context.Background(), eng.newOpenProposal(noAccountPartyID, time.Now()), "proposal-id")
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	emptyParty := eng.makeValidParty("no-token-party", 0)
	// eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(123456))
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, emptyParty.Id, p.PartyID)
	})
	err = eng.SubmitProposal(context.Background(), eng.newOpenProposal(emptyParty.Id, time.Now()), "proposal-id1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposer have insufficient governance token, expected >=")

	// eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(123456))
	poshParty := eng.makeValidParty("party-with-tokens", 123456-100)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, poshParty.Id, p.PartyID)
	})
	err = eng.SubmitProposal(context.Background(), eng.newOpenProposal(poshParty.Id, time.Now()), "proposal-id2")
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
	tooEarly := eng.newOpenProposal(party.Id, now)
	tooEarly.Terms.ClosingTimestamp = now.Unix()
	err := eng.SubmitProposal(context.Background(), tooEarly, "proposal-id")
	fmt.Printf("ERROR: %v\n", err)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal closing time too soon, expected >")

	tooLate := eng.newOpenProposal(party.Id, now)
	tooLate.Terms.ClosingTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()
	err = eng.SubmitProposal(context.Background(), tooLate, "proposal-id2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal closing time too late, expected <")

	// eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(1))
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err = eng.SubmitProposal(context.Background(), eng.newOpenProposal(party.Id, now), "proposal-id3")
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
	beforeClosingTime := eng.newOpenProposal(party.Id, now)
	beforeClosingTime.Terms.EnactmentTimestamp = now.Unix()
	assert.Less(t, beforeClosingTime.Terms.EnactmentTimestamp, beforeClosingTime.Terms.ClosingTimestamp)
	err := eng.SubmitProposal(context.Background(), beforeClosingTime, "proposal-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal enactment time too soon, expected >")

	tooLate := eng.newOpenProposal(party.Id, now)
	tooLate.Terms.EnactmentTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()
	err = eng.SubmitProposal(context.Background(), tooLate, "proposal-id1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal enactment time too lat, expected <")

	atClosingTime := eng.newOpenProposal(party.Id, now)
	atClosingTime.Terms.EnactmentTimestamp = atClosingTime.Terms.ClosingTimestamp
	// eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(1))
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err = eng.SubmitProposal(context.Background(), atClosingTime, "proposal-id2")
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
	prop := eng.newOpenProposal(party.Id, now)
	prop.Terms.ValidationTimestamp = prop.Terms.ClosingTimestamp + 10
	prop.Terms.Change = &types.ProposalTerms_NewAsset{}
	err := eng.SubmitProposal(context.Background(), prop, "proposal-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "proposal closing time cannot be before validation time, expected >")
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

	voter := eng.makeValidParty("voter", 1)
	vote := types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES,
		ProposalID: "id-of-non-existent-porposal",
	}
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	eng.assets.EXPECT().IsEnabled(gomock.Any()).AnyTimes().Return(true)
	eng.broker.EXPECT().Send(voteMatcher{}).Times(1).Do(func(evt events.Event) {
		// check we're getting the corret event
		assert.Equal(t, events.TxErrEvent, evt.Type())
		se, ok := evt.(streamEvt)
		assert.True(t, ok)
		be := se.StreamMessage()
		assert.Equal(t, types.BusEventType_BUS_EVENT_TYPE_TX_ERROR, be.Type)
		txErr := be.GetTxErrEvent()
		assert.NotNil(t, txErr)
		assert.Equal(t, governance.ErrProposalNotFound.Error(), txErr.ErrMsg)
		v := txErr.GetVote()
		assert.NotNil(t, v)
		assert.Equal(t, vote, *v)
	})

	err := eng.AddVote(context.Background(), vote)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(2)) // 2 proposals + 1 valid vote

	// default is 0
	// let'sset it up to more so it can fail
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.netp.Update(
		context.Background(),
		netparams.GovernanceProposalMarketMinProposerBalance,
		"1")

	emptyProposer := eng.makeValidParty("empty-proposer", 0)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, emptyProposer.Id, p.PartyID)
	})
	rejectedProposal := eng.newOpenProposal(emptyProposer.Id, time.Now())
	err = eng.SubmitProposal(context.Background(), rejectedProposal, "proposal-id")
	assert.Error(t, err)
	vote = types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_NO,
		ProposalID: rejectedProposal.ID,
	}
	eng.broker.EXPECT().Send(voteMatcher{}).Times(1).Do(func(evt events.Event) {
		// check we're getting the corret event
		assert.Equal(t, events.TxErrEvent, evt.Type())
		se, ok := evt.(streamEvt)
		assert.True(t, ok)
		be := se.StreamMessage()
		assert.Equal(t, types.BusEventType_BUS_EVENT_TYPE_TX_ERROR, be.Type)
		txErr := be.GetTxErrEvent()
		assert.NotNil(t, txErr)
		assert.Equal(t, governance.ErrProposalNotFound.Error(), txErr.ErrMsg)
		v := txErr.GetVote()
		assert.NotNil(t, v)
		assert.Equal(t, vote, *v)
	})

	err = eng.AddVote(context.Background(), vote)
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

	openProposal := eng.newOpenProposal(goodProposer.Id, time.Now())
	err = eng.SubmitProposal(context.Background(), openProposal, "proposal-id1")
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
	eng.accs.EXPECT().GetTotalTokens().Times(2).Return(uint64(2))

	proposer := eng.makeValidParty("proposer", 1)
	openProposal := eng.newOpenProposal(proposer.Id, time.Now())
	eng.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(e events.Event) {
		pe, ok := e.(*events.Proposal)
		assert.True(t, ok)
		p := pe.Proposal()
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
	})
	err := eng.SubmitProposal(context.Background(), openProposal, "proposal-id1")
	assert.NoError(t, err)

	voterNoAccount := "voter-no-account"
	notFoundError := errors.New("account not found")
	eng.accs.EXPECT().GetPartyTokenAccount(voterNoAccount).Times(1).Return(nil, notFoundError)
	eng.broker.EXPECT().Send(voteMatcher{}).Times(1)
	err = eng.AddVote(context.Background(), types.Vote{
		PartyID:    voterNoAccount,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "proposal-id1",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	eng.broker.EXPECT().Send(voteMatcher{}).Times(1).Do(func(evt events.Event) {
		ve, ok := evt.(streamEvt)
		assert.True(t, ok)
		be := ve.StreamMessage()
		txErr := be.GetTxErrEvent()
		assert.NotNil(t, txErr)
		assert.Equal(t, governance.ErrVoterInsufficientTokens.Error(), txErr.ErrMsg)
	})
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

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(2))

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
	err := eng.SubmitProposal(context.Background(), declined, "proposal-id1")
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
	eng.broker.EXPECT().Send(voteMatcher{}).Times(1)
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

	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(9))

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
	err := eng.SubmitProposal(context.Background(), passed, "proposal-id1")
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
	eng.broker.EXPECT().Send(voteMatcher{}).Times(1)
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

	eng.broker.EXPECT().Send(voteMatcher{}).Times(1)
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
	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(200))
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
	err := eng.SubmitProposal(context.Background(), proposal, proposal.ID)
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
	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(100))
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
	err := eng.SubmitProposal(context.Background(), proposal, proposal.ID)
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
	err = eng.SubmitProposal(context.Background(), proposal, proposal.ID)
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
		Asset:   "VOTE",
	}
	eng.accs.EXPECT().GetPartyTokenAccount(accountA.Owner).AnyTimes().Return(&accountA, nil)
	partyB := "party-B"
	accountB := types.Account{
		Id:      partyB + "-account",
		Owner:   partyB,
		Balance: 100,
		Asset:   "VOTE",
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
		err := eng.SubmitProposal(context.Background(), toBePassed, toBePassed.ID)
		assert.NoError(t, err)
		passed[toBePassed.ID] = &toBePassed

		toBeDeclined := eng.newOpenProposal(partyB, now)
		err = eng.SubmitProposal(context.Background(), toBeDeclined, toBeDeclined.ID)
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
	for id := range declined {
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
	now := time.Now()
	now = now.Truncate(time.Second)
	eng, err := governance.NewEngine(log, cfg, accs, broker, assets, erc, netp, now) // started as a validator
	assert.NotNil(t, eng)
	assert.NoError(t, err)
	return &tstEngine{
		Engine: eng,
		ctrl:   ctrl,
		accs:   accs,
		broker: broker,
		assets: assets,
		erc:    erc,
		netp:   netp,
	}
}

func newValidMarketTerms() *types.ProposalTerms_NewMarket {
	return &types.ProposalTerms_NewMarket{
		NewMarket: &types.NewMarket{
			Changes: &types.NewMarketConfiguration{
				Instrument: &types.InstrumentConfiguration{
					Name:      "June 2020 GBP vs VUSD future",
					Code:      "CRYPTO:GBPVUSD/JUN20",
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
			},
		},
	}
}

func (e *tstEngine) makeValidPartyTimes(partyID string, balance uint64, times int) *types.Party {
	account := types.Account{
		Id:      partyID + "-account",
		Owner:   partyID,
		Balance: balance,
		Asset:   "VOTE",
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
	if vote := txErr.GetVote(); vote == nil {
		return false
	}
	return true
}
