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

func getTestNodeValidation(t *testing.T, tm time.Time) *testNodeValidation {
	t.Helper()
	ctrl := gomock.NewController(t)
	assets := mocks.NewMockAssets(ctrl)
	witness := mocks.NewMockWitness(ctrl)

	nv := governance.NewNodeValidation(
		logging.NewTestLogger(), assets, tm, witness)
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
	nv := getTestNodeValidation(t, time.Now())
	defer nv.ctrl.Finish()

	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTermsNewAsset{},
		},
	}

	assert.True(t, nv.IsNodeValidationRequired(p))
}

func testNodeValidationRequiredFalse(t *testing.T) {
	nv := getTestNodeValidation(t, time.Now())
	defer nv.ctrl.Finish()

	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTermsNewMarket{},
		},
	}

	assert.False(t, nv.IsNodeValidationRequired(p))
}

func testStartErrorNoNodeValidationRequired(t *testing.T) {
	nv := getTestNodeValidation(t, time.Now())
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
	tm := time.Now()
	nv := getTestNodeValidation(t, tm)
	defer nv.ctrl.Finish()

	// first closing time < validation time
	p := &types.Proposal{
		Terms: &types.ProposalTerms{
			ClosingTimestamp:    tm.Add(1 * time.Hour).Unix(),
			ValidationTimestamp: tm.Add(2 * time.Hour).Unix(),
			Change:              &types.ProposalTermsNewAsset{},
		},
	}

	err := nv.Start(context.Background(), p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampTooLate.Error())

	// validation timestamp after 2 days
	p.Terms.ClosingTimestamp = tm.Add(3 * 24 * time.Hour).Unix()
	p.Terms.ValidationTimestamp = tm.Add(2*24*time.Hour + 1*time.Second).Unix()
	err = nv.Start(context.Background(), p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampOutsideRange.Error())

	// validation timestamp = submission time
	p.Terms.ValidationTimestamp = tm.Unix()
	err = nv.Start(context.Background(), p)
	assert.EqualError(t, err, governance.ErrProposalValidationTimestampOutsideRange.Error())

	// all good
	nv.assets.EXPECT().NewAsset(gomock.Any(), gomock.Any(), gomock.Any())
	nv.witness.EXPECT().StartCheck(gomock.Any(), gomock.Any(), gomock.Any())
	p.Terms.ValidationTimestamp = tm.Add(1 * 24 * time.Hour).Unix()
	err = nv.Start(context.Background(), p)
	assert.NoError(t, err)
}
