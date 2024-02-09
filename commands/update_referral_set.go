// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

		if cmd.Team.Closed == nil && len(cmd.Team.AllowList) > 0 {
			errs.AddForProperty("update_referral_set.team.allow_list", ErrSettingAllowListRequireSettingClosedState)
		}

		if cmd.Team.Closed != nil && !*cmd.Team.Closed && len(cmd.Team.AllowList) > 0 {
			errs.AddForProperty("update_referral_set.team.allow_list", ErrCannotSetAllowListWhenTeamIsOpened)
		}
	}

	return errs
}
