package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type MarginLevels struct {
	*SQLStore
}

const (
	sqlMarginLevelColumns = `market_id, asset_id, party_id, maintenance_margin, search_level, initial_margin,
		collateral_release_level, timestamp, vega_time`
)

func NewMarginLevels(sqlStore *SQLStore) *MarginLevels {
	return &MarginLevels{
		SQLStore: sqlStore,
	}
}

func (ml *MarginLevels) Upsert(marginLevel *entities.MarginLevels) error {
	ctx, cancel := context.WithTimeout(context.Background(), ml.conf.Timeout.Duration)
	defer cancel()

	query := fmt.Sprintf(`insert into margin_levels(%s)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
on conflict (market_id, asset_id, party_id, vega_time) do update
set
	maintenance_margin=EXCLUDED.maintenance_margin,
	search_level=EXCLUDED.search_level,
	initial_margin=EXCLUDED.initial_margin,
	collateral_release_level=EXCLUDED.collateral_release_level,
	timestamp=EXCLUDED.timestamp`, sqlMarginLevelColumns)

	if _, err := ml.pool.Exec(ctx, query, marginLevel.MarketID, marginLevel.AssetID, marginLevel.PartyID,
		marginLevel.MaintenanceMargin, marginLevel.SearchLevel, marginLevel.InitialMargin,
		marginLevel.CollateralReleaseLevel, marginLevel.Timestamp, marginLevel.VegaTime); err != nil {
		return err
	}

	return nil
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
