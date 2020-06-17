package governance_test

import (
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
	ctrl   *gomock.Controller
	accs   *mocks.MockAccounts
	buf    *mocks.MockBuffer
	vbuf   *mocks.MockVoteBuf
	top    *mocks.MockValidatorTopology
	wal    *mocks.MockWallet
	cmd    *mocks.MockCommander
	assets *mocks.MockAssets
}

func TestProposals(t *testing.T) {
	t.Run("Submit a valid proposal - success", testSubmitValidProposalSuccess)
	t.Run("Submit a valid proposal - duplicate", testSubmitValidProposalDuplicate)
	t.Run("Submit invalid proposal", testSubmitInvalidProposal)
}

func TestVotes(t *testing.T) {
	t.Run("Submit a valid vote - success", testSubmitValidVoteSuccess)
}

func TestTimeUpdate(t *testing.T) {
	t.Run("Accepted proposal on time update", testProposalAccepted)
}

func testSubmitValidProposalSuccess(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	partyID := "party1"
	now := time.Now()
	acc := types.Account{
		Id:      "acc-1",
		Owner:   partyID,
		Balance: 1000,
		Asset:   collateral.TokenAsset,
	}
	prop := types.Proposal{
		ID:        "prop-1",
		Reference: "test-prop-1",
		PartyID:   partyID,
		State:     types.Proposal_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:      now.Add(100 * time.Hour).Unix(),
			EnactmentTimestamp:    now.Add(240 * time.Hour).Unix(),
			MinParticipationStake: 55,
		},
	}
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(1).Return(&acc, nil)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	err := eng.AddProposal(prop)
	assert.NoError(t, err)
}

func testSubmitValidProposalDuplicate(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	partyID := "party1"
	now := time.Now()
	acc := types.Account{
		Id:      "acc-1",
		Owner:   partyID,
		Balance: 1000,
		Asset:   collateral.TokenAsset,
	}
	prop := types.Proposal{
		ID:        "prop-1",
		Reference: "test-prop-1",
		PartyID:   partyID,
		State:     types.Proposal_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:      now.Add(100 * time.Hour).Unix(),
			EnactmentTimestamp:    now.Add(240 * time.Hour).Unix(),
			MinParticipationStake: 55,
		},
	}
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(1).Return(&acc, nil)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_OPEN, p.State)
	})
	err := eng.AddProposal(prop)
	assert.NoError(t, err)
	did, dref := prop, prop
	did.Reference = "Something else"
	dref.ID = "foobar"
	data := map[string]types.Proposal{
		"Duplicate ID":        did,
		"Duplicate Reference": dref,
	}
	for k, prop := range data {
		err = eng.AddProposal(prop)
		assert.Error(t, err)
		assert.Equal(t, governance.ErrProposalIsDuplicate, err, k)
	}
}

func testSubmitInvalidProposal(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	partyID := "party1"
	now := time.Now()
	accErr := errors.New("account not found")
	prop := types.Proposal{
		ID:        "prop-1",
		Reference: "test-prop-1",
		PartyID:   partyID,
		State:     types.Proposal_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:      now.Add(100 * time.Hour).Unix(),
			EnactmentTimestamp:    now.Add(240 * time.Hour).Unix(),
			MinParticipationStake: 55,
		},
	}
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(1).Return(nil, accErr)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1).Do(func(p types.Proposal) {
		assert.Equal(t, types.Proposal_REJECTED, p.State)
	})
	err := eng.AddProposal(prop)
	assert.Error(t, err)
}

func testSubmitValidVoteSuccess(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	partyID := "party1"
	now := time.Now()
	acc := types.Account{
		Id:      "acc-1",
		Owner:   partyID,
		Balance: 1000,
		Asset:   collateral.TokenAsset,
	}
	partyID2 := "party2"
	acc2 := types.Account{
		Id:      "acc-2",
		Owner:   partyID2,
		Balance: 100,
		Asset:   collateral.TokenAsset,
	}
	prop := types.Proposal{
		ID:        "prop-1",
		Reference: "test-prop-1",
		PartyID:   partyID,
		State:     types.Proposal_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:      now.Add(100 * time.Hour).Unix(),
			EnactmentTimestamp:    now.Add(240 * time.Hour).Unix(),
			MinParticipationStake: 55,
		},
	}
	// we will call this getPartyTokenAccount twice
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(2).Return(&acc, nil)
	eng.accs.EXPECT().GetPartyTokenAccount(partyID2).Times(1).Return(&acc2, nil)
	eng.buf.EXPECT().Add(gomock.Any()).Times(1)
	err := eng.AddProposal(prop)
	assert.NoError(t, err)
	vote := types.Vote{
		PartyID:    partyID,
		Value:      types.Vote_YES,
		ProposalID: prop.ID,
	}
	eng.vbuf.EXPECT().Add(gomock.Any()).Times(2)
	assert.NoError(t, eng.AddVote(vote))
	vote.PartyID = partyID2
	vote.Value = types.Vote_NO
	assert.NoError(t, eng.AddVote(vote))
}

func testProposalAccepted(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	partyID := "party1"
	now := time.Now()
	acc := types.Account{
		Id:      "acc-1",
		Owner:   partyID,
		Balance: 1000,
		Asset:   collateral.TokenAsset,
	}
	partyID2 := "party2"
	acc2 := types.Account{
		Id:      "acc-2",
		Owner:   partyID2,
		Balance: 100,
		Asset:   collateral.TokenAsset,
	}
	closeTime := now.Add(100 * time.Hour)
	totalTokens := acc.Balance + acc2.Balance
	prop := types.Proposal{
		ID:        "prop-1",
		Reference: "test-prop-1",
		PartyID:   partyID,
		State:     types.Proposal_OPEN,
		Terms: &types.ProposalTerms{
			ClosingTimestamp:      closeTime.Unix(),
			EnactmentTimestamp:    closeTime.Unix(),
			MinParticipationStake: 55,
		},
	}
	// we will call this getPartyTokenAccount 3 times (1 for creating proposal, 1 for vote, 1 checking vote)
	eng.accs.EXPECT().GetPartyTokenAccount(partyID).Times(3).Return(&acc, nil)
	// call this once -> party is voting no, we just want % of yes votes vs total
	eng.accs.EXPECT().GetPartyTokenAccount(partyID2).Times(1).Return(&acc2, nil)
	eng.accs.EXPECT().GetTotalTokens().Times(1).Return(totalTokens)
	calls := 0
	states := []types.Proposal_State{
		types.Proposal_OPEN,
		types.Proposal_PASSED,
	}
	eng.buf.EXPECT().Add(gomock.Any()).Times(2).Do(func(p types.Proposal) {
		assert.Equal(t, states[calls], p.State)
		calls++
	})
	err := eng.AddProposal(prop)
	assert.NoError(t, err)
	vote := types.Vote{
		PartyID:    partyID,
		Value:      types.Vote_YES,
		ProposalID: prop.ID,
	}
	eng.vbuf.EXPECT().Add(gomock.Any()).Times(2)
	assert.NoError(t, eng.AddVote(vote))
	vote.PartyID = partyID2
	vote.Value = types.Vote_NO
	assert.NoError(t, eng.AddVote(vote))
	// simulate block time update triggering end of proposal vote
	accepted := eng.OnChainTimeUpdate(closeTime.Add(time.Hour))
	assert.NotEmpty(t, accepted)
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

	eng, err := governance.NewEngine(logging.NewTestLogger(), cfg, accs, buf, vbuf, top, wal, cmd, assets, time.Now(), true) // started as a validator
	assert.NotNil(t, eng)
	assert.NoError(t, err)

	buf.EXPECT().Flush().AnyTimes()
	vbuf.EXPECT().Flush().AnyTimes()
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
