package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckCreateTeam(cmd *commandspb.CreateTeam) error {
	return checkCreateTeam(cmd).ErrorOrNil()
}

func checkCreateTeam(cmd *commandspb.CreateTeam) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("create_team", ErrIsRequired)
	}

	return errs
}
