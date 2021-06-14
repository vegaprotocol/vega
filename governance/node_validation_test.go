package governance_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testNodeValidation struct {
	*governance.NodeValidation
	ctrl    *gomock.Controller
	assets  *mocks.MockAssets
	witness *mocks.MockWitness
}

func getTestNodeValidation(t *testing.T) *testNodeValidation {
	ctrl := gomock.NewController(t)
	assets := mocks.NewMockAssets(ctrl)
	witness := mocks.NewMockWitness(ctrl)

	nv, err := governance.NewNodeValidation(
		logging.NewTestLogger(), assets, time.Now(), witness)
	assert.NotNil(t, nv)
	assert.Nil(t, err)

	return &testNodeValidation{
		NodeValidation: nv,
		ctrl:           ctrl,
		assets:         assets,
		witness:        witness,
	}
}

func TestNodeValidation(t *testing.T) {
	t.Run("test node validation required - true", testNodeValidationRequiredTrue)
	t.Run("test node validation required - false", testNodeValidationRequiredFalse)

	t.Run("start - error no node validation required", testStartErrorNoNodeValidationRequired)
	t.Run("start - error check proposal failed", testStartErrorCheckProposalFailed)
}

func testNodeValidationRequiredTrue(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	p := &proto.Proposal{
		Terms: &proto.ProposalTerms{
			Change: &proto.ProposalTerms_NewAsset{},
		},
	}

	assert.True(t, nv.IsNodeValidationRequired(p))
}

func testNodeValidationRequiredFalse(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	p := &proto.Proposal{
		Terms: &proto.ProposalTerms{
			Change: &proto.ProposalTerms_NewMarket{},
		},
	}

	assert.False(t, nv.IsNodeValidationRequired(p))
}

func testStartErrorNoNodeValidationRequired(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	p := &proto.Proposal{
		Terms: &proto.ProposalTerms{
			Change: &proto.ProposalTerms_NewMarket{},
		},
	}

	err := nv.Start(p)
	assert.EqualError(t, err, governance.ErrNoNodeValidationRequired.Error())
}

func testStartErrorCheckProposalFailed(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	// first closing time < validation time
	p := &proto.Proposal{
		Terms: &proto.ProposalTerms{
			ClosingTimestamp:    1,
			ValidationTimestamp: 2,
			Change:              &proto.ProposalTerms_NewAsset{},
		},
	}

	err := nv.Start(p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampInvalid.Error())

	// now both are under required duration
	p.Terms.ClosingTimestamp = 3
	err = nv.Start(p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampInvalid.Error())

}
