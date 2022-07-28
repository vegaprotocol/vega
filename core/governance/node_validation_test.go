// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package governance_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/governance/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"

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
	t.Helper()
	ctrl := gomock.NewController(t)
	assets := mocks.NewMockAssets(ctrl)
	witness := mocks.NewMockWitness(ctrl)

	nv := governance.NewNodeValidation(
		logging.NewTestLogger(), assets, time.Now(), witness)
	assert.NotNil(t, nv)

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

	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTermsNewAsset{},
		},
	}

	assert.True(t, nv.IsNodeValidationRequired(p))
}

func testNodeValidationRequiredFalse(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTermsNewMarket{},
		},
	}

	assert.False(t, nv.IsNodeValidationRequired(p))
}

func testStartErrorNoNodeValidationRequired(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTermsNewMarket{},
		},
	}

	err := nv.Start(context.Background(), p)
	assert.EqualError(t, err, governance.ErrNoNodeValidationRequired.Error())
}

func testStartErrorCheckProposalFailed(t *testing.T) {
	nv := getTestNodeValidation(t)
	defer nv.ctrl.Finish()

	// first closing time < validation time
	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    1,
			ValidationTimestamp: 2,
			Change:              &types.ProposalTermsNewAsset{},
		},
	}

	err := nv.Start(context.Background(), p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampInvalid.Error())

	// now both are under required duration
	p.Terms.ClosingTimestamp = 3
	err = nv.Start(context.Background(), p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampInvalid.Error())
}
