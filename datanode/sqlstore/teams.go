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
	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/georgysavva/scany/pgxscan"
)

const (
	listTeamsStatsQuery = `WITH
  -- This CTE retrieves the all teams statistics reported for the last N epochs.
  eligible_stats AS (
    SELECT *
    FROM teams_stats
    WHERE at_epoch > (
      SELECT MAX(id) - $1
      FROM epochs
    ) %s
  ),
  team_numbers AS (
    SELECT t.team_id,
           SUM(total_quantum_reward) AS total_quantum_rewards,
           JSONB_AGG(JSONB_BUILD_OBJECT('epoch', at_epoch, 'total', total_quantum_reward) ORDER BY at_epoch, total_quantum_reward) AS quantum_rewards,
           SUM(total_quantum_volume) AS total_quantum_volumes,
           JSONB_AGG(JSONB_BUILD_OBJECT('epoch', at_epoch, 'total', total_quantum_volume) ORDER BY at_epoch, total_quantum_volume) AS quantum_volumes
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
SELECT tn.team_id AS team_id,
       tn.total_quantum_rewards AS total_quantum_rewards,
       tn.quantum_rewards AS quantum_rewards,
       tn.total_quantum_volumes AS total_quantum_volumes,
       tn.quantum_volumes AS quantum_volumes,
       mg.list AS games_played,
       mg.count AS total_games_played
FROM team_numbers tn
  LEFT JOIN team_games mg ON tn.team_id = mg.team_id
`

	listTeamMembersStatsQuery = `WITH
  -- This CTE retrieves the all teams statistics reported for the last N epochs.
  eligible_stats AS (
    SELECT *
    FROM teams_stats
    WHERE at_epoch > (
      SELECT MAX(id) - $1
      FROM epochs
    ) AND team_id = $2 %s
  ),
  members_numbers AS (
    SELECT team_id,
           party_id,
           SUM(total_quantum_reward) AS total_quantum_rewards,
           JSONB_AGG(JSONB_BUILD_OBJECT('epoch', at_epoch, 'total', total_quantum_reward) ORDER BY at_epoch, total_quantum_reward) AS quantum_rewards,
           SUM(total_quantum_volume) AS total_quantum_volumes,
           JSONB_AGG(JSONB_BUILD_OBJECT('epoch', at_epoch, 'total', total_quantum_volume) ORDER BY at_epoch, total_quantum_volume) AS quantum_volumes
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
SELECT mn.party_id AS party_id,
       mn.total_quantum_rewards AS total_quantum_rewards,
       mn.quantum_rewards AS quantum_rewards,
       mn.total_quantum_volumes AS total_quantum_volumes,
       mn.quantum_volumes AS quantum_volumes,
       mg.list AS games_played,
       mg.count AS total_games_played
FROM members_numbers mn
  LEFT JOIN members_games mg ON mn.team_id = mg.team_id AND mn.party_id = mg.party_id`

	upsertTeamsStats = `INSERT INTO teams_stats(team_id, party_id, at_epoch, total_quantum_volume, total_quantum_reward, games_played)
VALUES
  %s
ON CONFLICT (team_id, party_id, at_epoch) DO UPDATE
  SET total_quantum_volume = excluded.total_quantum_volume
	`
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

	if team.AllowList == nil {
		team.AllowList = []string{}
	}

	if _, err := t.Exec(
		ctx,
		"INSERT INTO teams(id, referrer, name, team_url, avatar_url, closed, allow_list, created_at, created_at_epoch, vega_time) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
		team.ID,
		team.Referrer,
		team.Name,
		team.TeamURL,
		team.AvatarURL,
		team.Closed,
		team.AllowList,
		team.CreatedAt,
		team.CreatedAtEpoch,
		team.VegaTime,
	); err != nil {
		return err
	}

	// in case the party already was in a team?
	_, _ = t.Exec(
		ctx,
		"DELETE FROM team_members WHERE party_id = $1",
		team.Referrer,
	)

	if _, err := t.Exec(
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

	if team.AllowList == nil {
		team.AllowList = []string{}
	}

	ct, err := t.Exec(ctx,
		`UPDATE teams
        SET name = $1,
            team_url = $2,
            avatar_url = $3,
            closed = $4,
            allow_list = $5
        WHERE id = $6`,
		team.Name,
		team.TeamURL,
		team.AvatarURL,
		team.Closed,
		team.AllowList,
		team.ID,
	)

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("could not update team with id %s", team.ID)
	}
	return err
}

func (t *Teams) RefereeJoinedTeam(ctx context.Context, referee *entities.TeamMember) error {
	defer metrics.StartSQLQuery("Teams", "RefereeJoinedTeam")()
	_, err := t.Exec(ctx,
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
	defer metrics.StartSQLQuery("Teams", "RefereeSwitchedTeam")()

	// in case the party was removed from the team owner from a team
	if len(referee.ToTeamID) <= 0 {
		_, err := t.Exec(
			ctx,
			"DELETE FROM team_members WHERE party_id = $1",
			referee.PartyID,
		)

		return err
	}

	// normal path, team_members just being updated.
	_, err := t.Exec(ctx,
		`INSERT INTO team_members(team_id, party_id, joined_at, joined_at_epoch, vega_time) VALUES ($1, $2, $3, $4, $5)`,
		referee.ToTeamID,
		referee.PartyID,
		referee.SwitchedAt,
		referee.SwitchedAtEpoch,
		referee.VegaTime,
	)

	return err
}

func (t *Teams) TeamsStatsUpdated(ctx context.Context, evt *eventspb.TeamsStatsUpdated) error {
	defer metrics.StartSQLQuery("Teams", "TeamsStatsUpdated")()

	var args []interface{}

	values := []string{}
	for _, teamStats := range evt.Stats {
		for _, memberStats := range teamStats.MembersStats {
			notionalVolume, hasErr := num.UintFromString(memberStats.NotionalVolume, 10)
			if hasErr {
				notionalVolume = num.UintZero()
			}

			values = append(values, fmt.Sprintf("(%s, %s, %s, %s, 0, '{}'::JSONB)",
				nextBindVar(&args, entities.TeamID(teamStats.TeamId)),
				nextBindVar(&args, entities.PartyID(memberStats.PartyId)),
				nextBindVar(&args, evt.AtEpoch),
				nextBindVar(&args, notionalVolume)),
			)
		}
	}

	if len(values) == 0 {
		return nil
	}

	query := fmt.Sprintf(upsertTeamsStats, strings.Join(values, ","))
	_, err := t.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("could not insert team stats update: %w", err)
	}

	return nil
}

func (t *Teams) GetTeam(ctx context.Context, teamID entities.TeamID, partyID entities.PartyID) (*entities.Team, error) {
	defer metrics.StartSQLQuery("Teams", "GetTeam")()

	var team entities.Team

	if teamID == "" && partyID == "" {
		return nil, fmt.Errorf("either teamID or partyID must be provided")
	}

	var args []interface{}

	query := `WITH
  members_stats AS (
    SELECT team_id, COUNT(DISTINCT party_id) AS total_members
    FROM current_team_members
    GROUP BY
      team_id
  )
SELECT teams.*, members_stats.total_members
FROM teams
  LEFT JOIN members_stats on teams.id = members_stats.team_id %s`

	var where string
	if teamID != "" {
		where = fmt.Sprintf("WHERE teams.id = %s", nextBindVar(&args, teamID))
	} else if partyID != "" {
		where = fmt.Sprintf("INNER JOIN current_team_members ON current_team_members.party_id = %s AND teams.id = current_team_members.team_id", nextBindVar(&args, partyID))
	}

	query = fmt.Sprintf(query, where)

	if err := pgxscan.Get(ctx, t.ConnectionSource, &team, query, args...); err != nil {
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

	query := `WITH
  members_stats AS (
    SELECT team_id, COUNT(DISTINCT party_id) AS total_members
    FROM current_team_members
    GROUP BY
      team_id
  )
SELECT teams.*, members_stats.total_members
FROM teams
  LEFT JOIN members_stats on teams.id = members_stats.team_id`
	query, args, err := PaginateQuery[entities.TeamCursor](query, args, teamsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, t.ConnectionSource, &teams, query, args...); err != nil {
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

	if err := pgxscan.Select(ctx, t.ConnectionSource, &teamsStats, query, args...); err != nil {
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

	if err := pgxscan.Select(ctx, t.ConnectionSource, &membersStats, query, args...); err != nil {
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

	if err := pgxscan.Select(ctx, t.ConnectionSource, &referees, query, args...); err != nil {
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

	if err := pgxscan.Select(ctx, t.ConnectionSource, &referees, query, args...); err != nil {
		return nil, pageInfo, err
	}

	referees, pageInfo = entities.PageEntities[*v2.TeamRefereeHistoryEdge](referees, pagination)

	return referees, pageInfo, nil
}
