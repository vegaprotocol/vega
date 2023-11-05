// Copyright (C) 2023  Gobalsky Labs Limited
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

package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestCheckProposalSubmissionForNewFreeform(t *testing.T) {
	t.Run("Submitting a new freeform change without new freeform fails", testNewFreeformChangeSubmissionWithoutNewFreeformFails)
	t.Run("Submitting a new freeform proposal without rational URL and hash fails", testNewFreeformProposalSubmissionWithoutRationalURLandHashFails)
}

func testNewFreeformChangeSubmissionWithoutNewFreeformFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewFreeform{},
		},
	})

	assert.Contains(t, err.Get("proposal_submission.terms.change.new_freeform"), commands.ErrIsRequired)
}

func testNewFreeformProposalSubmissionWithoutRationalURLandHashFails(t *testing.T) {
	err := checkProposalSubmission(&commandspb.ProposalSubmission{
		Terms: &types.ProposalTerms{
			Change: &types.ProposalTerms_NewFreeform{},
		},
		Rationale: &types.ProposalRationale{},
	})

	assert.Contains(t, err.Get("proposal_submission.rationale.description"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("proposal_submission.rationale.title"), commands.ErrIsRequired)
}
