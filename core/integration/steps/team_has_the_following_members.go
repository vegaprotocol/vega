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
