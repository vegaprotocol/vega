package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckUpdateReferralSet(cmd *commandspb.UpdateReferralSet) error {
	return checkUpdateReferralSet(cmd).ErrorOrNil()
}

func checkUpdateReferralSet(cmd *commandspb.UpdateReferralSet) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("update_team", ErrIsRequired)
	}

	if !IsVegaID(cmd.Id) {
		errs.AddForProperty("update_team.team_id", ErrShouldBeAValidVegaID)
	}

	if cmd.IsTeam {
		// TODO: validate team
	}

	return errs
}
