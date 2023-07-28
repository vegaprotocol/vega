package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckUpdateTeam(cmd *commandspb.UpdateTeam) error {
	return checkUpdateTeam(cmd).ErrorOrNil()
}

func checkUpdateTeam(cmd *commandspb.UpdateTeam) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("update_team", ErrIsRequired)
	}

	if !IsVegaID(cmd.TeamId) {
		errs.AddForProperty("update_team.team_id", ErrShouldBeAValidVegaID)
	}

	return errs
}
