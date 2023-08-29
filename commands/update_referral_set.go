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
		if cmd.Team.Name != nil {
			if len(*cmd.Team.Name) <= 0 {
				errs.AddForProperty("update_referral_set.team.name", ErrIsRequired)
			} else if len(*cmd.Team.Name) > 100 {
				errs.AddForProperty("update_referral_set.team.name", ErrMustBeLessThan100Chars)
			}
		}

		if cmd.Team.AvatarUrl != nil && len(*cmd.Team.AvatarUrl) > 200 {
			errs.AddForProperty("update_referral_set.team.avatar_url", ErrMustBeLessThan200Chars)
		}

		if cmd.Team.TeamUrl != nil && len(*cmd.Team.TeamUrl) > 200 {
			errs.AddForProperty("update_referral_set.team.team_url", ErrMustBeLessThan200Chars)
		}
	}

	return errs
}
