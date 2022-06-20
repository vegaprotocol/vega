package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type Rewards struct {
	*ConnectionSource
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
			vega_time)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6, $7, $8, $9);`,
		r.PartyID, r.AssetID, r.MarketID, r.RewardType, r.EpochID, r.Amount, r.PercentOfTotal, r.Timestamp, r.VegaTime)
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
	pagination entities.CursorPagination,
) ([]entities.Reward, entities.PageInfo, error) {
	query, args, err := selectRewards(partyIDHex, assetIDHex)
	if err != nil {
		return nil, entities.PageInfo{}, err
	}

	sorting, cmp, cursor := extractPaginationInfo(pagination)
	rc := &entities.RewardCursor{}
	if cursor != "" {
		err := rc.Parse(cursor)
		if err != nil {
			return nil, entities.PageInfo{}, fmt.Errorf("parsing cursor: %w", err)
		}
	}
	cursorParams := []CursorQueryParameter{
		NewCursorQueryParameter("party_id", sorting, cmp, entities.NewPartyID(rc.PartyID)),
		NewCursorQueryParameter("asset_id", sorting, cmp, entities.NewAssetID(rc.AssetID)),
		NewCursorQueryParameter("epoch_id", sorting, cmp, rc.EpochID),
	}

	query, args = orderAndPaginateWithCursor(query, pagination, cursorParams, args...)

	rewards := []entities.Reward{}
	if err := pgxscan.Select(ctx, rs.Connection, &rewards, query, args...); err != nil {
		return nil, entities.PageInfo{}, fmt.Errorf("querying rewards: %w", err)
	}

	pagedData, pageInfo := entities.PageEntities[*v2.RewardEdge](rewards, pagination)
	return pagedData, pageInfo, nil
}

func (rs *Rewards) GetByOffset(ctx context.Context,
	partyIDHex *string,
	assetIDHex *string,
	pagination *entities.OffsetPagination,
) ([]entities.Reward, error) {
	query, args, err := selectRewards(partyIDHex, assetIDHex)
	if err != nil {
		return nil, err
	}

	if pagination != nil {
		order_cols := []string{"epoch_id", "party_id", "asset_id"}
		query, args = orderAndPaginateQuery(query, order_cols, *pagination, args...)
	}

	rewards := []entities.Reward{}
	defer metrics.StartSQLQuery("Rewards", "Get")()
	err = pgxscan.Select(ctx, rs.Connection, &rewards, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying rewards: %w", err)
	}
	return rewards, nil
}

func selectRewards(partyIDHex, assetIDHex *string) (string, []interface{}, error) {
	query := `SELECT * from rewards`
	args := []interface{}{}
	if err := addRewardWhereClause(&query, &args, partyIDHex, assetIDHex); err != nil {
		return "", nil, err
	}

	return query, args, nil
}

func (rs *Rewards) GetSummaries(ctx context.Context,
	partyIDHex *string, assetIDHex *string,
) ([]entities.RewardSummary, error) {
	query := `SELECT party_id, asset_id, sum(amount) as amount FROM rewards`
	args := []interface{}{}
	if err := addRewardWhereClause(&query, &args, partyIDHex, assetIDHex); err != nil {
		return nil, err
	}

	query = fmt.Sprintf("%s GROUP BY party_id, asset_id", query)

	summaries := []entities.RewardSummary{}
	defer metrics.StartSQLQuery("Rewards", "GetSummaries")()
	err := pgxscan.Select(ctx, rs.Connection, &summaries, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying rewards: %w", err)
	}
	return summaries, nil
}

// -------------------------------------------- Utility Methods

func addRewardWhereClause(queryPtr *string, args *[]interface{}, partyIDHex, assetIDHex *string) error {
	query := *queryPtr
	if partyIDHex != nil && *partyIDHex != "" {
		partyID := entities.NewPartyID(*partyIDHex)
		query = fmt.Sprintf("%s WHERE party_id=%s", query, nextBindVar(args, partyID))
	}

	if assetIDHex != nil && *assetIDHex != "" {
		clause := "WHERE"
		if partyIDHex != nil {
			clause = "AND"
		}

		assetID := entities.ID(*assetIDHex)
		query = fmt.Sprintf("%s %s asset_id=%s", query, clause, nextBindVar(args, assetID))
	}
	*queryPtr = query
	return nil
}
