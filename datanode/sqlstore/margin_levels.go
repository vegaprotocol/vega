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

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

var mlOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
	ColumnOrdering{Name: "account_id", Sorting: ASC},
}

type AccountSource interface {
	Query(filter entities.AccountFilter) ([]entities.Account, error)
}

type MarginLevels struct {
	*ConnectionSource
	batcher MapBatcher[entities.MarginLevelsKey, entities.MarginLevels]
}

const (
	sqlMarginLevelColumns = `account_id,timestamp,maintenance_margin,search_level,initial_margin,collateral_release_level,tx_hash,vega_time`
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
	return ml.batcher.Flush(ctx, ml.Connection)
}

func buildAccountWhereClause(partyID, marketID string) (string, []interface{}) {
	party := entities.PartyID(partyID)
	market := entities.MarketID(marketID)

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

	return fmt.Sprintf("where current_margin_levels.account_id  in (select id from accounts %s)", accountsWhereClause), bindVars
}

func (ml *MarginLevels) GetMarginLevelsByIDWithCursorPagination(ctx context.Context, partyID, marketID string, pagination entities.CursorPagination) ([]entities.MarginLevels, entities.PageInfo, error) {
	whereClause, bindVars := buildAccountWhereClause(partyID, marketID)

	query := fmt.Sprintf(`select %s
		from current_margin_levels
		%s`, sqlMarginLevelColumns,
		whereClause)

	var err error
	var pageInfo entities.PageInfo
	query, bindVars, err = PaginateQuery[entities.MarginCursor](query, bindVars, mlOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}
	var marginLevels []entities.MarginLevels

	if err = pgxscan.Select(ctx, ml.Connection, &marginLevels, query, bindVars...); err != nil {
		return nil, entities.PageInfo{}, err
	}

	pagedMargins, pageInfo := entities.PageEntities[*v2.MarginEdge](marginLevels, pagination)
	return pagedMargins, pageInfo, nil
}

func (ml *MarginLevels) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.MarginLevels, error) {
	var marginLevels []entities.MarginLevels
	query := fmt.Sprintf(`SELECT %s FROM margin_levels WHERE tx_hash = $1`, sqlMarginLevelColumns)

	if err := pgxscan.Select(ctx, ml.Connection, &marginLevels, query, txHash); err != nil {
		return nil, err
	}

	return marginLevels, nil
}
