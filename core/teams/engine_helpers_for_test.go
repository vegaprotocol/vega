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
