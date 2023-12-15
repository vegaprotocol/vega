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

const (
	listTeamsStatsQuery = `
WITH
  -- This CTE retrieves the all teams statistics reported for the last N epochs.
  windowed_teams_stats AS (
    SELECT *
    FROM teams_stats
    WHERE at_epoch > (
      SELECT MAX(at_epoch) - $1
      FROM teams_stats
    ) %s
  ),
  -- This CTE is used to determine at which epoch the teams have stats within
  -- the aggregation window.
  teams_per_epochs AS (
    SELECT team_id,
           at_epoch
    FROM windowed_teams_stats
    GROUP BY
      team_id,
      at_epoch
  ),
  -- This CTE filters the team that have exactly N epochs worth of statistics.
  -- We are exclusively computing the stats for teams that have at least N epochs worth of data.
  -- If we are looking at the stats for the last 30 epochs, a team that has less than 30 epochs
  -- worth of data aggregated, then it's ignored.
  eligible_teams AS (
    SELECT team_id
    FROM teams_per_epochs
    GROUP BY
      team_id
    HAVING COUNT(*) = $1
  ),
  eligible_stats AS (
    SELECT *
    FROM windowed_teams_stats
    WHERE team_id IN (
      SELECT *
      FROM eligible_teams
    )
  ),
  team_rewards AS (
    SELECT t.team_id,
           SUM(total_quantum_reward) AS total_in_quantum,
           JSONB_AGG(JSONB_BUILD_OBJECT('epoch', at_epoch, 'total', total_quantum_reward) ORDER BY at_epoch, total_quantum_reward) AS list
    FROM eligible_stats t
    GROUP BY
      t.team_id
  ),
  team_games AS (
    SELECT team_id,
           COALESCE(ARRAY_LENGTH(ARRAY_REMOVE(ARRAY_AGG(DISTINCT game_played), NULL), 1), 0) AS count,
           COALESCE(JSONB_AGG(DISTINCT game_played::BYTEA ORDER BY game_played::BYTEA)
                    FILTER (WHERE game_played <> 'null' ), '[]'::JSONB) AS LIST
    FROM eligible_stats stats
      LEFT JOIN LATERAL JSONB_OBJECT_KEYS(stats.games_played) AS game_played ON TRUE
    GROUP BY
      team_id
  )
SELECT mr.team_id AS team_id,
       mr.total_in_quantum AS total_quantum_rewards,
       mr.list AS quantum_rewards,
       mg.list AS games_played,
       mg.count AS total_games_played
FROM team_rewards mr
  LEFT JOIN team_games mg ON mr.team_id = mg.team_id
`

	listTeamMembersStatsQuery = `
WITH
  -- This CTE retrieves the all teams statistics reported for the last N epochs.
  windowed_teams_stats AS (
    SELECT *
    FROM teams_stats
    WHERE at_epoch > (
      SELECT MAX(at_epoch) - $1
      FROM teams_stats
    ) AND team_id = $2 %s
  ),
  -- This CTE is used to determine at which epoch the teams have stats within
  -- the aggregation window.
  teams_per_epochs AS (
    SELECT team_id,
           at_epoch
    FROM windowed_teams_stats
    GROUP BY
      team_id,
      at_epoch
  ),
  -- This CTE filters the team that have exactly N epochs worth of statistics.
  -- We are exclusively computing the stats for teams that have at least N epochs worth of data.
  -- If we are looking at the stats for the last 30 epochs, a team that has less than 30 epochs
  -- worth of data aggregated, then it's ignored.
  eligible_teams AS (
    SELECT team_id
    FROM teams_per_epochs
    GROUP BY
      team_id
    HAVING COUNT(*) = $1
  ),
  eligible_stats AS (
    SELECT *
    FROM windowed_teams_stats
    WHERE team_id IN (
      SELECT *
      FROM eligible_teams
    )
  ),
  members_rewards AS (
    SELECT team_id,
           party_id,
           SUM(total_quantum_reward) AS total_in_quantum,
           JSONB_AGG(JSONB_BUILD_OBJECT('epoch', at_epoch, 'total', total_quantum_reward)) AS quantum_rewards
    FROM eligible_stats
    GROUP BY
      team_id,
      party_id
  ),
  members_games AS (
    SELECT team_id,
           party_id,
           COALESCE(ARRAY_LENGTH(ARRAY_REMOVE(ARRAY_AGG(DISTINCT game_played), NULL), 1), 0) AS count,
           COALESCE(JSONB_AGG(DISTINCT game_played::BYTEA ORDER BY game_played::BYTEA)
                    FILTER ( WHERE game_played <> 'null' ), '[]'::JSONB) AS list
    FROM eligible_stats stats
      LEFT JOIN LATERAL JSONB_OBJECT_KEYS(stats.games_played) AS game_played ON TRUE
    GROUP BY
      team_id,
      party_id
  )
SELECT mr.party_id AS party_id,
       mr.total_in_quantum AS total_quantum_rewards,
       mr.quantum_rewards AS quantum_rewards,
       mg.list AS games_played,
       mg.count AS total_games_played
FROM members_rewards mr
  LEFT JOIN members_games mg ON mr.team_id = mg.team_id AND mr.party_id = mg.party_id`
)

type (
	Teams struct {
		*ConnectionSource
	}

	ListTeamsStatisticsFilters struct {
		TeamID            *entities.TeamID
		AggregationEpochs uint64
	}

	ListTeamMembersStatisticsFilters struct {
		TeamID            entities.TeamID
		PartyID           *entities.PartyID
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
	teamMembersStatsOrdering = TableOrdering{
		ColumnOrdering{Name: "party_id", Sorting: ASC},
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

	query := listTeamsStatsQuery
	args := []any{filters.AggregationEpochs}

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

func (t *Teams) ListTeamMembersStatistics(ctx context.Context, pagination entities.CursorPagination, filters ListTeamMembersStatisticsFilters) ([]entities.TeamMembersStatistics, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("Teams", "ListTeamMembersStatistics")()

	var (
		membersStats []entities.TeamMembersStatistics
		pageInfo     entities.PageInfo
	)

	query := listTeamMembersStatsQuery
	args := []any{filters.AggregationEpochs, filters.TeamID}

	if filters.PartyID != nil {
		query = fmt.Sprintf(query, fmt.Sprintf(`AND party_id = %s`, nextBindVar(&args, *filters.PartyID)))
	} else {
		query = fmt.Sprintf(query, "")
	}

	query, args, err := PaginateQuery[entities.TeamMemberStatisticsCursor](query, args, teamMembersStatsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, t.Connection, &membersStats, query, args...); err != nil {
		return nil, pageInfo, err
	}

	membersStats, pageInfo = entities.PageEntities[*v2.TeamMemberStatisticsEdge](membersStats, pagination)

	// Deserializing the GameID array as a PostgreSQL array is not correctly
	// interpreted by the scanny library. So, we have to use the JSONB array which
	// convert the bytea as strings. This leaves the prefix `\\x` on the game ID.
	// As a result, we have to manually clean up of the ID.
	for i := range membersStats {
		for j := range membersStats[i].GamesPlayed {
			membersStats[i].GamesPlayed[j] = entities.GameID(strings.TrimLeft(membersStats[i].GamesPlayed[j].String(), "\\x"))
		}
	}

	return membersStats, pageInfo, nil
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
