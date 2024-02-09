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
	"code.vegaprotocol.io/vega/libs/crypto"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckIssueSignatures(cmd *commandspb.IssueSignatures) error {
	return checkIssueSignatures(cmd).ErrorOrNil()
}

func checkIssueSignatures(cmd *commandspb.IssueSignatures) Errors {
	errs := NewErrors()
	if cmd == nil {
		return errs.FinalAddForProperty("issue_signatures", ErrIsRequired)
	}

	if len(cmd.ValidatorNodeId) == 0 {
		errs.AddForProperty("issue_signatures.validator_node_id", ErrIsRequired)
	}

	if len(cmd.Submitter) == 0 {
		errs.AddForProperty("issue_signatures.submitter", ErrIsRequired)
	} else if !crypto.EthereumIsValidAddress(cmd.Submitter) {
		errs.AddForProperty("issue_signatures.submitter", ErrIsNotValidEthereumAddress)
	}

	if cmd.Kind != commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ERC20_MULTISIG_SIGNER_REMOVED &&
		cmd.Kind != commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ERC20_MULTISIG_SIGNER_ADDED {
		errs.AddForProperty("issue_signatures.kind", ErrIsNotValid)
	}

	return errs
}
