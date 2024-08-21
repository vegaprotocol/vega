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

import (
	"errors"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckCreateReferralSet(cmd *commandspb.CreateReferralSet) error {
	return checkCreateReferralSet(cmd).ErrorOrNil()
}

func checkCreateReferralSet(cmd *commandspb.CreateReferralSet) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("create_referral_set", ErrIsRequired)
	}

	// Basically this command should be rejected if we are not creating a team
	// but also not creating a referral set...
	// just check if this command is ineffective...
	if cmd.DoNotCreateReferralSet && !cmd.IsTeam {
		return errs.FinalAddForProperty("create_referral_set",
			errors.New("is ineffective"))
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

		if !cmd.Team.Closed && len(cmd.Team.AllowList) > 0 {
			errs.AddForProperty("create_referral_set.team.allow_list", ErrCannotSetAllowListWhenTeamIsOpened)
		}
	}

	return errs
}
