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
	"strconv"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckStateVariableProposal(cmd *commandspb.StateVariableProposal) error {
	return checkStateVariableProposal(cmd).ErrorOrNil()
}

func checkStateVariableProposal(cmd *commandspb.StateVariableProposal) Errors {
	errs := NewErrors()
	if cmd == nil {
		return errs.FinalAddForProperty("state_variable_proposal", ErrIsRequired)
	}

	if cmd.Proposal == nil {
		return errs.FinalAddForProperty("state_variable_proposal.missing_proposal", ErrIsRequired)
	}

	if len(cmd.Proposal.EventId) == 0 {
		errs.AddForProperty("state_variable_proposal.event_id", ErrIsRequired)
	}
	if len(cmd.Proposal.StateVarId) == 0 {
		errs.AddForProperty("state_variable_proposal.state_var_id", ErrIsRequired)
	}
	if len(cmd.Proposal.Kvb) == 0 {
		errs.AddForProperty("state_variable_proposal.key_value_bundle", ErrIsRequired)
	}

	for i, kvb := range cmd.Proposal.Kvb {
		if len(kvb.Key) == 0 {
			errs.AddForProperty("state_variable_proposal.key_value_bundle."+strconv.Itoa(i)+".key", ErrIsRequired)
		}

		if kvb.Value == nil {
			errs.AddForProperty("state_variable_proposal.key_value_bundle."+strconv.Itoa(i)+".value", ErrIsRequired)
		}
	}
	return errs
}
