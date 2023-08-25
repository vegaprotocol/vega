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
		errs.AddForProperty("update_referral_set.id", ErrShouldBeAValidVegaID)
	}

	if cmd.IsTeam {
		if cmd.Team == nil {
			return errs.FinalAddForProperty("update_referral_set.team", ErrIsRequired)
		}

		// now the only one which needs validation again is the name, as it's not allowed to be set to ""
		if cmd.Team.Name != nil && len(*cmd.Team.Name) <= 0 {
			return errs.FinalAddForProperty("update_referral_set.team.name", ErrIsRequired)
		}

	}

	return errs
}
