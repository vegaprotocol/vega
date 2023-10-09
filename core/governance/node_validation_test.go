// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
