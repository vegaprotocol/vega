package sqlstore

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
)

type PartyVestingBalance struct {
	*ConnectionSource
}

func NewPartyVestingBalances(connectionSource *ConnectionSource) *PartyVestingBalance {
	return &PartyVestingBalance{
		ConnectionSource: connectionSource,
	}
}

func (plb *PartyVestingBalance) Add(ctx context.Context, balance entities.PartyVestingBalance) error {
	defer metrics.StartSQLQuery("PartyVestingBalance", "Add")()
	_, err := plb.Connection.Exec(ctx,
		`INSERT INTO party_vesting_balances(party_id, asset_id, at_epoch, balance, vega_time)
         VALUES ($1, $2, $3, $4, $5)
         ON CONFLICT (vega_time, party_id, asset_id) DO NOTHING`,
		balance.PartyID,
		balance.AssetID,
		balance.AtEpoch,
		balance.Balance,
		balance.VegaTime,
	)
	return err
}

func (plb *PartyVestingBalance) Get(ctx context.Context, partyID *entities.PartyID, assetID *entities.AssetID) (
	[]entities.PartyVestingBalance, error,
) {
	defer metrics.StartSQLQuery("PartyVestingBalance", "Get")()
	var args []interface{}

	query := `SELECT * FROM party_vesting_balances_current`
	where := []string{}

	if partyID != nil {
		where = append(where, fmt.Sprintf("party_id = %s", nextBindVar(&args, *partyID)))
	}

	if assetID != nil {
		where = append(where, fmt.Sprintf("asset_id = %s", nextBindVar(&args, *assetID)))
	}

	whereClause := ""

	if len(where) > 0 {
		whereClause = "WHERE"
		for i, w := range where {
			if i > 0 {
				whereClause = fmt.Sprintf("%s AND", whereClause)
			}
			whereClause = fmt.Sprintf("%s %s", whereClause, w)
		}
	}

	query = fmt.Sprintf("%s %s", query, whereClause)

	var balances []entities.PartyVestingBalance
	if err := pgxscan.Select(ctx, plb.Connection, &balances, query, args...); err != nil {
		return balances, err
	}

	return balances, nil
}
