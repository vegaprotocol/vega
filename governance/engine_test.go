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
	"github.com/stretchr/testify/assert"
)

type tstEngine struct {
	*governance.Engine
	ctrl *gomock.Controller
	accs *mocks.MockAccounts
	buf  *mocks.MockBuffer
}

func TestProposals(t *testing.T) {
	t.Run("Submit a valid proposal - success", testSubmitValidProposalSuccess)
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

func getTestEngine(t *testing.T) *tstEngine {
	ctrl := gomock.NewController(t)
	cfg := governance.NewDefaultConfig()
	accs := mocks.NewMockAccounts(ctrl)
	buf := mocks.NewMockBuffer(ctrl)
	eng := governance.NewEngine(logging.NewTestLogger(), cfg, accs, buf, time.Now())
	return &tstEngine{
		Engine: eng,
		ctrl:   ctrl,
		accs:   accs,
		buf:    buf,
	}
}
