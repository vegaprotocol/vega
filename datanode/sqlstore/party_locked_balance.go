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
	"fmt"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"

	"github.com/georgysavva/scany/pgxscan"
)

type PartyLockedBalance struct {
	*ConnectionSource
}

func NewPartyLockedBalances(connectionSource *ConnectionSource) *PartyLockedBalance {
	return &PartyLockedBalance{
		ConnectionSource: connectionSource,
	}
}

func (plb *PartyLockedBalance) Prune(
	ctx context.Context,
	currentEpoch uint64,
) error {
	defer metrics.StartSQLQuery("PartyLockedBalance", "Prune")()
	_, err := plb.Connection.Exec(
		ctx,
		"DELETE FROM party_locked_balances_current WHERE until_epoch <= $1",
		currentEpoch,
	)

	return err
}

func (plb *PartyLockedBalance) Add(ctx context.Context, balance entities.PartyLockedBalance) error {
	defer metrics.StartSQLQuery("PartyLockedBalance", "Add")()
	_, err := plb.Connection.Exec(ctx,
		`INSERT INTO party_locked_balances(party_id, asset_id, at_epoch, until_epoch, balance, vega_time)
         VALUES ($1, $2, $3, $4, $5, $6)
         ON CONFLICT (vega_time, party_id, asset_id, until_epoch) DO NOTHING`,
		balance.PartyID,
		balance.AssetID,
		balance.AtEpoch,
		balance.UntilEpoch,
		balance.Balance,
		balance.VegaTime,
	)
	return err
}

func (plb *PartyLockedBalance) Get(ctx context.Context, partyID *entities.PartyID, assetID *entities.AssetID) (
	[]entities.PartyLockedBalance, error,
) {
	defer metrics.StartSQLQuery("PartyLockedBalance", "Get")()
	var args []interface{}

	query := `SELECT * FROM party_locked_balances_current`
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

	var balances []entities.PartyLockedBalance
	if err := pgxscan.Select(ctx, plb.Connection, &balances, query, args...); err != nil {
		return balances, err
	}

	return balances, nil
}
