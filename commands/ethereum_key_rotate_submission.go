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

package commands

import (
	"code.vegaprotocol.io/vega/libs/crypto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckEthereumKeyRotateSubmission(cmd *commandspb.EthereumKeyRotateSubmission) error {
	return checkEthereumKeyRotateSubmission(cmd).ErrorOrNil()
}

func checkEthereumKeyRotateSubmission(cmd *commandspb.EthereumKeyRotateSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("ethereum_key_rotate_submission", ErrIsRequired)
	}

	if len(cmd.NewAddress) <= 0 {
		errs.AddForProperty("ethereum_key_rotate_submission.new_address", ErrIsRequired)
	} else if !crypto.EthereumIsValidAddress(cmd.NewAddress) {
		errs.AddForProperty("ethereum_key_rotate_submission.new_address", ErrIsNotValidEthereumAddress)
	}

	if len(cmd.CurrentAddress) <= 0 {
		errs.AddForProperty("ethereum_key_rotate_submission.current_address", ErrIsRequired)
	} else if !crypto.EthereumIsValidAddress(cmd.CurrentAddress) {
		errs.AddForProperty("ethereum_key_rotate_submission.current_address", ErrIsNotValidEthereumAddress)
	}

	if cmd.TargetBlock == 0 {
		errs.AddForProperty("ethereum_key_rotate_submission.target_block", ErrIsRequired)
	}

	if cmd.EthereumSignature == nil {
		errs.AddForProperty("ethereum_key_rotate_submission.signature", ErrIsRequired)
	}

	if len(cmd.SubmitterAddress) != 0 && !crypto.EthereumIsValidAddress(cmd.SubmitterAddress) {
		errs.AddForProperty("ethereum_key_rotate_submission.submitter_address", ErrIsNotValidEthereumAddress)
	}

	return errs
}
