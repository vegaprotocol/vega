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
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/shopspring/decimal"
)

type GameScores struct {
	*ConnectionSource
}

var gamesTeamOrderding = TableOrdering{
	ColumnOrdering{Name: "game_id", Sorting: ASC},
	ColumnOrdering{Name: "team_id", Sorting: ASC},
}

var gamesPartyOrderding = TableOrdering{
	ColumnOrdering{Name: "game_id", Sorting: ASC},
	ColumnOrdering{Name: "party_id", Sorting: ASC},
}

func NewGameScores(connectionSource *ConnectionSource) *GameScores {
	r := &GameScores{
		ConnectionSource: connectionSource,
	}
	return r
}

func (gs *GameScores) AddPartyScore(ctx context.Context, r entities.GamePartyScore) error {
	defer metrics.StartSQLQuery("GameScores", "AddPartyScores")()
	_, err := gs.Connection.Exec(ctx,
		`INSERT INTO game_party_scores(
			game_id,
			team_id,
			epoch_id,
			party_id,
			score,
			staking_balance,
			open_volume,
			total_fees_paid,
			is_eligible,
			rank,
			vega_time
		)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6, $7, $8, $9, $10, $11);`,
		r.GameID, r.TeamID, r.EpochID, r.PartyID, r.Score, r.StakingBalance, r.OpenVolume, r.TotalFeesPaid, r.IsEligible,
		r.Rank, r.VegaTime)
	return err
}

func (gs *GameScores) AddTeamScore(ctx context.Context, r entities.GameTeamScore) error {
	defer metrics.StartSQLQuery("GameScores", "AddPartyScores")()
	_, err := gs.Connection.Exec(ctx,
		`INSERT INTO game_team_scores(
			game_id,
			team_id,
			epoch_id,
			score,
			vega_time
		)
		 VALUES ($1,  $2,  $3,  $4,  $5);`,
		r.GameID, r.TeamID, r.EpochID, r.Score, r.VegaTime)
	return err
}

// scany does not like deserializing byte arrays to strings so if an ID
// needs to be nillable, we need to scan it into a temporary struct that will
// define the ID field as a byte array and then parse the value accordingly.
type scannedPartyGameScore struct {
	GameID         entities.GameID
	TeamID         []byte
	EpochID        int64
	PartyID        entities.PartyID
	Score          decimal.Decimal
	StakingBalance decimal.Decimal
	OpenVolume     decimal.Decimal
	TotalFeesPaid  decimal.Decimal
	IsEligible     bool
	Rank           *uint64
	VegaTime       time.Time
	TxHash         entities.TxHash
	SeqNum         uint64
}

func (gs *GameScores) ListPartyScores(
	ctx context.Context,
	gameIDs []entities.GameID,
	partyIDs []entities.PartyID,
	teamIDs []entities.TeamID,
	pagination entities.CursorPagination,
) ([]entities.GamePartyScore, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	where, args, err := filterPartyQuery(gameIDs, partyIDs, teamIDs)
	if err != nil {
		return nil, pageInfo, err
	}

	query := `SELECT * FROM game_party_scores_current `
	query = fmt.Sprintf("%s %s", query, where)
	query, args, err = PaginateQuery[entities.PartyGameScoreCursor](query, args, gamesPartyOrderding, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	sPgs := []scannedPartyGameScore{}
	defer metrics.StartSQLQuery("GameScores", "ListPartyScores")()

	if err = pgxscan.Select(ctx, gs.Connection, &sPgs, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("querying game party scores: %w", err)
	}

	pgs := parseScannedPartyGameScores(sPgs)
	ret, pageInfo := entities.PageEntities[*v2.GamePartyScoresEdge](pgs, pagination)
	return ret, pageInfo, nil
}

func filterPartyQuery(gameIDs []entities.GameID, partyIDs []entities.PartyID, teamIDs []entities.TeamID) (string, []any, error) {
	var (
		args       []any
		conditions []string
	)

	if len(gameIDs) > 0 {
		gids := make([][]byte, len(gameIDs))
		for i, gid := range gameIDs {
			bytes, err := gid.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode game ID: %w", err)
			}
			gids[i] = bytes
		}
		conditions = append(conditions, fmt.Sprintf("game_id = ANY(%s)", nextBindVar(&args, gids)))
	}

	if len(partyIDs) > 0 {
		pids := make([][]byte, len(partyIDs))
		for i, pid := range partyIDs {
			bytes, err := pid.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode party ID: %w", err)
			}
			pids[i] = bytes
		}
		conditions = append(conditions, fmt.Sprintf("party_id = ANY(%s)", nextBindVar(&args, pids)))
	}

	if len(teamIDs) > 0 {
		tids := make([][]byte, len(teamIDs))
		for i, tid := range teamIDs {
			bytes, err := tid.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode team ID: %w", err)
			}
			tids[i] = bytes
		}
		conditions = append(conditions, fmt.Sprintf("team_id = ANY(%s)", nextBindVar(&args, tids)))
	}

	whereClause := strings.Join(conditions, " AND ")
	if len(whereClause) > 0 {
		return " WHERE " + whereClause, args, nil
	}
	return "", args, nil
}

func filterTeamQuery(gameIDs []entities.GameID, teamIDs []entities.TeamID) (string, []any, error) {
	var (
		args       []any
		conditions []string
	)

	if len(gameIDs) > 0 {
		gids := make([][]byte, len(gameIDs))
		for i, gid := range gameIDs {
			bytes, err := gid.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode game ID: %w", err)
			}
			gids[i] = bytes
		}
		conditions = append(conditions, fmt.Sprintf("game_id = ANY(%s)", nextBindVar(&args, gids)))
	}
	if len(teamIDs) > 0 {
		tids := make([][]byte, len(teamIDs))
		for i, tid := range teamIDs {
			bytes, err := tid.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode team ID: %w", err)
			}
			tids[i] = bytes
		}
		conditions = append(conditions, fmt.Sprintf("team_id = ANY(%s)", nextBindVar(&args, tids)))
	}
	if len(conditions) > 0 {
		return " WHERE " + strings.Join(conditions, " AND "), args, nil
	}
	return "", args, nil
}

func (gs *GameScores) ListTeamScores(
	ctx context.Context,
	gameIDs []entities.GameID,
	teamIDs []entities.TeamID,
	pagination entities.CursorPagination,
) ([]entities.GameTeamScore, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	where, args, err := filterTeamQuery(gameIDs, teamIDs)
	if err != nil {
		return nil, pageInfo, err
	}

	query := `SELECT * FROM game_team_scores_current `
	query = fmt.Sprintf("%s %s", query, where)
	query, args, err = PaginateQuery[entities.TeamGameScoreCursor](query, args, gamesTeamOrderding, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	tgs := []entities.GameTeamScore{}
	defer metrics.StartSQLQuery("GameScores", "ListTeamScores")()

	if err = pgxscan.Select(ctx, gs.Connection, &tgs, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("querying game team scores: %w", err)
	}

	ret, pageInfo := entities.PageEntities[*v2.GameTeamScoresEdge](tgs, pagination)
	return ret, pageInfo, nil
}

func parseScannedPartyGameScores(scanned []scannedPartyGameScore) []entities.GamePartyScore {
	pgs := make([]entities.GamePartyScore, 0, len(scanned))
	for _, s := range scanned {
		var teamID *entities.TeamID
		if s.TeamID != nil {
			id := hex.EncodeToString(s.TeamID)
			if id != "" {
				teamID = ptr.From(entities.TeamID(id))
			}
		}

		pgs = append(pgs, entities.GamePartyScore{
			GameID:         s.GameID,
			TeamID:         teamID,
			EpochID:        s.EpochID,
			PartyID:        s.PartyID,
			Score:          s.Score,
			StakingBalance: s.StakingBalance,
			OpenVolume:     s.OpenVolume,
			TotalFeesPaid:  s.TotalFeesPaid,
			IsEligible:     s.IsEligible,
			Rank:           s.Rank,
			VegaTime:       s.VegaTime,
		})
	}
	return pgs
}
