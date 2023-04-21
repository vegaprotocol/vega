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
