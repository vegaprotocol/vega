package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckJoinTeam(cmd *commandspb.JoinTeam) error {
	return checkJoinTeam(cmd).ErrorOrNil()
}

func checkJoinTeam(cmd *commandspb.JoinTeam) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("join_team", ErrIsRequired)
	}

	if !IsVegaPubkey(cmd.TeamId) {
		errs.AddForProperty("join_team.team_id", ErrShouldBeAValidVegaID)
	}

	return errs
}
