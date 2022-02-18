package sqlstore

import (
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
)

// Return an SQL query string and corresponding bind arguments to return rows
// from the account table filtered according to this AccountFilter.
func filterAccountsQuery(af entities.AccountFilter) (string, []interface{}) {
	var args []interface{}

	query := `SELECT id, party_id, asset_id, market_id, type, vega_time
	          FROM ACCOUNTS `
	if af.Asset.ID != nil {
		query = fmt.Sprintf("%s WHERE asset_id=%s", query, nextBindVar(&args, af.Asset.ID))
	} else {
		query = fmt.Sprintf("%s WHERE true", query)
	}

	if len(af.Parties) > 0 {
		partyIDs := make([][]byte, len(af.Parties))
		for i, party := range af.Parties {
			partyIDs[i] = party.ID
		}
		query += " AND party_id=ANY(" + nextBindVar(&args, partyIDs) + ")"
	}

	if len(af.AccountTypes) > 0 {
		query += " AND type=ANY(" + nextBindVar(&args, af.AccountTypes) + ")"
	}

	if len(af.Markets) > 0 {
		marketIds := make([][]byte, len(af.Markets))
		for i, market := range af.Markets {
			marketIds[i] = market.ID
		}

		query += " AND market_id=ANY(" + nextBindVar(&args, marketIds) + ")"
	}

	return query, args
}
