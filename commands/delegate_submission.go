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
	"math/big"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckDelegateSubmission(cmd *commandspb.DelegateSubmission) error {
	return checkDelegateSubmission(cmd).ErrorOrNil()
}

func checkDelegateSubmission(cmd *commandspb.DelegateSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("delegate_submission", ErrIsRequired)
	}

	if len(cmd.Amount) <= 0 {
		errs.AddForProperty("delegate_submission.amount", ErrIsRequired)
	} else {
		if amount, ok := big.NewInt(0).SetString(cmd.Amount, 10); !ok {
			errs.AddForProperty("delegate_submission.amount", ErrNotAValidInteger)
		} else {
			if amount.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("delegate_submission.amount", ErrIsRequired)
			}
		}
	}

	if len(cmd.NodeId) <= 0 {
		errs.AddForProperty("delegate_submission.node_id", ErrIsRequired)
	} else if !IsVegaPublicKey(cmd.NodeId) {
		errs.AddForProperty("delegate_submission.node_id", ErrShouldBeAValidVegaPublicKey)
	}

	return errs
}

func CheckUndelegateSubmission(cmd *commandspb.UndelegateSubmission) error {
	return checkUndelegateSubmission(cmd).ErrorOrNil()
}

func checkUndelegateSubmission(cmd *commandspb.UndelegateSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("undelegate_submission", ErrIsRequired)
	}

	if _, ok := commandspb.UndelegateSubmission_Method_value[cmd.Method.String()]; !ok || cmd.Method == commandspb.UndelegateSubmission_METHOD_UNSPECIFIED {
		errs.AddForProperty("undelegate_submission.method", ErrIsRequired)
	}

	if len(cmd.NodeId) <= 0 {
		errs.AddForProperty("undelegate_submission.node_id", ErrIsRequired)
	} else if !IsVegaPublicKey(cmd.NodeId) {
		errs.AddForProperty("undelegate_submission.node_id", ErrShouldBeAValidVegaPublicKey)
	}

	return errs
}
