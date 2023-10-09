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
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/teams"
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

func parseMembersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
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
