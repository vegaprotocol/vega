package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckCreateReferralSet(cmd *commandspb.CreateReferralSet) error {
	return checkCreateReferralSet(cmd).ErrorOrNil()
}

func checkCreateReferralSet(cmd *commandspb.CreateReferralSet) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("create_referral_set", ErrIsRequired)
	}

	if cmd.IsTeam {
		if cmd.Team == nil {
			return errs.FinalAddForProperty("create_referral_set.team", ErrIsRequired)
		}

		if len(cmd.Team.Name) <= 0 {
			errs.AddForProperty("create_referral_set.team.name", ErrIsRequired)
		} else if len(cmd.Team.Name) > 100 {
			errs.AddForProperty("create_referral_set.team.name", ErrMustBeLessThan100Chars)
		}

		if cmd.Team.AvatarUrl != nil && len(*cmd.Team.AvatarUrl) > 200 {
			errs.AddForProperty("create_referral_set.team.avatar_url", ErrMustBeLessThan200Chars)
		}

		if cmd.Team.TeamUrl != nil && len(*cmd.Team.TeamUrl) > 200 {
			errs.AddForProperty("create_referral_set.team.team_url", ErrMustBeLessThan200Chars)
		}
	}

	return errs
}
