// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type AccountSource interface {
	Query(filter entities.AccountFilter) ([]entities.Account, error)
}

type MarginLevels struct {
	*ConnectionSource
	columns       []string
	marginLevels  []*entities.MarginLevels
	batcher       MapBatcher[entities.MarginLevelsKey, entities.MarginLevels]
	accountSource AccountSource
}

const (
	sqlMarginLevelColumns = `account_id,timestamp,maintenance_margin,search_level,initial_margin,collateral_release_level,vega_time`
)

func NewMarginLevels(connectionSource *ConnectionSource) *MarginLevels {
	return &MarginLevels{
		ConnectionSource: connectionSource,
		batcher: NewMapBatcher[entities.MarginLevelsKey, entities.MarginLevels](
			"margin_levels",
			entities.MarginLevelsColumns),
	}
}

func (ml *MarginLevels) Add(marginLevel entities.MarginLevels) error {
	ml.batcher.Add(marginLevel)
	return nil
}

func (ml *MarginLevels) Flush(ctx context.Context) ([]entities.MarginLevels, error) {
	defer metrics.StartSQLQuery("MarginLevels", "Flush")()
	return ml.batcher.Flush(ctx, ml.pool)
}

func (ml *MarginLevels) GetMarginLevelsByID(ctx context.Context, partyID, marketID string, pagination entities.OffsetPagination) ([]entities.MarginLevels, error) {
	whereClause, bindVars := buildAccountWhereClause(partyID, marketID)

	query := fmt.Sprintf(`select distinct on (account_id) %s
		from all_margin_levels
		%s
		order by account_id, vega_time desc`, sqlMarginLevelColumns,
		whereClause)

	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	defer metrics.StartSQLQuery("MarginLevels", "GetByID")()
	var marginLevels []entities.MarginLevels
	err := pgxscan.Select(ctx, ml.Connection, &marginLevels, query, bindVars...)
	return marginLevels, err
}

func buildAccountWhereClause(partyID, marketID string) (string, []interface{}) {
	party := entities.NewPartyID(partyID)
	market := entities.NewMarketID(marketID)

	var bindVars []interface{}

	whereParty := ""
	if partyID != "" {
		whereParty = fmt.Sprintf("party_id = %s", nextBindVar(&bindVars, party))
	}

	whereMarket := ""
	if marketID != "" {
		whereMarket = fmt.Sprintf("market_id = %s", nextBindVar(&bindVars, market))
	}

	accountsWhereClause := ""

	if whereParty != "" && whereMarket != "" {
		accountsWhereClause = fmt.Sprintf("where %s and %s", whereParty, whereMarket)
	} else if whereParty != "" {
		accountsWhereClause = fmt.Sprintf("where %s", whereParty)
	} else if whereMarket != "" {
		accountsWhereClause = fmt.Sprintf("where %s", whereMarket)
	}

	return fmt.Sprintf("where all_margin_levels.account_id  in (select id from accounts %s)", accountsWhereClause), bindVars
}

func (ml *MarginLevels) GetMarginLevelsByIDWithCursorPagination(ctx context.Context, partyID, marketID string, pagination entities.CursorPagination) ([]entities.MarginLevels, entities.PageInfo, error) {
	whereClause, bindVars := buildAccountWhereClause(partyID, marketID)

	query := fmt.Sprintf(`select distinct on (account_id) %s
		from all_margin_levels
		%s`, sqlMarginLevelColumns,
		whereClause)

	sorting, cmp, cursor := extractPaginationInfo(pagination)
	var err error
	mc := &entities.MarginCursor{}

	if cursor != "" {
		err = mc.Parse(cursor)
		if err != nil {
			return nil, entities.PageInfo{}, fmt.Errorf("parsing cursor: %w", err)
		}
	}

	builders := []CursorQueryParameter{
		NewCursorQueryParameter("account_id", sorting, cmp, mc.AccountID),
		NewCursorQueryParameter("vega_time", sorting, cmp, mc.VegaTime),
	}

	query, bindVars = orderAndPaginateWithCursor(query, pagination, builders, bindVars...)
	var marginLevels []entities.MarginLevels

	if err := pgxscan.Select(ctx, ml.Connection, &marginLevels, query, bindVars...); err != nil {
		return nil, entities.PageInfo{}, err
	}

	pagedMargins, pageInfo := entities.PageEntities[*v2.MarginEdge](marginLevels, pagination)
	return pagedMargins, pageInfo, nil
}
