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
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/shopspring/decimal"
)

type Rewards struct {
	*ConnectionSource
	runningTotals        map[entities.GameID]map[entities.PartyID]decimal.Decimal
	runningTotalsQuantum map[entities.GameID]map[entities.PartyID]decimal.Decimal
}

var rewardsOrdering = TableOrdering{
	ColumnOrdering{Name: "epoch_id", Sorting: ASC},
}

func NewRewards(ctx context.Context, connectionSource *ConnectionSource) *Rewards {
	r := &Rewards{
		ConnectionSource: connectionSource,
	}
	r.runningTotals = make(map[entities.GameID]map[entities.PartyID]decimal.Decimal)
	r.runningTotalsQuantum = make(map[entities.GameID]map[entities.PartyID]decimal.Decimal)
	r.fetchRunningTotals(ctx)
	return r
}

func (rs *Rewards) fetchRunningTotals(ctx context.Context) {
	query := `SELECT * FROM current_game_reward_totals`
	var totals []entities.RewardTotals
	err := pgxscan.Select(ctx, rs.Connection, &totals, query)
	if err != nil && !pgxscan.NotFound(err) {
		panic(fmt.Errorf("could not retrieve game reward totals: %w", err))
	}
	for _, total := range totals {
		if _, ok := rs.runningTotals[total.GameID]; !ok {
			rs.runningTotals[total.GameID] = make(map[entities.PartyID]decimal.Decimal)
		}
		if _, ok := rs.runningTotalsQuantum[total.GameID]; !ok {
			rs.runningTotalsQuantum[total.GameID] = make(map[entities.PartyID]decimal.Decimal)
		}
		rs.runningTotals[total.GameID][total.PartyID] = total.TotalRewards
		rs.runningTotalsQuantum[total.GameID][total.PartyID] = total.TotalRewardsQuantum
	}
}

func (rs *Rewards) Add(ctx context.Context, r entities.Reward) error {
	defer metrics.StartSQLQuery("Rewards", "Add")()
	_, err := rs.Connection.Exec(ctx,
		`INSERT INTO rewards(
			party_id,
			asset_id,
			market_id,
			reward_type,
			epoch_id,
			amount,
			quantum_amount,
			percent_of_total,
			timestamp,
			tx_hash,
			vega_time,
			seq_num,
			locked_until_epoch_id,
            game_id
		)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6, $7, $8, $9, $10, $11, $12, $13, $14);`,
		r.PartyID, r.AssetID, r.MarketID, r.RewardType, r.EpochID, r.Amount, r.QuantumAmount, r.PercentOfTotal, r.Timestamp, r.TxHash,
		r.VegaTime, r.SeqNum, r.LockedUntilEpochID, r.GameID)

	if r.GameID != nil && *r.GameID != "" {
		gID := *r.GameID
		if _, ok := rs.runningTotals[gID]; !ok {
			rs.runningTotals[gID] = make(map[entities.PartyID]decimal.Decimal)
			rs.runningTotals[gID][r.PartyID] = num.DecimalZero()
		}
		if _, ok := rs.runningTotalsQuantum[gID]; !ok {
			rs.runningTotalsQuantum[gID] = make(map[entities.PartyID]decimal.Decimal)
			rs.runningTotalsQuantum[gID][r.PartyID] = num.DecimalZero()
		}

		rs.runningTotals[gID][r.PartyID] = rs.runningTotals[gID][r.PartyID].Add(r.Amount)
		rs.runningTotalsQuantum[gID][r.PartyID] = rs.runningTotalsQuantum[gID][r.PartyID].Add(r.QuantumAmount)

		defer metrics.StartSQLQuery("GameRewardTotals", "Add")()
		_, err = rs.Connection.Exec(ctx, `INSERT INTO game_reward_totals(
			game_id,
			party_id,
			asset_id,
			market_id,
			epoch_id,
            team_id,
			total_rewards,
			total_rewards_quantum
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);`,
			r.GameID,
			r.PartyID,
			r.AssetID,
			r.MarketID,
			r.EpochID,
			entities.TeamID(""),
			rs.runningTotals[gID][r.PartyID],
			rs.runningTotalsQuantum[gID][r.PartyID])
	}
	return err
}

// scany does not like deserializing byte arrays to strings so if an ID
// needs to be nillable, we need to scan it into a temporary struct that will
// define the ID field as a byte array and then parse the value accordingly.
type scannedRewards struct {
	PartyID            entities.PartyID
	AssetID            entities.AssetID
	MarketID           entities.MarketID
	EpochID            int64
	Amount             decimal.Decimal
	QuantumAmount      decimal.Decimal
	PercentOfTotal     float64
	RewardType         string
	Timestamp          time.Time
	TxHash             entities.TxHash
	VegaTime           time.Time
	SeqNum             uint64
	LockedUntilEpochID int64
	GameID             []byte
	TeamID             []byte
}

func (rs *Rewards) GetAll(ctx context.Context) ([]entities.Reward, error) {
	defer metrics.StartSQLQuery("Rewards", "GetAll")()
	scanned := []scannedRewards{}
	err := pgxscan.Select(ctx, rs.Connection, &scanned, `SELECT * FROM rewards;`)
	if err != nil {
		return nil, err
	}
	return parseScannedRewards(scanned), nil
}

func (rs *Rewards) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Reward, error) {
	defer metrics.StartSQLQuery("Rewards", "GetByTxHash")()

	scanned := []scannedRewards{}
	err := pgxscan.Select(ctx, rs.Connection, &scanned, `SELECT * FROM rewards WHERE tx_hash = $1`, txHash)
	if err != nil {
		return nil, err
	}

	return parseScannedRewards(scanned), nil
}

func (rs *Rewards) GetByCursor(ctx context.Context,
	partyIDs []string,
	assetIDHex *string,
	fromEpoch *uint64,
	toEpoch *uint64,
	pagination entities.CursorPagination,
	teamIDHex, gameIDHex *string,
) ([]entities.Reward, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	query := `
	WITH cte_rewards AS (
		SELECT r.*, grt.team_id
		FROM rewards r
		LEFT JOIN game_reward_totals grt ON r.game_id = grt.game_id AND r.party_id = grt.party_id and r.epoch_id = grt.epoch_id
	)
	SELECT * from cte_rewards`
	args := []interface{}{}
	query, args = addRewardWhereClause(query, args, partyIDs, assetIDHex, teamIDHex, gameIDHex, fromEpoch, toEpoch)

	query, args, err := PaginateQuery[entities.RewardCursor](query, args, rewardsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	scanned := []scannedRewards{}
	if err := pgxscan.Select(ctx, rs.Connection, &scanned, query, args...); err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("querying rewards: %w", err)
	}

	rewards := parseScannedRewards(scanned)
	rewards, pageInfo = entities.PageEntities[*v2.RewardEdge](rewards, pagination)
	return rewards, pageInfo, nil
}

func (rs *Rewards) GetSummaries(ctx context.Context,
	partyIDs []string, assetIDHex *string,
) ([]entities.RewardSummary, error) {
	query := `SELECT party_id, asset_id, SUM(amount) AS amount FROM rewards`
	args := []interface{}{}
	query, args = addRewardWhereClause(query, args, partyIDs, assetIDHex, nil, nil, nil, nil)
	query = fmt.Sprintf("%s GROUP BY party_id, asset_id", query)
	fmt.Println("query", query)

	summaries := []entities.RewardSummary{}
	defer metrics.StartSQLQuery("Rewards", "GetSummaries")()
	err := pgxscan.Select(ctx, rs.Connection, &summaries, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying rewards: %w", err)
	}
	return summaries, nil
}

// GetEpochSummaries returns paged epoch reward summary aggregated by asset, market, and reward type for a given range of epochs.
func (rs *Rewards) GetEpochSummaries(ctx context.Context,
	filter entities.RewardSummaryFilter,
	pagination entities.CursorPagination,
) ([]entities.EpochRewardSummary, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	query := `SELECT epoch_id, asset_id, market_id, reward_type, SUM(amount) AS amount FROM rewards `
	where, args, err := FilterRewardsQuery(filter)
	if err != nil {
		return nil, pageInfo, err
	}

	query = fmt.Sprintf("%s %s GROUP BY epoch_id, asset_id, market_id, reward_type", query, where)
	query = fmt.Sprintf("WITH subquery AS (%s) SELECT * FROM subquery", query)
	query, args, err = PaginateQuery[entities.EpochRewardSummaryCursor](query, args, rewardsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	var summaries []entities.EpochRewardSummary
	defer metrics.StartSQLQuery("Rewards", "GetEpochSummaries")()

	if err = pgxscan.Select(ctx, rs.Connection, &summaries, query, args...); err != nil {
		return nil, pageInfo, fmt.Errorf("querying epoch reward summaries: %w", err)
	}

	summaries, pageInfo = entities.PageEntities[*v2.EpochRewardSummaryEdge](summaries, pagination)
	return summaries, pageInfo, nil
}

// -------------------------------------------- Utility Methods

func addRewardWhereClause(query string, args []interface{}, partyIDs []string, assetIDHex, teamIDHex, gameIDHex *string, fromEpoch, toEpoch *uint64) (string, []interface{}) {
	predicates := []string{}

	if len(partyIDs) > 0 {
		inArgs, inList := prepareInClauseList[entities.PartyID](partyIDs)
		args = append(args, inArgs...)
		predicates = append(predicates, fmt.Sprintf("party_id IN (%s)", inList))
	}

	if assetIDHex != nil && *assetIDHex != "" {
		assetID := entities.AssetID(*assetIDHex)
		predicates = append(predicates, fmt.Sprintf("asset_id = %s", nextBindVar(&args, assetID)))
	}

	if teamIDHex != nil && *teamIDHex != "" {
		teamID := entities.TeamID(*teamIDHex)
		predicates = append(predicates, fmt.Sprintf("team_id = %s", nextBindVar(&args, teamID)))
	}

	if gameIDHex != nil && *gameIDHex != "" {
		gameID := entities.GameID(*gameIDHex)
		predicates = append(predicates, fmt.Sprintf("game_id = %s", nextBindVar(&args, gameID)))
	}

	if fromEpoch != nil {
		predicates = append(predicates, fmt.Sprintf("epoch_id >= %s", nextBindVar(&args, *fromEpoch)))
	}

	if toEpoch != nil {
		predicates = append(predicates, fmt.Sprintf("epoch_id <= %s", nextBindVar(&args, *toEpoch)))
	}

	if len(predicates) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(predicates, " AND "))
	}

	return query, args
}

func prepareInClauseList[A any, T entities.ID[A]](ids []string) ([]interface{}, string) {
	var args []interface{}
	var list strings.Builder
	for i, id := range ids {
		if i > 0 {
			list.WriteString(",")
		}

		list.WriteString(nextBindVar(&args, T(id)))
	}
	return args, list.String()
}

// FilterRewardsQuery returns a WHERE part of the query and args for filtering the rewards table.
func FilterRewardsQuery(filter entities.RewardSummaryFilter) (string, []any, error) {
	var (
		args       []any
		conditions []string
	)

	if len(filter.AssetIDs) > 0 {
		assetIDs := make([][]byte, len(filter.AssetIDs))
		for i, assetID := range filter.AssetIDs {
			bytes, err := assetID.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode asset ID: %w", err)
			}
			assetIDs[i] = bytes
		}
		conditions = append(conditions, fmt.Sprintf("asset_id = ANY(%s)", nextBindVar(&args, assetIDs)))
	}

	if len(filter.MarketIDs) > 0 {
		marketIDs := make([][]byte, len(filter.MarketIDs))
		for i, marketID := range filter.MarketIDs {
			bytes, err := marketID.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode market ID: %w", err)
			}
			marketIDs[i] = bytes
		}
		conditions = append(conditions, fmt.Sprintf("market_id = ANY(%s)", nextBindVar(&args, marketIDs)))
	}

	if filter.FromEpoch != nil {
		conditions = append(conditions, fmt.Sprintf("epoch_id >= %s", nextBindVar(&args, filter.FromEpoch)))
	}

	if filter.ToEpoch != nil {
		conditions = append(conditions, fmt.Sprintf("epoch_id <= %s", nextBindVar(&args, filter.ToEpoch)))
	}

	if len(conditions) == 0 {
		return "", nil, nil
	}
	return " WHERE " + strings.Join(conditions, " AND "), args, nil
}

func parseScannedRewards(scanned []scannedRewards) []entities.Reward {
	rewards := make([]entities.Reward, len(scanned))
	for i, s := range scanned {
		var gID *entities.GameID
		var teamID *entities.TeamID
		if s.GameID != nil {
			id := hex.EncodeToString(s.GameID)
			if id != "" {
				gID = ptr.From(entities.GameID(id))
			}
		}
		if s.TeamID != nil {
			id := hex.EncodeToString(s.TeamID)
			if id != "" {
				teamID = ptr.From(entities.TeamID(id))
			}
		}
		rewards[i] = entities.Reward{
			PartyID:            s.PartyID,
			AssetID:            s.AssetID,
			MarketID:           s.MarketID,
			EpochID:            s.EpochID,
			Amount:             s.Amount,
			QuantumAmount:      s.QuantumAmount,
			PercentOfTotal:     s.PercentOfTotal,
			RewardType:         s.RewardType,
			Timestamp:          s.Timestamp,
			TxHash:             s.TxHash,
			VegaTime:           s.VegaTime,
			SeqNum:             s.SeqNum,
			LockedUntilEpochID: s.LockedUntilEpochID,
			GameID:             gID,
			TeamID:             teamID,
		}
	}
	return rewards
}
