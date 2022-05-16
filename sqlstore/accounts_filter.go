package sqlstore

import (
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
)

// Return an SQL query string and corresponding bind arguments to return rows
// from the account table filtered according to this AccountFilter.
func filterAccountsQuery(af entities.AccountFilter) (string, []interface{}, error) {
	var args []interface{}
	var err error

	query := `SELECT id, party_id, asset_id, market_id, type, vega_time
	          FROM ACCOUNTS `
	if af.Asset.ID.String() != "" {
		query = fmt.Sprintf("%s WHERE asset_id=%s", query, nextBindVar(&args, af.Asset.ID))
	} else {
		query = fmt.Sprintf("%s WHERE true", query)
	}

	if len(af.Parties) > 0 {
		partyIDs := make([][]byte, len(af.Parties))
		for i, party := range af.Parties {
			partyIDs[i], err = party.ID.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("invalid party id: %w", err)
			}
		}
		query += " AND party_id=ANY(" + nextBindVar(&args, partyIDs) + ")"
	}

	if len(af.AccountTypes) > 0 {
		query += " AND type=ANY(" + nextBindVar(&args, af.AccountTypes) + ")"
	}

	if len(af.Markets) > 0 {
		marketIds := make([][]byte, len(af.Markets))
		for i, market := range af.Markets {
			marketIds[i], err = market.ID.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("invalid market id: %w", err)
			}
		}

		query += " AND market_id=ANY(" + nextBindVar(&args, marketIds) + ")"
	}

	return query, args, nil
}

func filterAccountBalancesQuery(af entities.AccountFilter, pagination entities.OffsetPagination) (string, []interface{}, error) {
	var args []interface{}

	where := ""
	and := ""

	if len(af.Asset.ID.String()) != 0 {
		where = fmt.Sprintf("ACCOUNTS.asset_id=%s", nextBindVar(&args, af.Asset.ID))
		and = " AND "
	}

	if len(af.Parties) > 0 {
		partyIDs := make([][]byte, len(af.Parties))
		for i, party := range af.Parties {
			bytes, err := party.ID.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("Couldn't decode party ID: %w", err)
			}
			partyIDs[i] = bytes
		}
		where = fmt.Sprintf(`%s%sACCOUNTS.party_id=ANY(%s)`, where, and, nextBindVar(&args, partyIDs))
		if and == "" {
			and = " AND "
		}
	}

	if len(af.AccountTypes) > 0 {
		where = fmt.Sprintf(`%s%stype=ANY(%s)`, where, and, nextBindVar(&args, af.AccountTypes))
		if and == "" {
			and = " AND "
		}
	}

	if len(af.Markets) > 0 {
		marketIDs := make([][]byte, len(af.Markets))
		for i, market := range af.Markets {
			bytes, err := market.ID.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("Couldn't decode market ID: %w", err)
			}
			marketIDs[i] = bytes
		}

		where = fmt.Sprintf(`%s%sACCOUNTS.market_id=ANY(%s)`, where, and, nextBindVar(&args, marketIDs))
	}

	query := `SELECT DISTINCT ON (ACCOUNTS.id) ACCOUNTS.id, ACCOUNTS.party_id, ACCOUNTS.asset_id, ACCOUNTS.market_id, ACCOUNTS.type, 
				BALANCES.balance, BALANCES.vega_time
	          FROM ACCOUNTS JOIN BALANCES ON ACCOUNTS.id = BALANCES.account_id `

	if where != "" {
		query = fmt.Sprintf("%s WHERE %s", query, where)
	}

	// We are adding a custom ordering to ensure we're getting the latest balances for each account from our query
	query = fmt.Sprintf("%s ORDER BY ACCOUNTS.id, BALANCES.vega_time DESC", query)

	// and we're calling the order and paginate query method so that we can paginate later as it is a requirement for
	// data-node API v2, but pass no ordering columns as we've already defined the ordering we want for this query.
	query, args = orderAndPaginateQuery(query, nil, pagination, args...)
	return query, args, nil
}
