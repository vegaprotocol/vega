// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type Rewards struct {
	*ConnectionSource
}

var rewardsOrdering = TableOrdering{
	ColumnOrdering{Name: "epoch_id", Sorting: ASC},
}

func NewRewards(connectionSource *ConnectionSource) *Rewards {
	r := &Rewards{
		ConnectionSource: connectionSource,
	}
	return r
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
			percent_of_total,
			timestamp,
			tx_hash,
			vega_time,
			seq_num)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6, $7, $8, $9, $10, $11);`,
		r.PartyID, r.AssetID, r.MarketID, r.RewardType, r.EpochID, r.Amount, r.PercentOfTotal, r.Timestamp, r.TxHash,
		r.VegaTime, r.SeqNum)
	return err
}

func (rs *Rewards) GetAll(ctx context.Context) ([]entities.Reward, error) {
	defer metrics.StartSQLQuery("Rewards", "GetAll")()
	rewards := []entities.Reward{}
	err := pgxscan.Select(ctx, rs.Connection, &rewards, `
		SELECT * from rewards;`)
	return rewards, err
}

func (rs *Rewards) GetByCursor(ctx context.Context,
	partyIDHex *string,
	assetIDHex *string,
	fromEpoch *uint64,
	toEpoch *uint64,
	pagination entities.CursorPagination,
) ([]entities.Reward, entities.PageInfo, error) {
	var pageInfo entities.PageInfo
	query := `SELECT * from rewards`
	args := []interface{}{}
	query, args = addRewardWhereClause(query, args, partyIDHex, assetIDHex, fromEpoch, toEpoch)

	query, args, err := PaginateQuery[entities.RewardCursor](query, args, rewardsOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	rewards := []entities.Reward{}
	if err := pgxscan.Select(ctx, rs.Connection, &rewards, query, args...); err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("querying rewards: %w", err)
	}

	rewards, pageInfo = entities.PageEntities[*v2.RewardEdge](rewards, pagination)
	return rewards, pageInfo, nil
}

func (rs *Rewards) GetSummaries(ctx context.Context,
	partyIDHex *string, assetIDHex *string,
) ([]entities.RewardSummary, error) {
	query := `SELECT party_id, asset_id, sum(amount) as amount FROM rewards`
	args := []interface{}{}
	query, args = addRewardWhereClause(query, args, partyIDHex, assetIDHex, nil, nil)
	query = fmt.Sprintf("%s GROUP BY party_id, asset_id", query)

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
	query := `SELECT epoch_id, asset_id, market_id, reward_type, sum(amount) as amount FROM rewards `
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

func addRewardWhereClause(query string, args []interface{}, partyIDHex, assetIDHex *string, fromEpoch, toEpoch *uint64) (string, []interface{}) {
	predicates := []string{}

	if partyIDHex != nil && *partyIDHex != "" {
		partyID := entities.PartyID(*partyIDHex)
		predicates = append(predicates, fmt.Sprintf("party_id = %s", nextBindVar(&args, partyID)))
	}

	if assetIDHex != nil && *assetIDHex != "" {
		assetID := entities.AssetID(*assetIDHex)
		predicates = append(predicates, fmt.Sprintf("asset_id = %s", nextBindVar(&args, assetID)))
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
