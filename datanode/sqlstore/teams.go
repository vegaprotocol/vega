package sqlstore

import (
	"context"
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/metrics"

	"code.vegaprotocol.io/vega/datanode/entities"
)

type (
	Teams struct {
		*ConnectionSource
	}
)

var (
	teamsOrdering = TableOrdering{
		ColumnOrdering{Name: "created_at", Sorting: ASC},
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
		"INSERT INTO teams(id, referrer, name, team_url, avatar_url, closed, created_at, created_at_epoch, vega_time) values ($1, $2, $3, $4, $5, $6, $7, $8, $9)",
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
		"INSERT INTO team_members(team_id, party_id, joined_at_epoch, joined_at, vega_time) values ($1, $2, $3, $4, $5)",
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
        where id = $5`,
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
		`INSERT INTO team_members(team_id, party_id, joined_at, joined_at_epoch, vega_time) values ($1, $2, $3, $4, $5)`,
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
		`INSERT INTO team_members(team_id, party_id, joined_at, joined_at_epoch, vega_time) values ($1, $2, $3, $4, $5)`,
		referee.ToTeamID,
		referee.PartyID,
		referee.SwitchedAt,
		referee.SwitchedAtEpoch,
		referee.VegaTime,
	)

	return err
}

func (t *Teams) GetTeam(ctx context.Context, teamID entities.TeamID, partyID entities.PartyID) (entities.Team, error) {
	defer metrics.StartSQLQuery("Teams", "GetTeam")()

	var team entities.Team

	if teamID == "" && partyID == "" {
		return team, fmt.Errorf("either teamID or partyID must be provided")
	}

	var args []interface{}

	var query string

	if teamID != "" {
		query = fmt.Sprintf("select * from teams where id = %s", nextBindVar(&args, teamID))
	} else if partyID != "" {
		query = fmt.Sprintf("select t.* from teams t left join current_team_members ctm on t.id = ctm.team_id where ctm.party_id = %s", nextBindVar(&args, partyID))
	}

	if err := pgxscan.Get(ctx, t.Connection, &team, query, args...); err != nil {
		return team, err
	}

	return team, nil
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

	query := `select ctm.*
	from current_team_members ctm
    left join teams t on t.id = ctm.team_id
	 where ctm.party_id != t.referrer and ctm.team_id = %s`

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
