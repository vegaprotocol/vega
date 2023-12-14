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

package teams

import (
	"strings"

	"code.vegaprotocol.io/vega/core/types"

	"golang.org/x/exp/slices"
)

func (e *Engine) ListTeams() []types.Team {
	teams := make([]types.Team, 0, len(e.teams))

	for _, team := range e.teams {
		teams = append(teams, *team)
	}

	SortByTeamID(teams)

	return teams
}

func SortByTeamID(teamsToSort []types.Team) {
	slices.SortStableFunc(teamsToSort, func(a, b types.Team) int {
		return strings.Compare(string(a.ID), string(b.ID))
	})
}
