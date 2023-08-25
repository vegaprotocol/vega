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
			return errs.FinalAddForProperty("create_referral_set.team.name", ErrIsRequired)
		}
	}

	return errs
}
