package sqlstore

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
)

type MarginLevels struct {
	*SQLStore
	columns      []string
	marginLevels []*entities.MarginLevels
}

const (
	sqlMarginLevelColumns = `market_id,asset_id,party_id,timestamp,maintenance_margin,search_level,initial_margin,collateral_release_level,vega_time,synthetic_time,seq_num`
)

func NewMarginLevels(sqlStore *SQLStore) *MarginLevels {
	return &MarginLevels{
		SQLStore: sqlStore,
		columns:  strings.Split(sqlMarginLevelColumns, ","),
	}
}

func (ml *MarginLevels) Add(marginLevel *entities.MarginLevels) error {
	ml.marginLevels = append(ml.marginLevels, marginLevel)
	return nil
}

func (ml *MarginLevels) OnTimeUpdateEvent(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, ml.conf.Timeout.Duration)
	defer cancel()

	var rows [][]interface{}
	for _, data := range ml.marginLevels {
		rows = append(rows, []interface{}{
			data.MarketID,
			data.AssetID,
			data.PartyID,
			data.Timestamp,
			data.MaintenanceMargin,
			data.SearchLevel,
			data.InitialMargin,
			data.CollateralReleaseLevel,
			data.VegaTime,
			data.SyntheticTime,
			data.SeqNum,
		})
	}

	if rows != nil {
		copyCount, err := ml.pool.CopyFrom(
			timeoutCtx,
			pgx.Identifier{"margin_levels"},
			ml.columns,
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("failed to copy margin level data into database: %w", err)
		}

		expectedCount := int64(len(rows))

		if copyCount != expectedCount {
			return fmt.Errorf("copied %d margin level rows into the database, expected to copy %d", copyCount, expectedCount)
		}
	}

	ml.marginLevels = nil
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
		order by party_id, market_id, synthetic_time desc, asset_id`, sqlMarginLevelColumns,
		whereClause)

	query, bindVars = orderAndPaginateQuery(query, nil, pagination, bindVars...)
	err := pgxscan.Select(ctx, ml.pool, &marginLevels, query, bindVars...)

	return marginLevels, err
}
