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
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
)

var listPartyMarginModesOrdering = TableOrdering{
	ColumnOrdering{Name: "market_id", Sorting: ASC},
	ColumnOrdering{Name: "party_id", Sorting: ASC},
}

type ListPartyMarginModesFilters struct {
	MarketID *entities.MarketID
	PartyID  *entities.PartyID
}

type MarginModes struct {
	*ConnectionSource
}

func (t *MarginModes) UpdatePartyMarginMode(ctx context.Context, update entities.PartyMarginMode) error {
	defer metrics.StartSQLQuery("MarginModes", "UpdatePartyMarginMode")()
	if _, err := t.Connection.Exec(
		ctx,
		`INSERT INTO party_margin_modes(market_id, party_id, margin_mode, margin_factor, min_theoretical_margin_factor, max_theoretical_leverage, at_epoch)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (market_id, party_id) DO UPDATE SET
			margin_mode = excluded.margin_mode,
			margin_factor = excluded.margin_factor,
			min_theoretical_margin_factor = excluded.min_theoretical_margin_factor,
			max_theoretical_leverage = excluded.max_theoretical_leverage,
			at_epoch = excluded.at_epoch`,
		update.MarketID,
		update.PartyID,
		update.MarginMode,
		update.MarginFactor,
		update.MinTheoreticalMarginFactor,
		update.MaxTheoreticalLeverage,
		update.AtEpoch,
	); err != nil {
		return err
	}

	return nil
}

func (t *MarginModes) ListPartyMarginModes(ctx context.Context, pagination entities.CursorPagination, filters ListPartyMarginModesFilters) ([]entities.PartyMarginMode, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("MarginModes", "ListPartyMarginModes")()

	var (
		modes    []entities.PartyMarginMode
		args     []interface{}
		pageInfo entities.PageInfo
	)

	query := `SELECT * FROM party_margin_modes`

	whereClauses := []string{}
	if filters.MarketID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("market_id = %s", nextBindVar(&args, *filters.MarketID)))
	}
	if filters.PartyID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("party_id = %s", nextBindVar(&args, *filters.PartyID)))
	}

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query, args, err := PaginateQuery[entities.PartyMarginModeCursor](query, args, listPartyMarginModesOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	if err := pgxscan.Select(ctx, t.Connection, &modes, query, args...); err != nil {
		return nil, pageInfo, err
	}

	modes, pageInfo = entities.PageEntities[*v2.PartyMarginModeEdge](modes, pagination)

	return modes, pageInfo, nil
}

func NewMarginModes(connectionSource *ConnectionSource) *MarginModes {
	return &MarginModes{
		ConnectionSource: connectionSource,
	}
}
