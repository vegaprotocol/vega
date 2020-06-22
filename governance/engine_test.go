package governance_test

import (
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type tstEngine struct {
	*governance.Engine
	ctrl            *gomock.Controller
	accs            *mocks.MockAccounts
	buf             *mocks.MockBuffer
	vbuf            *mocks.MockVoteBuf
	top             *mocks.MockValidatorTopology
	wal             *mocks.MockWallet
	cmd             *mocks.MockCommander
	assets          *mocks.MockAssets
	proposalCounter uint // to streamline proposal generation
}

func TestSubmitProposals(t *testing.T) {
	t.Run("Submit a valid proposal - success", testSubmitValidProposal)
	t.Run("Validate proposal state on submission", testProposalState)
	t.Run("Validate duplicate proposal", testProposalDuplicate)
	t.Run("Validate closing time", testClosingTime)
	t.Run("Validate enactment time", testEnactmentTime)
	t.Run("Validate proposer stake", testProposerStake)
}

func testSubmitValidProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	var balance uint64 = 123456789
	party := eng.makeValidParty("a-valid-party", balance)

	// to check min required level
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(balance)
	// once proposal is validated, it is added to the buffer
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err := eng.SubmitProposal(eng.newOpenProposal(party.Id, time.Now()))
	assert.NoError(t, err)
}

func testProposalState(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	var tokens uint64 = 1000
	party := eng.makeValidParty("valid-party", tokens)

	unspecified := eng.newOpenProposal(party.Id, time.Now())
	unspecified.State = types.Proposal_STATE_UNSPECIFIED
	err := eng.SubmitProposal(unspecified)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	failed := eng.newOpenProposal(party.Id, time.Now())
	failed.State = types.Proposal_STATE_FAILED
	err = eng.SubmitProposal(failed)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	passed := eng.newOpenProposal(party.Id, time.Now())
	passed.State = types.Proposal_STATE_PASSED
	err = eng.SubmitProposal(passed)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	rejected := eng.newOpenProposal(party.Id, time.Now())
	rejected.State = types.Proposal_STATE_REJECTED
	err = eng.SubmitProposal(rejected)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	declined := eng.newOpenProposal(party.Id, time.Now())
	declined.State = types.Proposal_STATE_DECLINED
	err = eng.SubmitProposal(declined)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	enacted := eng.newOpenProposal(party.Id, time.Now())
	enacted.State = types.Proposal_STATE_ENACTED
	err = eng.SubmitProposal(enacted)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalInvalidState, err.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(tokens)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err = eng.SubmitProposal(eng.newOpenProposal(party.Id, time.Now()))
	assert.NoError(t, err)
}

func testProposalDuplicate(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	var balance uint64 = 1000
	party := eng.makeValidParty("valid-party", balance)
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(balance)

	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})

	original := eng.newOpenProposal(party.Id, time.Now())
	err := eng.SubmitProposal(original)
	assert.NoError(t, err)

	aCopy := original
	aCopy.Reference = "this-is-a-copy"
	err = eng.SubmitProposal(aCopy)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error())

	aCopy = original
	aCopy.State = types.Proposal_STATE_PASSED
	err = eng.SubmitProposal(aCopy)
	assert.Error(t, err)
	assert.EqualError(t, governance.ErrProposalIsDuplicate, err.Error(), "reject atempt to change state indirectly")
}

func testProposerStake(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	noAccountPartyID := "party"

	notFoundError := errors.New("account not found")
	eng.accs.EXPECT().GetPartyTokenAccount(noAccountPartyID).Times(1).Return(nil, notFoundError)

	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, noAccountPartyID, p.PartyID)
	})
	err := eng.SubmitProposal(eng.newOpenProposal(noAccountPartyID, time.Now()))
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	emptyParty := eng.makeValidParty("no-token-party", 0)
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(123456))
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, emptyParty.Id, p.PartyID)
	})
	err = eng.SubmitProposal(eng.newOpenProposal(emptyParty.Id, time.Now()))
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalInsufficientTokens.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(123456))
	poshParty := eng.makeValidParty("party-with-tokens", 123456-100)

	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, poshParty.Id, p.PartyID)
	})
	err = eng.SubmitProposal(eng.newOpenProposal(poshParty.Id, time.Now()))
	assert.NoError(t, err)
}

func testClosingTime(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := eng.makeValidParty("a-valid-party", 1)

	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(p types.Proposal) {
		fmt.Printf("STATE  %v\n", p.State.String())
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})

	now := time.Now()
	tooEarly := eng.newOpenProposal(party.Id, now)
	tooEarly.Terms.ClosingTimestamp = now.Unix()
	err := eng.SubmitProposal(tooEarly)
	fmt.Printf("ERROR: %v\n", err)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalCloseTimeTooSoon.Error())

	tooLate := eng.newOpenProposal(party.Id, now)
	tooLate.Terms.ClosingTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()
	err = eng.SubmitProposal(tooLate)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalCloseTimeTooLate.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(1))
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err = eng.SubmitProposal(eng.newOpenProposal(party.Id, now))
	assert.NoError(t, err)
}

func testEnactmentTime(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	party := eng.makeValidParty("a-valid-party", 1)

	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})

	now := time.Now()
	beforeClosingTime := eng.newOpenProposal(party.Id, now)
	beforeClosingTime.Terms.EnactmentTimestamp = now.Unix()
	assert.Less(t, beforeClosingTime.Terms.EnactmentTimestamp, beforeClosingTime.Terms.ClosingTimestamp)
	err := eng.SubmitProposal(beforeClosingTime)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalEnactTimeTooSoon.Error())

	tooLate := eng.newOpenProposal(party.Id, now)
	tooLate.Terms.EnactmentTimestamp = now.Add(3 * 365 * 24 * time.Hour).Unix()
	err = eng.SubmitProposal(tooLate)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalEnactTimeTooLate.Error())

	atClosingTime := eng.newOpenProposal(party.Id, now)
	atClosingTime.Terms.EnactmentTimestamp = atClosingTime.Terms.ClosingTimestamp
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(uint64(1))
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, party.Id, p.PartyID)
	})
	err = eng.SubmitProposal(atClosingTime)
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

	err := eng.AddVote(types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: "id-of-non-existent-proposal",
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())

	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(2)) // 2 proposals + 1 valid vote

	emptyProposer := eng.makeValidParty("empty-proposer", 0)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_REJECTED, p.State)
		assert.Equal(t, emptyProposer.Id, p.PartyID)
	})
	rejectedProposal := eng.newOpenProposal(emptyProposer.Id, time.Now())
	err = eng.SubmitProposal(rejectedProposal)
	assert.Error(t, err)

	err = eng.AddVote(types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_NO, // does not matter
		ProposalID: rejectedProposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())

	goodProposer := eng.makeValidParty("proposer", 1)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, goodProposer.Id, p.PartyID)
	})
	openProposal := eng.newOpenProposal(goodProposer.Id, time.Now())
	err = eng.SubmitProposal(openProposal)
	assert.NoError(t, err)

	eng.vbuf.EXPECT().Add(gomock.Any()).Times(1).Do(func(vote types.Vote) {
		assert.Equal(t, openProposal.ID, vote.ProposalID)
		assert.Equal(t, voter.Id, vote.PartyID)
	})
	err = eng.AddVote(types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: openProposal.ID,
	})
	assert.NoError(t, err)
}

func testVoterStake(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.accs.EXPECT().GetTotalTokens().Times(3).Return(uint64(2))

	proposer := eng.makeValidParty("proposer", 1)
	openProposal := eng.newOpenProposal(proposer.Id, time.Now())
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
	})
	err := eng.SubmitProposal(openProposal)
	assert.NoError(t, err)

	voterNoAccount := "voter-no-account"
	notFoundError := errors.New("account not found")
	eng.accs.EXPECT().GetPartyTokenAccount(voterNoAccount).Times(1).Return(nil, notFoundError)
	err = eng.AddVote(types.Vote{
		PartyID:    voterNoAccount,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: openProposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, notFoundError.Error())

	emptyAccount := eng.makeValidParty("empty-account", 0)
	err = eng.AddVote(types.Vote{
		PartyID:    emptyAccount.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: openProposal.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrVoterInsufficientTokens.Error())

	validAccount := eng.makeValidParty("valid-account", 1)
	eng.vbuf.EXPECT().Add(gomock.Any()).Times(1).Do(func(vote types.Vote) {
		assert.Equal(t, openProposal.ID, vote.ProposalID)
		assert.Equal(t, validAccount.Id, vote.PartyID)
	})
	err = eng.AddVote(types.Vote{
		PartyID:    validAccount.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: openProposal.ID,
	})
	assert.NoError(t, err)
}

func testVotingDeclinedProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.accs.EXPECT().GetTotalTokens().Times(2).Return(uint64(2))

	proposer := eng.makeValidParty("proposer", 1)
	declined := eng.newOpenProposal(proposer.Id, time.Now())
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
		assert.Equal(t, declined.ID, p.ID)
	})
	err := eng.SubmitProposal(declined)
	assert.NoError(t, err)

	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_DECLINED, p.State)
		assert.Equal(t, declined.ID, p.ID)
	})
	afterClose := time.Unix(declined.Terms.ClosingTimestamp, 0).Add(time.Hour)
	accepted := eng.OnChainTimeUpdate(afterClose)
	assert.Empty(t, accepted) // nothing was accepted

	voter := eng.makeValidPartyTimes("voter", 1, 0)
	err = eng.AddVote(types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES, // does not matter
		ProposalID: declined.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testVotingPassedProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(uint64(9))

	proposer := eng.makeValidParty("proposer", 1)
	passed := eng.newOpenProposal(proposer.Id, time.Now())
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
		assert.Equal(t, passed.ID, p.ID)
	})
	err := eng.SubmitProposal(passed)
	assert.NoError(t, err)

	voter1 := eng.makeValidPartyTimes("voter-1", 7, 2)

	eng.vbuf.EXPECT().Add(gomock.Any()).Times(1).Do(func(vote types.Vote) {
		assert.Equal(t, passed.ID, vote.ProposalID)
		assert.Equal(t, voter1.Id, vote.PartyID)
	})
	err = eng.AddVote(types.Vote{
		PartyID:    voter1.Id,
		Value:      types.Vote_VALUE_YES, // matters!
		ProposalID: passed.ID,
	})
	assert.NoError(t, err)

	afterClosing := time.Unix(passed.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, passed.ID, p.ID)
	})
	eng.OnChainTimeUpdate(afterClosing)

	voter2 := eng.makeValidPartyTimes("voter2", 1, 0)
	err = eng.AddVote(types.Vote{
		PartyID:    voter2.Id,
		Value:      types.Vote_VALUE_NO, // does not matter
		ProposalID: passed.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalPassed.Error())

	afterEnactment := time.Unix(passed.Terms.EnactmentTimestamp, 0).Add(time.Second)
	// no calculations, no state change, simply removed from governance engine
	tobeEnacted := eng.OnChainTimeUpdate(afterEnactment)
	assert.Len(t, tobeEnacted, 1)
	assert.Equal(t, passed.ID, tobeEnacted[0].ID)

	err = eng.AddVote(types.Vote{
		PartyID:    voter2.Id,
		Value:      types.Vote_VALUE_NO, // does not matter
		ProposalID: passed.ID,
	})
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalNotFound.Error())
}

func testProposalDeclined(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := time.Now()

	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(uint64(200))
	proposer := eng.makeValidParty("proposer", 100)
	voter := eng.makeValidPartyTimes("voter", 100, 3)

	proposal := eng.newOpenProposal(proposer.Id, now)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposer.Id, p.PartyID)
		assert.Equal(t, proposal.ID, p.ID)
	})
	err := eng.SubmitProposal(proposal)
	assert.NoError(t, err)

	eng.vbuf.EXPECT().Add(gomock.Any()).Times(2).Do(func(vote types.Vote) {
		assert.Equal(t, proposal.ID, vote.ProposalID)
		assert.Equal(t, voter.Id, vote.PartyID)
	})
	err = eng.AddVote(types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_YES, // matters!
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)

	err = eng.AddVote(types.Vote{
		PartyID:    voter.Id,
		Value:      types.Vote_VALUE_NO, // matters!
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)

	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_DECLINED, p.State)
		assert.Equal(t, proposal.ID, p.ID)
	})
	eng.OnChainTimeUpdate(afterClosing)

	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)
	tobeEnacted := eng.OnChainTimeUpdate(afterEnactment)
	assert.Empty(t, tobeEnacted)
}

func testProposalPassed(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	now := time.Now()

	eng.accs.EXPECT().GetTotalTokens().Times(4).Return(uint64(100))
	proposerVoter := eng.makeValidPartyTimes("proposer-and-voter", 100, 3)

	proposal := eng.newOpenProposal(proposerVoter.Id, now)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		assert.Equal(t, proposerVoter.Id, p.PartyID)
		assert.Equal(t, proposal.ID, p.ID)
	})
	err := eng.SubmitProposal(proposal)
	assert.NoError(t, err)

	eng.vbuf.EXPECT().Add(gomock.Any()).Times(1).Do(func(vote types.Vote) {
		assert.Equal(t, proposal.ID, vote.ProposalID)
		assert.Equal(t, proposerVoter.Id, vote.PartyID)
	})
	err = eng.AddVote(types.Vote{
		PartyID:    proposerVoter.Id,
		Value:      types.Vote_VALUE_YES, // matters!
		ProposalID: proposal.ID,
	})
	assert.NoError(t, err)

	afterClosing := time.Unix(proposal.Terms.ClosingTimestamp, 0).Add(time.Second)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_STATE_PASSED, p.State)
		assert.Equal(t, proposal.ID, p.ID)
	})
	eng.OnChainTimeUpdate(afterClosing)

	modified := proposal
	modified.State = types.Proposal_STATE_DECLINED
	err = eng.SubmitProposal(proposal)
	assert.Error(t, err)
	assert.EqualError(t, err, governance.ErrProposalIsDuplicate.Error())

	eng.makeValidPartyTimes(proposerVoter.Id, 0, 0) // effectively draining proposerVoter
	afterEnactment := time.Unix(proposal.Terms.EnactmentTimestamp, 0).Add(time.Second)
	tobeEnacted := eng.OnChainTimeUpdate(afterEnactment)
	assert.Len(t, tobeEnacted, 1)
	assert.Equal(t, proposal.ID, tobeEnacted[0].ID)
}

func testMultipleProposalsLifecycle(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

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
		eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(p types.Proposal) {
			assert.Equal(t, types.Proposal_STATE_OPEN, p.State)
		})
		toBePassed := eng.newOpenProposal(partyA, now)
		err := eng.SubmitProposal(toBePassed)
		assert.NoError(t, err)
		passed[toBePassed.ID] = &toBePassed

		toBeDeclined := eng.newOpenProposal(partyB, now)
		err = eng.SubmitProposal(toBeDeclined)
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
		eng.vbuf.EXPECT().Add(gomock.Any()).Times(2).Do(func(vote types.Vote) {
			assert.Equal(t, id, vote.ProposalID)
		})
		err := eng.AddVote(types.Vote{
			PartyID:    partyA,
			Value:      types.Vote_VALUE_YES, // matters!
			ProposalID: id,
		})
		assert.NoError(t, err)
		err = eng.AddVote(types.Vote{
			PartyID:    partyB,
			Value:      types.Vote_VALUE_NO, // matters!
			ProposalID: id,
		})
		assert.NoError(t, err)
	}
	for id, _ := range declined {
		eng.vbuf.EXPECT().Add(gomock.Any()).Times(2).Do(func(vote types.Vote) {
			assert.Equal(t, id, vote.ProposalID)
		})
		err := eng.AddVote(types.Vote{
			PartyID:    partyA,
			Value:      types.Vote_VALUE_NO, // matters!
			ProposalID: id,
		})
		assert.NoError(t, err)
		err = eng.AddVote(types.Vote{
			PartyID:    partyB,
			Value:      types.Vote_VALUE_YES, // matters!
			ProposalID: id,
		})
		assert.NoError(t, err)
	}

	var howManyPassed, howManyDeclined int
	eng.buf.EXPECT().Add(gomock.Any()).Times(howMany * 2).Do(func(p types.Proposal) {
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
	eng.OnChainTimeUpdate(afterClosing)
	assert.Equal(t, howMany, howManyPassed)
	assert.Equal(t, howMany, howManyDeclined)

	tobeEnacted := eng.OnChainTimeUpdate(afterEnactment)
	assert.Len(t, tobeEnacted, howMany)
	for i := 0; i < howMany; i++ {
		_, found := passed[tobeEnacted[i].ID]
		assert.True(t, found)
	}
}

func getTestEngine(t *testing.T) *tstEngine {
	ctrl := gomock.NewController(t)
	cfg := governance.NewDefaultConfig()
	accs := mocks.NewMockAccounts(ctrl)
	buf := mocks.NewMockBuffer(ctrl)
	vbuf := mocks.NewMockVoteBuf(ctrl)
	top := mocks.NewMockValidatorTopology(ctrl)
	wal := mocks.NewMockWallet(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	assets := mocks.NewMockAssets(ctrl)

	wal.EXPECT().Get(gomock.Any()).Times(1).Return(testVegaWallet{
		chain: "vega",
	}, true)

	buf.EXPECT().Flush().AnyTimes()
	vbuf.EXPECT().Flush().AnyTimes()
	log := logging.NewTestLogger()
	eng, err := governance.NewEngine(log, cfg, governance.DefaultNetworkParameters(log), accs, buf, vbuf, top, wal, cmd, assets, time.Now(), true) // started as a validator
	assert.NotNil(t, eng)
	assert.NoError(t, err)
	return &tstEngine{
		Engine: eng,
		ctrl:   ctrl,
		accs:   accs,
		buf:    buf,
		vbuf:   vbuf,
		cmd:    cmd,
		assets: assets,
		top:    top,
		wal:    wal,
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
			Changes: &types.Market{
				Id:            "a-unit-test-market",
				DecimalPlaces: 5,
				Name:          "a-unit-test-market-name",
				TradingMode: &types.Market_Continuous{
					Continuous: &types.ContinuousTrading{
						TickSize: 0,
					},
				},
				TradableInstrument: &types.TradableInstrument{
					Instrument: &types.Instrument{
						Id:        "Crypto/GBPVUSD/Futures/Jun20",
						Code:      "CRYPTO:GBPVUSD/JUN20",
						Name:      "June 2020 GBP vs VUSD future",
						BaseName:  "GBP",
						QuoteName: "VUSD",
						Metadata: &types.InstrumentMetadata{
							Tags: []string{"asset_class:fx/crypto", "product:futures"},
						},
						InitialMarkPrice: 123321,
						Product: &types.Instrument_Future{
							Future: &types.Future{
								Maturity: "2030-06-30T22:59:59Z",
								Asset:    "VUSD",
								Oracle: &types.Future_EthereumEvent{
									EthereumEvent: &types.EthereumEvent{
										ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
										Event:      "price_changed",
									},
								},
							},
						},
					},
					MarginCalculator: &types.MarginCalculator{
						ScalingFactors: &types.ScalingFactors{
							InitialMargin:     1.2,
							CollateralRelease: 1.4,
							SearchLevel:       1.1,
						},
					},
					RiskModel: &types.TradableInstrument_LogNormalRiskModel{
						LogNormalRiskModel: &types.LogNormalRiskModel{
							RiskAversionParameter: 0.01,
							Tau:                   0.00011407711613050422,
							Params: &types.LogNormalModelParams{
								Mu:    0,
								R:     0.016,
								Sigma: 0.09,
							},
						},
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
			ClosingTimestamp:   now.Add(48 * time.Hour).Unix(),
			EnactmentTimestamp: now.Add(2 * 48 * time.Hour).Unix(),
			Change:             newValidMarketTerms(), //TODO: add more variaty here (when available)
		},
	}
}
