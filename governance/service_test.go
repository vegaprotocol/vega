package governance_test

import (
	"testing"

	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/orders/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/tj/assert"
)

type testSvc struct {
	ctrl *gomock.Controller

	time  *mocks.MockTimeService
	block *mocks.MockBlockchain
	svc   *governance.Svc
}

func newTestService(t *testing.T) *testSvc {

	ctrl := gomock.NewController(t)

	time := mocks.NewMockTimeService(ctrl)
	block := mocks.NewMockBlockchain(ctrl)
	return &testSvc{
		ctrl:  ctrl,
		time:  time,
		block: block,
		svc:   governance.NewService(logging.NewTestLogger(), governance.NewDefaultConfig(), time, block),
	}
}

func TestGovernanceEngine(t *testing.T) {
	engine := newTestService(t)
	assert.NotNil(t, engine)
}

func TestValidateProposal(t *testing.T) {
	s := newTestService(t)
	assert.NotNil(t, s.svc)

	updateNetwork := types.Proposal_Terms_UpdateNetwork{
		Changes: &types.NetworkConfiguration{
			MinCloseInDays: 100,
			MaxCloseInDays: 1000,
		},
	}

	proposal := types.Proposal_Terms{
		Parameters: &types.Proposal_Terms_Parameters{
			CloseInDays:           30,
			EnactInDays:           31,
			MinParticipationStake: 50,
		},
		Change: &types.Proposal_Terms_UpdateNetwork_{
			UpdateNetwork: &updateNetwork,
		},
	}
	err := s.svc.ValidateProposal(&proposal)
	assert.NoError(t, err)
}

func TestSubmitProposal(t *testing.T) {
	s := newTestService(t)
	assert.NotNil(t, s.svc)

	updateNetwork := types.Proposal_Terms_UpdateNetwork{
		Changes: &types.NetworkConfiguration{
			MinCloseInDays:        10,
			MaxCloseInDays:        100,
			MinParticipationStake: 70,
		},
	}

	proposal := types.Proposal_Terms{
		Parameters: &types.Proposal_Terms_Parameters{
			CloseInDays:           300,
			EnactInDays:           301,
			MinParticipationStake: 80,
		},
		Change: &types.Proposal_Terms_UpdateNetwork_{
			UpdateNetwork: &updateNetwork,
		},
	}
	confirmation, err := s.svc.SubmitProposal(&proposal)
	assert.NoError(t, err)
	assert.NotNil(t, confirmation)
	assert.EqualValues(t, proposal, *confirmation.Proposal)
	assert.NotEmpty(t, confirmation.Id)
	assert.EqualValues(t, types.Proposal_OPEN, confirmation.State)
}
