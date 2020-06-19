package governance_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testNodeValidation struct {
	*governance.NodeValidation
	ctrl   *gomock.Controller
	top    *mocks.MockValidatorTopology
	wal    *mocks.MockWallet
	cmd    *mocks.MockCommander
	assets *mocks.MockAssets
}

func getTestNodeValidation(t *testing.T) *testNodeValidation {
	ctrl := gomock.NewController(t)
	top := mocks.NewMockValidatorTopology(ctrl)
	wal := mocks.NewMockWallet(ctrl)
	cmd := mocks.NewMockCommander(ctrl)
	assets := mocks.NewMockAssets(ctrl)

	wal.EXPECT().Get(gomock.Any()).Times(1).Return(testVegaWallet{
		chain: "vega",
	}, true)

	nv, err := governance.NewNodeValidation(
		logging.NewTestLogger(), top, wal, cmd, assets, time.Now(), true)
	assert.NotNil(t, nv)
	assert.Nil(t, err)

	return &testNodeValidation{
		NodeValidation: nv,
		ctrl:           ctrl,
		top:            top,
		wal:            wal,
		cmd:            cmd,
		assets:         assets,
	}
}

func TestNodeValidation(t *testing.T) {
	t.Run("test node validation required - true", testNodeValidationRequiredTrue)
	t.Run("test node validation required - false", testNodeValidationRequiredFalse)

	t.Run("start - error no node validation required", testStartErrorNoNodeValidationRequired)
	t.Run("start - error duplicate", testStartErrorDuplicate)
	t.Run("start - error check proposal failed", testStartErrorCheckProposalFailed)
	t.Run("start - unable to instanciate assets", testStartErrorUnableToInstanciateAsset)
	t.Run("start - OK", testStartOK)

	t.Run("add node vote - error invalid proposal reference", testNodeVoteInvalidProposalReference)
	// t.Run("add node vote - error note a validator", testNodeVoteNotAValidator)
	// t.Run("add node vote - error duplicate vote", testNodeVoteDuplicateVote)
	// t.Run("add node vote - OK", testNodeVoteOK)

}

func testNodeValidationRequiredTrue(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewAsset{},
		},
	}

	assert.True(t, nv.IsNodeValidationRequired(p))
}

func testNodeValidationRequiredFalse(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{},
		},
	}

	assert.False(t, nv.IsNodeValidationRequired(p))
}

func testStartErrorNoNodeValidationRequired(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewMarket{},
		},
	}

	err := nv.Start(p)
	assert.EqualError(t, err, governance.ErrNoNodeValidationRequired.Error())
}

func testNodeVoteInvalidProposalReference(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	v := &types.NodeVote{
		Reference: "nope",
	}

	err := nv.AddNodeVote(v)
	assert.EqualError(t, err, governance.ErrInvalidProposalReferenceForNodeVote.Error())
}

func testStartErrorCheckProposalFailed(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	// first closing time < validation time
	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    1,
			ValidationTimestamp: 2,
			Change:              &types.ProposalTerms_NewAsset{},
		},
	}

	err := nv.Start(p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampInvalid.Error())

	// now both are under required duration
	p.Terms.ClosingTimestamp = 3
	err = nv.Start(p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampInvalid.Error())

}

func testStartOK(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	now := time.Now()

	builtinAsset := &types.BuiltinAsset{
		Name:   "USDC",
		Symbol: "USDC",
	}

	// first closing time < validation time
	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(24 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(700 * time.Minute).Unix(),
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_BuiltinAsset{
							BuiltinAsset: builtinAsset,
						},
					},
				},
			},
		},
	}

	nv.assets.EXPECT().NewAsset(
		gomock.Any(), gomock.Any()).
		Times(1).Return("ASSETID", nil)

	ch := make(chan struct{}, 1)
	asset := builtin.New("ASSETID", builtinAsset)
	nv.assets.EXPECT().Get(gomock.Any()).Times(1).DoAndReturn(func(string) (assets.Asset, error) {
		ch <- struct{}{}
		return asset, nil
	})

	err := nv.Start(p)
	<-ch
	assert.NoError(t, err)

}

func testStartErrorDuplicate(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	now := time.Now()

	builtinAsset := &types.BuiltinAsset{
		Name:   "USDC",
		Symbol: "USDC",
	}

	// first closing time < validation time
	p := &types.Proposal{
		Reference: "ref",
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(24 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(700 * time.Minute).Unix(),
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_BuiltinAsset{
							BuiltinAsset: builtinAsset,
						},
					},
				},
			},
		},
	}

	nv.assets.EXPECT().NewAsset(
		gomock.Any(), gomock.Any()).
		Times(1).Return("ASSETID", nil)

	ch := make(chan struct{}, 1)
	asset := builtin.New("ASSETID", builtinAsset)
	nv.assets.EXPECT().Get(gomock.Any()).Times(1).DoAndReturn(func(string) (assets.Asset, error) {
		ch <- struct{}{}
		return asset, nil
	})

	err := nv.Start(p)
	<-ch
	assert.NoError(t, err)

	// first was fine, now let's try to submit it again
	err = nv.Start(p)
	assert.EqualError(t, err, governance.ErrProposalReferenceDuplicate.Error())
}

func testStartErrorUnableToInstanciateAsset(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	now := time.Now()

	// first closing time < validation time
	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    now.Add(24 * time.Hour).Unix(),
			ValidationTimestamp: now.Add(700 * time.Minute).Unix(),
			Change: &types.ProposalTerms_NewAsset{
				NewAsset: &types.NewAsset{
					Changes: &types.AssetSource{
						Source: &types.AssetSource_Erc20{
							Erc20: &types.ERC20{
								ContractAddress: "0xOK",
							},
						},
					},
				},
			},
		},
	}

	nv.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any()).Times(1).Return("", errors.New("unable to instanciate"))

	err := nv.Start(p)
	assert.EqualError(t, err, "unable to instanciate")
}
