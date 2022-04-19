package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type MarginLevels struct {
	*SQLStore
	batcher MapBatcher[entities.MarginLevelsKey, entities.MarginLevels]
}

const (
	sqlMarginLevelColumns = `market_id,asset_id,party_id,timestamp,maintenance_margin,search_level,initial_margin,collateral_release_level,vega_time`
)

func NewMarginLevels(sqlStore *SQLStore) *MarginLevels {
	return &MarginLevels{
		SQLStore: sqlStore,
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
	timeoutCtx, cancel := context.WithTimeout(ctx, ml.conf.Timeout.Duration)
	defer cancel()
	return ml.batcher.Flush(timeoutCtx, ml.pool)
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

	whereClause := ""

	if whereParty != "" && whereMarket != "" {
		whereClause = fmt.Sprintf("where %s and %s", whereParty, whereMarket)
	} else if whereParty != "" {
		whereClause = fmt.Sprintf("where %s", whereParty)
	} else if whereMarket != "" {
		whereClause = fmt.Sprintf("where %s", whereMarket)
	}

	query := fmt.Sprintf(`select distinct on (party_id, market_id) %s
		from margin_levels
		%s
		order by party_id, market_id, vega_time desc, asset_id`, sqlMarginLevelColumns,
		whereClause)

	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	err := pgxscan.Select(ctx, ml.pool, &marginLevels, query, bindVars...)

	return marginLevels, err
}
