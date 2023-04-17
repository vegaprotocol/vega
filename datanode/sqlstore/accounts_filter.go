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
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
)

// Return an SQL query string and corresponding bind arguments to return rows
// from the account table filtered according to this AccountFilter.
func filterAccountsQuery(af entities.AccountFilter, includeVegaTime bool) (string, []interface{}, error) {
	var args []interface{}
	var err error

	query := `SELECT id, party_id, asset_id, market_id, type, tx_hash FROM ACCOUNTS `
	if includeVegaTime {
		query = `SELECT id, party_id, asset_id, market_id, type, tx_hash, vega_time FROM ACCOUNTS `
	}

	if af.AssetID.String() != "" {
		query = fmt.Sprintf("%s WHERE asset_id=%s", query, nextBindVar(&args, af.AssetID))
	} else {
		query = fmt.Sprintf("%s WHERE true", query)
	}

	if len(af.PartyIDs) > 0 {
		partyIDs := make([][]byte, len(af.PartyIDs))
		for i, party := range af.PartyIDs {
			partyIDs[i], err = party.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("invalid party id: %w", err)
			}
		}
		query += " AND party_id=ANY(" + nextBindVar(&args, partyIDs) + ")"
	}

	if len(af.AccountTypes) > 0 {
		query += " AND type=ANY(" + nextBindVar(&args, af.AccountTypes) + ")"
	}

	if len(af.MarketIDs) > 0 {
		marketIds := make([][]byte, len(af.MarketIDs))
		for i, market := range af.MarketIDs {
			marketIds[i], err = market.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("invalid market id: %w", err)
			}
		}

		query += " AND market_id=ANY(" + nextBindVar(&args, marketIds) + ")"
	}

	return query, args, nil
}

func currentAccountBalancesQuery() string {
	return `SELECT ACCOUNTS.id, ACCOUNTS.party_id, ACCOUNTS.asset_id, ACCOUNTS.market_id, ACCOUNTS.type,
			current_balances.balance, current_balances.tx_hash, current_balances.vega_time
			FROM ACCOUNTS JOIN current_balances ON ACCOUNTS.id = current_balances.account_id `
}

func accountBalancesQuery() string {
	return `SELECT ACCOUNTS.id, ACCOUNTS.party_id, ACCOUNTS.asset_id, ACCOUNTS.market_id, ACCOUNTS.type,
			balances.balance, balances.tx_hash, balances.vega_time
			FROM ACCOUNTS JOIN balances ON ACCOUNTS.id = balances.account_id `
}

func filterAccountBalancesQuery(af entities.AccountFilter) (string, []interface{}, error) {
	var args []interface{}

	where := ""
	and := ""

	if len(af.AssetID.String()) != 0 {
		where = fmt.Sprintf("ACCOUNTS.asset_id=%s", nextBindVar(&args, af.AssetID))
		and = " AND "
	}

	if len(af.PartyIDs) > 0 {
		partyIDs := make([][]byte, len(af.PartyIDs))
		for i, party := range af.PartyIDs {
			bytes, err := party.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode party ID: %w", err)
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

	if len(af.MarketIDs) > 0 {
		marketIDs := make([][]byte, len(af.MarketIDs))
		for i, market := range af.MarketIDs {
			bytes, err := market.Bytes()
			if err != nil {
				return "", nil, fmt.Errorf("could not decode market ID: %w", err)
			}
			marketIDs[i] = bytes
		}

		where = fmt.Sprintf(`%s%sACCOUNTS.market_id=ANY(%s)`, where, and, nextBindVar(&args, marketIDs))
	}

	query := currentAccountBalancesQuery()

	if where != "" {
		query = fmt.Sprintf("%s WHERE %s", query, where)
	}

	return query, args, nil
}
