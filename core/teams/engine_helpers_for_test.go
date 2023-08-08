// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package teams

import (
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
	slices.SortStableFunc(teamsToSort, func(a, b types.Team) bool {
		return a.ID < b.ID
	})
}
