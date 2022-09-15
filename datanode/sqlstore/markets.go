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
	"sync"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"github.com/georgysavva/scany/pgxscan"
)

type Markets struct {
	*ConnectionSource
	cache     map[string]entities.Market
	cacheLock sync.Mutex
}

var marketOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

const (
	sqlMarketsColumns = `id, tx_hash, vega_time, instrument_id, tradable_instrument, decimal_places,
		fees, opening_auction, price_monitoring_settings, liquidity_monitoring_parameters,
		trading_mode, state, market_timestamps, position_decimal_places`
)

func NewMarkets(connectionSource *ConnectionSource) *Markets {
	return &Markets{
		ConnectionSource: connectionSource,
		cache:            make(map[string]entities.Market),
	}
}

func (m *Markets) Upsert(ctx context.Context, market *entities.Market) error {
	query := fmt.Sprintf(`insert into markets(%s)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
on conflict (id, vega_time) do update
set
	instrument_id=EXCLUDED.instrument_id,
	tradable_instrument=EXCLUDED.tradable_instrument,
	decimal_places=EXCLUDED.decimal_places,
	fees=EXCLUDED.fees,
	opening_auction=EXCLUDED.opening_auction,
	price_monitoring_settings=EXCLUDED.price_monitoring_settings,
	liquidity_monitoring_parameters=EXCLUDED.liquidity_monitoring_parameters,
	trading_mode=EXCLUDED.trading_mode,
	state=EXCLUDED.state,
	market_timestamps=EXCLUDED.market_timestamps,
	position_decimal_places=EXCLUDED.position_decimal_places,
	tx_hash=EXCLUDED.tx_hash;`, sqlMarketsColumns)

	defer metrics.StartSQLQuery("Markets", "Upsert")()
	if _, err := m.Connection.Exec(ctx, query, market.ID, market.TxHash, market.VegaTime, market.InstrumentID, market.TradableInstrument, market.DecimalPlaces,
		market.Fees, market.OpeningAuction, market.PriceMonitoringSettings, market.LiquidityMonitoringParameters,
		market.TradingMode, market.State, market.MarketTimestamps, market.PositionDecimalPlaces); err != nil {
		err = fmt.Errorf("could not insert market into database: %w", err)
		return err
	}

	m.AfterCommit(ctx, func() {
		// delete cache
		m.cacheLock.Lock()
		defer m.cacheLock.Unlock()
		delete(m.cache, market.ID.String())
	})

	return nil
}

func (m *Markets) GetByID(ctx context.Context, marketID string) (entities.Market, error) {
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()

	var market entities.Market

	if market, ok := m.cache[marketID]; ok {
		return market, nil
	}

	query := fmt.Sprintf(`select distinct on (id) %s
from markets
where id = $1
order by id, vega_time desc
`, sqlMarketsColumns)
	defer metrics.StartSQLQuery("Markets", "GetByID")()
	err := pgxscan.Get(ctx, m.Connection, &market, query, entities.MarketID(marketID))

	if err == nil {
		m.cache[marketID] = market
	}

	return market, err
}

func (m *Markets) GetAll(ctx context.Context, pagination entities.OffsetPagination) ([]entities.Market, error) {
	var markets []entities.Market
	query := fmt.Sprintf(`select distinct on (id) %s
from markets
order by id, vega_time desc
`, sqlMarketsColumns)

	query, _ = orderAndPaginateQuery(query, nil, pagination)

	defer metrics.StartSQLQuery("Markets", "GetAll")()
	err := pgxscan.Select(ctx, m.Connection, &markets, query)

	return markets, err
}

func (m *Markets) GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination) ([]entities.Market, entities.PageInfo, error) {
	if marketID != "" {
		market, err := m.GetByID(ctx, marketID)
		if err != nil {
			return nil, entities.PageInfo{}, err
		}

		return []entities.Market{market}, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     market.Cursor().Encode(),
			EndCursor:       market.Cursor().Encode(),
		}, nil
	}

	markets := make([]entities.Market, 0)
	args := make([]interface{}, 0)

	query := fmt.Sprintf(`select %s
		from markets_current`, sqlMarketsColumns)

	var (
		pageInfo entities.PageInfo
		err      error
	)

	query, args, err = PaginateQuery[entities.MarketCursor](query, args, marketOrdering, pagination)
	if err != nil {
		return markets, pageInfo, err
	}

	if err = pgxscan.Select(ctx, m.Connection, &markets, query, args...); err != nil {
		return markets, pageInfo, err
	}

	markets, pageInfo = entities.PageEntities[*v2.MarketEdge](markets, pagination)
	return markets, pageInfo, nil
}
