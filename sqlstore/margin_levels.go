package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
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

func (ml *MarginLevels) Flush(ctx context.Context) error {
	return ml.batcher.Flush(ctx, ml.pool)
}

func (ml *MarginLevels) GetMarginLevelsByID(ctx context.Context, partyID, marketID string, pagination entities.Pagination) ([]entities.MarginLevels, error) {
	party := entities.NewPartyID(partyID)
	market := entities.NewMarketID(marketID)

	var marginLevels []entities.MarginLevels

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

	whereClause := fmt.Sprintf("where all_margin_levels.account_id  in (select id from accounts %s)", accountsWhereClause)

	query := fmt.Sprintf(`select distinct on (account_id) %s
		from all_margin_levels
		%s
		order by account_id, vega_time desc`, sqlMarginLevelColumns,
		whereClause)

	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	err := pgxscan.Select(ctx, ml.Connection, &marginLevels, query, bindVars...)

	return marginLevels, err
}
