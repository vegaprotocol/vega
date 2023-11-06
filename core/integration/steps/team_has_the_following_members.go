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
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/teams"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/cucumber/godog"
)

func TheTeamHasTheFollowingMembers(teamsEngine *teams.Engine, team string, table *godog.Table) error {
	// Get a list of all the parties in a team
	members := teamsEngine.GetTeamMembers(team, 0)
	rows := []string{}
	for _, r := range parseMembersTable(table) {
		row := newMembersRow(r)
		rows = append(rows, row.Party())
	}
	sort.Strings(members)
	sort.Strings(rows)

	// Do we have the same amount of parties in each list?
	if len(members) != len(rows) {
		return fmt.Errorf("different number of team members between the table (%d) and the engine (%d)", len(rows), len(members))
	}

	// Do we have the same members in each list?
	for i, r := range members {
		if r != rows[i] {
			return fmt.Errorf("party details are different (%s != %s)", r, rows[i])
		}
	}
	return nil
}

func TheFollowingTeamsWithRefereesAreCreated(
	col *collateral.Engine,
	broker *stubs.BrokerStub,
	netDeposits *num.Uint,
	referralEngine *referral.Engine,
	teamsEngine *teams.Engine,
	table *godog.Table,
) error {
	ctx := context.Background()
	for _, r := range parseCreateTeamTable(table) {
		row := teamRow{
			r: r,
		}
		asset := row.Asset()
		balance := row.Balance()
		parties := row.Members()
		evts := make([]events.Event, 0, len(parties))
		// 1. ensure deposits are made
		for _, pid := range parties {
			res, err := col.Deposit(
				ctx,
				pid,
				asset,
				balance.Clone(),
			)
			if err != nil {
				return err
			}
			evts = append(evts, events.NewLedgerMovements(ctx, []*types.LedgerMovement{res}))
			// increase overal deposits by the balance added
			netDeposits.AddSum(balance)
		}
		broker.SendBatch(evts)
		// 2. Now create the referral code
		code, team, referrer := types.ReferralSetID(row.Code()), row.Team(), types.PartyID(row.Referrer())
		if err := referralEngine.CreateReferralSet(ctx, referrer, code); err != nil {
			return err
		}
		// 3. Create a team
		teamPB := &commandspb.CreateReferralSet_Team{
			Name: team,
		}
		if err := teamsEngine.CreateTeam(ctx, referrer, types.TeamID(team), teamPB); err != nil {
			return err
		}
		// 4. All parties apply the referral code, skip the first in parties slice, they are the referrer
		refCode := &commandspb.ApplyReferralCode{
			Id: team,
		}
		// 5. Join team
		for _, pid := range parties[1:] {
			if err := referralEngine.ApplyReferralCode(ctx, types.PartyID(pid), code); err != nil {
				return err
			}
			if err := teamsEngine.JoinTeam(ctx, types.PartyID(pid), refCode); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseMembersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
	}, []string{})
}

func parseCreateTeamTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"referrer",
		"prefix",
		"code",
		"team name",
		"referees",
		"balance",
		"asset",
	}, []string{})
}

type membersRow struct {
	row RowWrapper
}

func newMembersRow(r RowWrapper) membersRow {
	row := membersRow{
		row: r,
	}
	return row
}

func (r membersRow) Party() string {
	return r.row.MustStr("party")
}

type teamRow struct {
	r RowWrapper
}

func (t teamRow) Referrer() string {
	return t.r.MustStr("referrer")
}

func (t teamRow) Code() string {
	return t.r.MustStr("code")
}

func (t teamRow) Team() string {
	return t.r.MustStr("team name")
}

func (t teamRow) Asset() string {
	return t.r.MustStr("asset")
}

func (t teamRow) Prefix() string {
	return t.r.MustStr("prefix")
}

func (t teamRow) MemberCount() int {
	return int(t.r.MustU32("referees"))
}

func (t teamRow) Balance() *num.Uint {
	return t.r.MustUint("balance")
}

func (t teamRow) Members() []string {
	cnt := t.MemberCount()
	ids := make([]string, 0, cnt)
	ids = append(ids, t.Referrer())
	pidFmt := fmt.Sprintf("%s-%%04d", t.Prefix())
	for i := 0; i < cnt; i++ {
		ids = append(ids, fmt.Sprintf(pidFmt, i+1))
	}
	return ids
}
