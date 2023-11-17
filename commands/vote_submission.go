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

package commands

import (
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckVoteSubmission(cmd *commandspb.VoteSubmission) error {
	return checkVoteSubmission(cmd).ErrorOrNil()
}

func checkVoteSubmission(cmd *commandspb.VoteSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("vote_submission", ErrIsRequired)
	}

	if len(cmd.ProposalId) <= 0 {
		errs.AddForProperty("vote_submission.proposal_id", ErrIsRequired)
	} else if !IsVegaID(cmd.ProposalId) {
		errs.AddForProperty("vote_submission.proposal_id", ErrShouldBeAValidVegaID)
	}

	if cmd.Value == types.Vote_VALUE_UNSPECIFIED {
		errs.AddForProperty("vote_submission.value", ErrIsRequired)
	}

	if _, ok := types.Vote_Value_name[int32(cmd.Value)]; !ok {
		errs.AddForProperty("vote_submission.value", ErrIsNotValid)
	}

	return errs
}
