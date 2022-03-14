package sqlstore

import (
	"context"
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"github.com/georgysavva/scany/pgxscan"
)

type Markets struct {
	*SQLStore
}

const (
	sqlMarketsColumns = `id, vega_time, instrument_id, tradable_instrument, decimal_places,
		fees, opening_auction, price_monitoring_settings, liquidity_monitoring_parameters,
		trading_mode, state, market_timestamps, position_decimal_places`
)

func NewMarkets(sqlStore *SQLStore) *Markets {
	return &Markets{
		SQLStore: sqlStore,
	}
}

func (m *Markets) Upsert(market *entities.Market) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.conf.Timeout.Duration)
	defer cancel()

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

	if _, err := m.pool.Exec(ctx, query, market.ID, market.VegaTime, market.InstrumentID, market.TradableInstrument, market.DecimalPlaces,
		market.Fees, market.OpeningAuction, market.PriceMonitoringSettings, market.LiquidityMonitoringParameters,
		market.TradingMode, market.State, market.MarketTimestamps, market.PositionDecimalPlaces); err != nil {
		err = fmt.Errorf("could not insert market into database: %w", err)
		return err
	}

	return nil
}

func (m *Markets) GetByID(ctx context.Context, marketID string) (entities.Market, error) {
	id, err := hex.DecodeString(marketID)

	if err != nil {
		return entities.Market{}, fmt.Errorf("bad ID (must be a hex string): %w", err)
	}

	var market entities.Market

	query := fmt.Sprintf(`select distinct on (id) %s 
from markets 
where id = $1
order by id, vega_time desc
`, sqlMarketsColumns)
	err = pgxscan.Get(ctx, m.pool, &market, query, id)

	return market, err
}

func (m *Markets) GetAll(ctx context.Context, pagination entities.Pagination) ([]entities.Market, error) {
	var markets []entities.Market
	query := fmt.Sprintf(`select distinct on (id) %s
from markets
order by id, vega_time desc
`, sqlMarketsColumns)

	query, _ = orderAndPaginateQuery(query, nil, pagination)

	err := pgxscan.Select(ctx, m.pool, &markets, query)

	return markets, err
}
