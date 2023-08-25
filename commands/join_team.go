package commands

import commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

func CheckApplyReferralCode(cmd *commandspb.ApplyReferralCode) error {
	return checkApplyReferralCode(cmd).ErrorOrNil()
}

func checkApplyReferralCode(cmd *commandspb.ApplyReferralCode) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("join_team", ErrIsRequired)
	}

	if !IsVegaID(cmd.TeamId) {
		errs.AddForProperty("join_team.team_id", ErrShouldBeAValidVegaID)
	}

	return errs
}
