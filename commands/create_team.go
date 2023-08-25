package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckCreateReferralSet(cmd *commandspb.CreateReferralSet) error {
	return checkCreateReferralSet(cmd).ErrorOrNil()
}

func checkCreateReferralSet(cmd *commandspb.CreateReferralSet) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("create_team", ErrIsRequired)
	}

	if cmd.IsTeam {
		// TODO: validate team fields
	}

	return errs
}
