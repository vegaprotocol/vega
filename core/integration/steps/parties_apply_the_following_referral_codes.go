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
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/cucumber/godog"
)

func PartiesApplyTheFollowingReferralCode(referralEngine *referral.Engine, teamsEngine *teams.Engine, table *godog.Table) error {
	ctx := context.Background()

	for _, r := range parseApplyReferralCodeTable(table) {
		row := newApplyReferralCodeRow(r)
		err := referralEngine.ApplyReferralCode(ctx, row.Party(), row.Code())
		if checkErr := checkExpectedError(row, err, nil); checkErr != nil {
			if !row.IsTeam() {
				return checkErr
			}
			err = checkErr
		}
		// If we have team details, submit a new team
		if row.IsTeam() {
			team := &commandspb.JoinTeam{
				Id: row.Team(),
			}
			if joinErr := teamsEngine.JoinTeam(ctx, row.Party(), team); joinErr != nil {
				err = checkExpectedError(row, joinErr, nil)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func parseApplyReferralCodeTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"code",
	}, []string{
		"error",
		"reference",
		"is_team",
		"team",
	})
}

type applyReferralCodeRow struct {
	row RowWrapper
}

func newApplyReferralCodeRow(r RowWrapper) applyReferralCodeRow {
	row := applyReferralCodeRow{
		row: r,
	}
	return row
}

func (r applyReferralCodeRow) Party() types.PartyID {
	return types.PartyID(r.row.MustStr("party"))
}

func (r applyReferralCodeRow) Code() types.ReferralSetID {
	return types.ReferralSetID(r.row.MustStr("code"))
}

func (r applyReferralCodeRow) Error() string {
	return r.row.Str("error")
}

func (r applyReferralCodeRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r applyReferralCodeRow) Reference() string {
	return r.row.MustStr("reference")
}

func (r applyReferralCodeRow) IsTeam() bool {
	if !r.row.HasColumn("is_team") {
		return false
	}
	return r.row.Bool("is_team")
}

func (r applyReferralCodeRow) Team() string {
	return r.row.Str("team")
}
