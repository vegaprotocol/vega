package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
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
			vega_time)
		 VALUES ($1,  $2,  $3,  $4,  $5,  $6, $7, $8);`,
		r.PartyID, r.AssetID, r.MarketID, r.RewardType, r.EpochID, r.Amount, r.PercentOfTotal, r.VegaTime)
	return err
}

func (rs *Rewards) GetAll(ctx context.Context) ([]entities.Reward, error) {
	defer metrics.StartSQLQuery("Rewards", "GetAll")()
	rewards := []entities.Reward{}
	err := pgxscan.Select(ctx, rs.Connection, &rewards, `
		SELECT * from rewards;`)
	return rewards, err
}

func (rs *Rewards) Get(ctx context.Context,
	partyIDHex *string,
	assetIDHex *string,
	p *entities.OffsetPagination,
) ([]entities.Reward, error) {
	query := `SELECT * from rewards`
	args := []interface{}{}
	if err := addRewardWhereClause(&query, &args, partyIDHex, assetIDHex); err != nil {
		return nil, err
	}

	if p != nil {
		order_cols := []string{"epoch_id", "party_id", "asset_id"}
		query, args = orderAndPaginateQuery(query, order_cols, *p, args...)
	}

	rewards := []entities.Reward{}
	defer metrics.StartSQLQuery("Rewards", "Get")()
	err := pgxscan.Select(ctx, rs.Connection, &rewards, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying rewards: %w", err)
	}
	return rewards, nil
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
	if partyIDHex != nil {
		partyID := entities.NewPartyID(*partyIDHex)
		query = fmt.Sprintf("%s WHERE party_id=%s", query, nextBindVar(args, partyID))
	}

	if assetIDHex != nil {
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
