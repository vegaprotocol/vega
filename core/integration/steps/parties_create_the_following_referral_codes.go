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

package steps

import (
	"context"

	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/types"
	"github.com/cucumber/godog"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func PartiesCreateTheFollowingReferralCode(referralEngine *referral.Engine, teamsEngine *teams.Engine, table *godog.Table) error {
	ctx := context.Background()

	for _, r := range parseCreateReferralCodeTable(table) {
		row := newCreateReferralCodeRow(r)
		err := referralEngine.CreateReferralSet(ctx, row.Party(), row.Code())
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}

		if row.IsTeam() {
			team := &commandspb.CreateReferralSet_Team{
				Name: row.Team(),
			}

			err = teamsEngine.CreateTeam(ctx, row.Party(), types.TeamID(row.Team()), team)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseCreateReferralCodeTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"code",
	}, []string{
		"is_team",
		"team",
		"error",
		"reference",
	})
}

type createReferralCodeRow struct {
	row RowWrapper
}

func newCreateReferralCodeRow(r RowWrapper) createReferralCodeRow {
	row := createReferralCodeRow{
		row: r,
	}
	return row
}

func (r createReferralCodeRow) Party() types.PartyID {
	return types.PartyID(r.row.MustStr("party"))
}

func (r createReferralCodeRow) Code() types.ReferralSetID {
	return types.ReferralSetID(r.row.MustStr("code"))
}

func (r createReferralCodeRow) Error() string {
	return r.row.Str("error")
}

func (r createReferralCodeRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r createReferralCodeRow) Reference() string {
	return r.row.MustStr("reference")
}

func (r createReferralCodeRow) IsTeam() bool {
	if !r.row.HasColumn("is_team") {
		return false
	}
	return r.row.Bool("is_team")
}

func (r createReferralCodeRow) Team() string {
	return r.row.Str("team")
}
