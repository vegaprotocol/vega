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

package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

const listTeamsStatsQuery = `
-- This CTE retrieves the teams statistics for the last N epochs.
WITH windowed_teams_stats AS (
    SELECT *
    FROM teams_stats
    WHERE at_epoch > (
      SELECT max(at_epoch) - $1
      FROM teams_stats
    ) %s -- This is where we filter the output either based on team ID or party ID. 
  ),
  -- This CTE filters the team that have exactly N epochs worth of statistics.
  -- We are exclusively computing the stats for teams that have at least N epochs worth of data.
  -- If we are looking at the stats for the last 30 epochs, a team that has less than 30 epochs
  -- worth of data aggregated, then it's ignored.
  eligible_teams AS (
    SELECT team_id,
           count(*) AS total_of_data_points
    FROM windowed_teams_stats
    GROUP BY
      team_id
    HAVING count(*) = $1
  )
SELECT eligible_teams.team_id AS team_id,
       rewards.total_in_quantum AS total_quantum_rewards,
       rewards.list AS quantum_rewards,
       COALESCE(games_played.count, 0) AS total_games_played,
       games_played.list AS games_played
FROM eligible_teams
  -- For each team ID in 'eligible_teams', we expand the JSON object keys from column 'games_played' to get a row
  -- for each game ID, that we deduplicate and aggregate into an array.
  INNER JOIN LATERAL (SELECT ARRAY_LENGTH(ARRAY_AGG(DISTINCT game_played), 1) AS count,
                             JSONB_AGG(DISTINCT game_played::bytea ORDER BY game_played::bytea) AS list
                      FROM windowed_teams_stats AS stats
                        -- That is the tricky part. For each rows matching 'team_id', we generate a row for each object's
                        -- key from the column 'games_played'. This allows us to flatten the object and effectively
                        -- deduplicate the game IDs before counting.
                        INNER JOIN LATERAL JSONB_OBJECT_KEYS(stats.games_played) AS game_played ON TRUE
                      WHERE eligible_teams.team_id = stats.team_id ) AS games_played ON TRUE
  -- For each team ID in 'eligible_teams', we compute the rewards from the statistics retrieved from the table
  -- 'windowed_teams_stats'.
  INNER JOIN LATERAL (
  SELECT SUM(total_quantum_reward) AS total_in_quantum,
         -- For each line before the aggregation, we build an object { epoch: total }, and group
         -- all these objects into an array.
         JSONB_AGG(JSONB_BUILD_OBJECT('epoch', stats.at_epoch, 'total', stats.total_quantum_reward)) AS list
  FROM windowed_teams_stats AS stats
  WHERE eligible_teams.team_id = stats.team_id ) AS rewards ON TRUE`

type (
	Teams struct {
		*ConnectionSource
	}

	ListTeamsStatisticsFilters struct {
		TeamID            *entities.TeamID
		AggregationEpochs uint64
	}
)

var (
	teamsOrdering = TableOrdering{
		ColumnOrdering{Name: "created_at", Sorting: ASC},
	}
	teamsStatsOrdering = TableOrdering{
		ColumnOrdering{Name: "team_id", Sorting: ASC},
	}
	refereesOrdering = TableOrdering{
		ColumnOrdering{Name: "party_id", Sorting: ASC},
	}
	refereeHistoryOrdering = TableOrdering{
		ColumnOrdering{Name: "joined_at_epoch", Sorting: ASC},
	}
)

func NewTeams(connectionSource *ConnectionSource) *Teams {
	return &Teams{
		ConnectionSource: connectionSource,
	}
}

func (t *Teams) AddTeam(ctx context.Context, team *entities.Team) error {
	defer metrics.StartSQLQuery("Teams", "AddTeam")()
	if _, err := t.Connection.Exec(
		ctx,
		"INSERT INTO teams(id, referrer, name, team_url, avatar_url, closed, created_at, created_at_epoch, vega_time) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
		team.ID,
		team.Referrer,
		team.Name,
		team.TeamURL,
		team.AvatarURL,
		team.Closed,
		team.CreatedAt,
		team.CreatedAtEpoch,
		team.VegaTime,
	); err != nil {
		return err
	}

	if _, err := t.Connection.Exec(
		ctx,
		"INSERT INTO team_members(team_id, party_id, joined_at_epoch, joined_at, vega_time) VALUES ($1, $2, $3, $4, $5)",
		team.ID,
		team.Referrer,
		team.CreatedAtEpoch,
		team.CreatedAt,
		team.VegaTime,
	); err != nil {
		return err
	}

	return nil
}

func (t *Teams) UpdateTeam(ctx context.Context, team *entities.TeamUpdated) error {
	defer metrics.StartSQLQuery("Teams", "UpdateTeam")()
	ct, err := t.Connection.Exec(ctx,
		`UPDATE teams
        SET name = $1,
            team_url = $2,
            avatar_url = $3,
            closed = $4
        WHERE id = $5`,
		team.Name,
		team.TeamURL,
		team.AvatarURL,
		team.Closed,
		team.ID,
	)

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("could not update team with id %s", team.ID)
	}
	return err
}

func (t *Teams) RefereeJoinedTeam(ctx context.Context, referee *entities.TeamMember) error {
	defer metrics.StartSQLQuery("Teams", "RefereeJoinedTeam")()
	_, err := t.Connection.Exec(ctx,
		`INSERT INTO team_members(team_id, party_id, joined_at, joined_at_epoch, vega_time) VALUES ($1, $2, $3, $4, $5)`,
		referee.TeamID,
		referee.PartyID,
		referee.JoinedAt,
		referee.JoinedAtEpoch,
		referee.VegaTime,
	)

	return err
}

func (t *Teams) RefereeSwitchedTeam(ctx context.Context, referee *entities.RefereeTeamSwitch) error {
	defer metrics.StartSQLQuery("Teams", "RefereeJoinedTeam")()

	_, err := t.Connection.Exec(ctx,
		`INSERT INTO team_members(team_id, party_id, joined_at, joined_at_epoch, vega_time) VALUES ($1, $2, $3, $4, $5)`,
		referee.ToTeamID,
		referee.PartyID,
		referee.SwitchedAt,
		referee.SwitchedAtEpoch,
		referee.VegaTime,
	)

	return err
}

func (t *Teams) GetTeam(ctx context.Context, teamID entities.TeamID, partyID entities.PartyID) (*entities.Team, error) {
	defer metrics.StartSQLQuery("Teams", "GetTeam")()

	var team entities.Team

	if teamID == "" && partyID == "" {
		return nil, fmt.Errorf("either teamID or partyID must be provided")
	}

	var args []interface{}

	var query string

	if teamID != "" {
		query = fmt.Sprintf("SELECT * FROM teams WHERE id = %s", nextBindVar(&args, teamID))
	} else if partyID != "" {
		query = fmt.Sprintf("SELECT t.* FROM teams t LEFT JOIN current_team_members ctm ON t.id = ctm.team_id WHERE ctm.party_id = %s", nextBindVar(&args, partyID))
	}

	if err := pgxscan.Get(ctx, t.Connection, &team, query, args...); err != nil {
		return nil, err
	}

	return &team, nil
}

func (t *Teams) ListTeams(ctx context.Context, pagination entities.CursorPagination) ([]entities.Team, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Teams", "ListTeams")()

	var (
		teams    []entities.Team
		args     []interface{}
		pageInfo entities.PageInfo
	)

	query := `SELECT * FROM teams`

	query, args, err := PaginateQuery[entities.TeamCursor](query, args, teamsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, t.Connection, &teams, query, args...); err != nil {
		return nil, pageInfo, err
	}

	teams, pageInfo = entities.PageEntities[*v2.TeamEdge](teams, pagination)

	return teams, pageInfo, nil
}

func (t *Teams) ListTeamsStatistics(ctx context.Context, pagination entities.CursorPagination, filters ListTeamsStatisticsFilters) ([]entities.TeamsStatistics, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Teams", "ListTeamsStatistics")()

	var (
		teamsStats []entities.TeamsStatistics
		pageInfo   entities.PageInfo
	)

	args := []any{filters.AggregationEpochs}

	query := listTeamsStatsQuery
	if filters.TeamID != nil {
		query = fmt.Sprintf(query, fmt.Sprintf(`AND team_id = %s`, nextBindVar(&args, *filters.TeamID)))
	} else {
		query = fmt.Sprintf(query, "")
	}

	query, args, err := PaginateQuery[entities.TeamsStatisticsCursor](query, args, teamsStatsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, t.Connection, &teamsStats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	teamsStats, pageInfo = entities.PageEntities[*v2.TeamStatisticsEdge](teamsStats, pagination)

	// Deserializing the GameID array as a PostgreSQL array is not correctly
	// interpreted by the scanny library. So, we have to use the JSONB array which
	// convert the bytea as strings. This leaves the prefix `\\x` on the game ID.
	// As a result, we have to manually clean up of the ID.
	for i := range teamsStats {
		for j := range teamsStats[i].GamesPlayed {
			teamsStats[i].GamesPlayed[j] = entities.GameID(strings.TrimLeft(teamsStats[i].GamesPlayed[j].String(), "\\x"))
		}
	}

	return teamsStats, pageInfo, nil
}

func (t *Teams) ListReferees(ctx context.Context, teamID entities.TeamID, pagination entities.CursorPagination) ([]entities.TeamMember, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Teams", "ListReferees")()
	var (
		referees []entities.TeamMember
		args     []interface{}
		pageInfo entities.PageInfo
	)

	if teamID == "" {
		return nil, pageInfo, fmt.Errorf("teamID must be provided")
	}

	query := `SELECT ctm.*
	FROM current_team_members ctm
    LEFT JOIN teams t ON t.id = ctm.team_id
	 WHERE ctm.party_id != t.referrer AND ctm.team_id = %s`

	query = fmt.Sprintf(query, nextBindVar(&args, teamID))

	query, args, err := PaginateQuery[entities.RefereeCursor](query, args, refereesOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, t.Connection, &referees, query, args...); err != nil {
		return nil, pageInfo, err
	}

	referees, pageInfo = entities.PageEntities[*v2.TeamRefereeEdge](referees, pagination)

	return referees, pageInfo, nil
}

func (t *Teams) ListRefereeHistory(ctx context.Context, referee entities.PartyID, pagination entities.CursorPagination) ([]entities.TeamMemberHistory, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Teams", "ListRefereeHistory")()
	var (
		referees []entities.TeamMemberHistory
		args     []interface{}
		pageInfo entities.PageInfo
	)

	if referee == "" {
		return nil, pageInfo, fmt.Errorf("referee must be provided")
	}

	query := fmt.Sprintf(`SELECT team_id, joined_at_epoch, joined_at FROM team_members WHERE party_id = %s`, nextBindVar(&args, referee))

	query, args, err := PaginateQuery[entities.RefereeHistoryCursor](query, args, refereeHistoryOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, t.Connection, &referees, query, args...); err != nil {
		return nil, pageInfo, err
	}

	referees, pageInfo = entities.PageEntities[*v2.TeamRefereeHistoryEdge](referees, pagination)

	return referees, pageInfo, nil
}
