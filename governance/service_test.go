package governance_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testSvc struct {
	*governance.Svc
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc

	plugin *mocks.MockPlugin
}

func newTestService(t *testing.T) *testSvc {
	ctrl := gomock.NewController(t)
	plugin := mocks.NewMockPlugin(ctrl)

	ctx, cfunc := context.WithCancel(context.Background())

	result := &testSvc{
		ctrl:   ctrl,
		ctx:    ctx,
		cfunc:  cfunc,
		plugin: plugin,
	}
	result.Svc = governance.NewService(logging.NewTestLogger(), governance.NewDefaultConfig(), plugin)
	assert.NotNil(t, result.Svc)
	return result
}

func TestPrepareVote(t *testing.T) {
	t.Run("prepare vote - success", testPrepareVoteSuccess)
	t.Run("prepare vote - failure", testPrepareVoteFail)
}

func testPrepareVoteSuccess(t *testing.T) {
	svc := newTestService(t)
	defer svc.ctrl.Finish()
	vote := types.Vote{
		PartyID:    "party-1",
		ProposalID: "prop-1",
		Value:      types.Vote_VALUE_YES,
	}
	v, err := svc.PrepareVote(&vote)
	assert.NoError(t, err)
	assert.Equal(t, vote.Value, v.Value)
	assert.Equal(t, vote.PartyID, v.PartyID)
	assert.Equal(t, vote.ProposalID, v.ProposalID)
}

func testPrepareVoteFail(t *testing.T) {
	svc := newTestService(t)
	defer svc.ctrl.Finish()

	data := map[string]types.Vote{
		"Missing PartyID": {
			ProposalID: "prop1",
			Value:      types.Vote_VALUE_NO,
		},
		"Missing ProposalID": {
			PartyID: "Party1",
			Value:   types.Vote_VALUE_YES,
		},
		"Invalid vote value": {
			ProposalID: "prop1",
			PartyID:    "party1",
			Value:      types.Vote_Value(213),
		},
	}
	for k, vote := range data {
		v, err := svc.PrepareVote(&vote)
		assert.Error(t, err, k)
		assert.Nil(t, v, k)
		assert.Equal(t, governance.ErrMissingVoteData, err, k)
	}
}

func TestGovernanceService(t *testing.T) {
	svc := newTestService(t)
	defer svc.ctrl.Finish()

	cfg := svc.Config
	cfg.Level.Level = logging.DebugLevel
	svc.ReloadConf(cfg)
	assert.Equal(t, svc.Config.Level.Level, logging.DebugLevel)

	cfg.Level.Level = logging.InfoLevel
	svc.ReloadConf(cfg)
	assert.Equal(t, svc.Config.Level.Level, logging.InfoLevel)
}

func testPrepareProposalNormal(t *testing.T) {
	svc := newTestService(t)
	defer svc.ctrl.Finish()

	updateNetwork := types.UpdateNetwork{
		Changes: &types.NetworkConfiguration{
			MinCloseInSeconds: 100 * 24 * 60 * 60,
			MaxCloseInSeconds: 1000 * 24 * 60 * 60,
		},
	}
	terms := types.ProposalTerms{
		ClosingTimestamp:   time.Now().Add(time.Hour * 24 * 2).UTC().Unix(),
		EnactmentTimestamp: time.Now().Add(time.Hour * 24 * 60).UTC().Unix(),
		Change: &types.ProposalTerms_UpdateNetwork{
			UpdateNetwork: &updateNetwork,
		},
	}

	testAuthor := "test-author"
	proposal, err := svc.PrepareProposal(svc.ctx, testAuthor, "", &terms)

	assert.NoError(t, err)
	assert.NotNil(t, proposal)
	assert.NotEmpty(t, proposal.Reference, "reference expected to be auto-generated if empty")
	assert.EqualValues(t, testAuthor, proposal.PartyID)
	assert.EqualValues(t, types.Proposal_STATE_OPEN, proposal.State)
	assert.EqualValues(t, terms, *proposal.Terms)
}

func testPrepareProposalEmpty(t *testing.T) {
	svc := newTestService(t)
	defer svc.ctrl.Finish()

	updateNetwork := types.UpdateNetwork{
		Changes: &types.NetworkConfiguration{},
	}
	terms := types.ProposalTerms{
		Change: &types.ProposalTerms_UpdateNetwork{
			UpdateNetwork: &updateNetwork,
		},
	}

	proposal, err := svc.PrepareProposal(svc.ctx, "", "", &terms)

	assert.Error(t, err)
	assert.Nil(t, proposal)
}

func TestPrepareProposal(t *testing.T) {
	t.Run("Prepare a normal proposal", testPrepareProposalNormal)
	t.Run("Prepare an empty proposal", testPrepareProposalEmpty)
}
