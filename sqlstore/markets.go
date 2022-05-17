package sqlstore

import (
	"context"
	"fmt"
	"sync"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
)

type Markets struct {
	*ConnectionSource
	cache     map[string]entities.Market
	cacheLock sync.Mutex
}

const (
	sqlMarketsColumns = `id, vega_time, instrument_id, tradable_instrument, decimal_places,
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
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()

	query := fmt.Sprintf(`insert into markets(%s) 
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
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
	position_decimal_places=EXCLUDED.position_decimal_places;`, sqlMarketsColumns)

	defer metrics.StartSQLQuery("Markets", "Upsert")()
	if _, err := m.Connection.Exec(ctx, query, market.ID, market.VegaTime, market.InstrumentID, market.TradableInstrument, market.DecimalPlaces,
		market.Fees, market.OpeningAuction, market.PriceMonitoringSettings, market.LiquidityMonitoringParameters,
		market.TradingMode, market.State, market.MarketTimestamps, market.PositionDecimalPlaces); err != nil {
		err = fmt.Errorf("could not insert market into database: %w", err)
		return err
	}

	m.cache[market.ID.String()] = *market
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
	err := pgxscan.Get(ctx, m.Connection, &market, query, entities.NewMarketID(marketID))

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

func (m *Markets) GetAllPaged(ctx context.Context, marketID string, pagination entities.Pagination) ([]entities.Market, entities.PageInfo, error) {
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

	var pagedMarkets []entities.Market
	var pageInfo entities.PageInfo

	sorting, cmp, cursor := extractPaginationInfo(pagination)
	cursors := []CursorBuilder{NewCursorBuilder("vega_time", sorting, cmp, cursor)}
	query, args = orderAndPaginateWithCursor(query, pagination, cursors, args...)

	if err := pgxscan.Select(ctx, m.Connection, &markets, query, args...); err != nil {
		return pagedMarkets, pageInfo, err
	}

	pagedMarkets, pageInfo = entities.PageEntities(markets, pagination)
	return pagedMarkets, pageInfo, nil
}
