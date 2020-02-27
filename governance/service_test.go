package governance_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/orders/mocks"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/tj/assert"
)

type testSvcBundle struct {
	ctrl  *gomock.Controller
	ctx   context.Context
	cfunc context.CancelFunc

	time *mocks.MockTimeService
	gov  *governance.Svc
}

func newTestServiceBundle(t *testing.T) *testSvcBundle {
	ctrl := gomock.NewController(t)
	time := mocks.NewMockTimeService(ctrl)

	ctx, cfunc := context.WithCancel(context.Background())

	svc := governance.NewService(logging.NewTestLogger(), governance.NewDefaultConfig(), time)
	assert.NotNil(t, svc)

	return &testSvcBundle{
		ctrl:  ctrl,
		ctx:   ctx,
		cfunc: cfunc,
		time:  time,
		gov:   svc,
	}
}

func TestGovernanceService(t *testing.T) {
	svc := newTestServiceBundle(t)

	cfg := svc.gov.Config
	cfg.Level.Level = logging.DebugLevel
	svc.gov.ReloadConf(cfg)
	assert.Equal(t, svc.gov.Config.Level.Level, logging.DebugLevel)

	cfg.Level.Level = logging.InfoLevel
	svc.gov.ReloadConf(cfg)
	assert.Equal(t, svc.gov.Config.Level.Level, logging.InfoLevel)
}

func TestPrepareProposal(t *testing.T) {
	svc := newTestServiceBundle(t)

	updateNetwork := types.Proposal_Terms_UpdateNetwork{
		Changes: &types.NetworkConfiguration{
			MinCloseInDays: 100,
			MaxCloseInDays: 1000,
		},
	}
	terms := types.Proposal_Terms{
		Parameters: &types.Proposal_Terms_Parameters{
			CloseInDays:           30,
			EnactInDays:           31,
			MinParticipationStake: 50,
		},
		Change: &types.Proposal_Terms_UpdateNetwork_{
			UpdateNetwork: &updateNetwork,
		},
	}

	rightNow := time.Now()
	svc.time.EXPECT().GetTimeNow().Times(1).Return(rightNow, nil)

	testAuthor := "test-author"
	proposal, err := svc.gov.PrepareProposal(svc.ctx, testAuthor, "", &terms)

	assert.NoError(t, err)
	assert.NotNil(t, proposal)
	assert.NotEmpty(t, proposal.Reference, "reference expected to be auto-generated if empty")
	assert.EqualValues(t, testAuthor, proposal.Party)
	assert.EqualValues(t, types.Proposal_OPEN, proposal.State)
	assert.EqualValues(t, terms, *proposal.Terms)
}

func TestPrepareEmptyProposal(t *testing.T) {
	svc := newTestServiceBundle(t)

	updateNetwork := types.Proposal_Terms_UpdateNetwork{
		Changes: &types.NetworkConfiguration{},
	}
	terms := types.Proposal_Terms{
		Parameters: &types.Proposal_Terms_Parameters{},
		Change: &types.Proposal_Terms_UpdateNetwork_{
			UpdateNetwork: &updateNetwork,
		},
	}

	svc.time.EXPECT().GetTimeNow().MaxTimes(0)

	proposal, err := svc.gov.PrepareProposal(svc.ctx, "", "", &terms)

	assert.Error(t, err)
	assert.Nil(t, proposal)
	assert.Contains(t, err.Error(), "proposal validation failed")
}
