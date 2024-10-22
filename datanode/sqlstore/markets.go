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
	"errors"
	"fmt"
	"strings"
	"sync"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
	"golang.org/x/exp/maps"
)

type cacheKey struct {
	forwardOffset  int32
	backwardOffset int32
	forwardCursor  string
	backwardCursor string
	newestFirst    bool
	marketID       string
	includeSettled bool
}

type cacheValue struct {
	markets  []entities.Market
	pageInfo entities.PageInfo
}

func newCacheKey(marketID string, pagination entities.CursorPagination, includeSettled bool) cacheKey {
	k := cacheKey{
		marketID:       marketID,
		newestFirst:    pagination.NewestFirst,
		includeSettled: includeSettled,
	}

	if pagination.Forward != nil {
		if pagination.Forward.Limit != nil {
			k.forwardOffset = *pagination.Forward.Limit
		}
		if pagination.Forward.Cursor != nil {
			k.forwardCursor = pagination.Forward.Cursor.Value()
		}
	}

	if pagination.Backward != nil {
		if pagination.Backward.Limit != nil {
			k.backwardOffset = *pagination.Backward.Limit
		}
		if pagination.Backward.Cursor != nil {
			k.backwardCursor = pagination.Backward.Cursor.Value()
		}
	}

	return k
}

type Markets struct {
	*ConnectionSource
	cache        map[string]entities.Market
	cacheLock    sync.Mutex
	allCache     map[cacheKey]cacheValue
	allCacheLock sync.Mutex
}

var marketOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC},
}

var lineageOrdering = TableOrdering{
	ColumnOrdering{Name: "vega_time", Sorting: ASC, Prefix: "m"},
	// ColumnOrdering{Name: "id", Sorting: ASC, Prefix: "m"},
	// ColumnOrdering{Name: "id", Sorting: ASC, Prefix: "pc"},
}

const (
	sqlMarketsColumns = `id, tx_hash, vega_time, instrument_id, tradable_instrument, decimal_places,
		fees, opening_auction, price_monitoring_settings, liquidity_monitoring_parameters,
		trading_mode, state, market_timestamps, position_decimal_places, lp_price_range, linear_slippage_factor, quadratic_slippage_factor,
		parent_market_id, insurance_pool_fraction, liquidity_sla_parameters, liquidation_strategy, mark_price_configuration, tick_size, enable_tx_reordering, allowed_empty_amm_levels, allowed_sellers`
)

func NewMarkets(connectionSource *ConnectionSource) *Markets {
	return &Markets{
		ConnectionSource: connectionSource,
		cache:            make(map[string]entities.Market),
		allCache:         make(map[cacheKey]cacheValue),
	}
}

func (m *Markets) Upsert(ctx context.Context, market *entities.Market) error {
	query := fmt.Sprintf(`insert into markets(%s)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
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
	lp_price_range=EXCLUDED.lp_price_range,
	linear_slippage_factor=EXCLUDED.linear_slippage_factor,
    quadratic_slippage_factor=EXCLUDED.quadratic_slippage_factor,
    parent_market_id=EXCLUDED.parent_market_id,
    insurance_pool_fraction=EXCLUDED.insurance_pool_fraction,
	tx_hash=EXCLUDED.tx_hash,
    liquidity_sla_parameters=EXCLUDED.liquidity_sla_parameters,
	liquidation_strategy=EXCLUDED.liquidation_strategy,
	mark_price_configuration=EXCLUDED.mark_price_configuration,
	tick_size=EXCLUDED.tick_size,
	enable_tx_reordering=EXCLUDED.enable_tx_reordering,
	allowed_empty_amm_levels=EXCLUDED.allowed_empty_amm_levels,
	allowed_sellers=EXCLUDED.allowed_sellers;`, sqlMarketsColumns)

	defer metrics.StartSQLQuery("Markets", "Upsert")()

	if market.AllowedSellers == nil {
		market.AllowedSellers = []string{}
	}

	if _, err := m.Exec(ctx, query, market.ID, market.TxHash, market.VegaTime, market.InstrumentID, market.TradableInstrument, market.DecimalPlaces,
		market.Fees, market.OpeningAuction, market.PriceMonitoringSettings, market.LiquidityMonitoringParameters,
		market.TradingMode, market.State, market.MarketTimestamps, market.PositionDecimalPlaces, market.LpPriceRange,
		market.LinearSlippageFactor, market.QuadraticSlippageFactor, market.ParentMarketID, market.InsurancePoolFraction,
		market.LiquiditySLAParameters, market.LiquidationStrategy,
		market.MarkPriceConfiguration, market.TickSize, market.EnableTXReordering, market.AllowedEmptyAMMLevels, market.AllowedSellers); err != nil {
		err = fmt.Errorf("could not insert market into database: %w", err)
		return err
	}

	m.AfterCommit(ctx, func() {
		// delete cache
		m.cacheLock.Lock()
		defer m.cacheLock.Unlock()
		delete(m.cache, market.ID.String())

		m.allCacheLock.Lock()
		defer m.allCacheLock.Unlock()
		maps.Clear(m.allCache)
	})

	return nil
}

func getSelect() string {
	return `with lineage(market_id, parent_market_id) as (
	select market_id, parent_market_id
    from market_lineage
)
select mc.id,  mc.tx_hash,  mc.vega_time,  mc.instrument_id,  mc.tradable_instrument,  mc.decimal_places,
		mc.fees, mc.opening_auction, mc.price_monitoring_settings, mc.liquidity_monitoring_parameters,
		mc.trading_mode, mc.state, mc.market_timestamps, mc.position_decimal_places, mc.lp_price_range, mc.linear_slippage_factor, mc.quadratic_slippage_factor,
		mc.parent_market_id, mc.insurance_pool_fraction, ml.market_id as successor_market_id, mc.liquidity_sla_parameters, mc.liquidation_strategy, mc.mark_price_configuration, mc.tick_size, mc.enable_tx_reordering, mc.allowed_empty_amm_levels, mc.allowed_sellers
from markets_current mc
left join lineage ml on mc.id = ml.parent_market_id
`
}

func (m *Markets) GetByID(ctx context.Context, marketID string) (entities.Market, error) {
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()

	if market, ok := m.cache[marketID]; ok {
		return market, nil
	}

	defer metrics.StartSQLQuery("Markets", "GetByID")()
	market, err := m.queryByID(ctx, marketID)

	if err == nil {
		m.cache[marketID] = market
	}

	return market, m.wrapE(err)
}

// queryByID assumes the cache lock has been acquired, and the cache doesn't yet contain the requested market.
// This function does not access the cache, so technically it can be called without a cache lock, but it's best not to go down that route.
func (m *Markets) queryByID(ctx context.Context, mktID string) (entities.Market, error) {
	var market entities.Market
	query := fmt.Sprintf(`%s
where id = $1
order by id, vega_time desc
`, getSelect())

	err := pgxscan.Get(ctx, m.ConnectionSource, &market, query, entities.MarketID(mktID))

	return market, m.wrapE(err)
}

func (m *Markets) GetByIDs(ctx context.Context, markets []string) ([]entities.Market, error) {
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()
	ret := make([]entities.Market, 0, len(markets))
	missing := make([]string, 0, len(markets))
	for _, mid := range markets {
		if mkt, ok := m.cache[mid]; ok {
			ret = append(ret, mkt)
		} else {
			missing = append(missing, mid)
		}
	}
	if len(missing) == 0 {
		return ret, nil
	}
	for _, mid := range missing {
		mkt, err := m.queryByID(ctx, mid)
		// if a requested market couldn't be found, just return an error.
		if err != nil {
			return nil, err
		}
		m.cache[mid] = mkt
		ret = append(ret, mkt)
	}
	return ret, nil
}

func (m *Markets) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Market, error) {
	defer metrics.StartSQLQuery("Markets", "GetByTxHash")()

	var markets []entities.Market
	query := fmt.Sprintf(`%s where tx_hash = $1`, getSelect())
	err := pgxscan.Select(ctx, m.ConnectionSource, &markets, query, txHash)

	if err == nil {
		m.cacheLock.Lock()
		for _, market := range markets {
			m.cache[market.ID.String()] = market
		}
		m.cacheLock.Unlock()
	}

	return markets, m.wrapE(err)
}

// GetAllFees returns fee information for all markets.
// it returns a market entity with only id and fee.
// NB: it's not cached nor paged.
func (m *Markets) GetAllFees(ctx context.Context) ([]entities.Market, error) {
	markets := make([]entities.Market, 0)
	args := make([]interface{}, 0)
	query := `select mc.id,  mc.fees from markets_current mc where state != 'STATE_REJECTED' AND state != 'STATE_SETTLED' AND state != 'STATE_CLOSED' order by mc.id`
	err := pgxscan.Select(ctx, m.ConnectionSource, &markets, query, args...)
	return markets, err
}

func (m *Markets) GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination, includeSettled bool) ([]entities.Market, entities.PageInfo, error) {
	key := newCacheKey(marketID, pagination, includeSettled)
	m.allCacheLock.Lock()
	defer m.allCacheLock.Unlock()
	if value, ok := m.allCache[key]; ok {
		return value.markets, value.pageInfo, nil
	}

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

	settledClause := ""
	if !includeSettled {
		settledClause = " AND state != 'STATE_SETTLED' AND state != 'STATE_CLOSED'"
	}

	query := fmt.Sprintf(`%s
		where state != 'STATE_REJECTED' %s`, getSelect(), settledClause)

	var (
		pageInfo entities.PageInfo
		err      error
	)

	query, args, err = PaginateQuery[entities.MarketCursor](query, args, marketOrdering, pagination)
	if err != nil {
		return markets, pageInfo, err
	}

	if err = pgxscan.Select(ctx, m.ConnectionSource, &markets, query, args...); err != nil {
		return markets, pageInfo, err
	}

	markets, pageInfo = entities.PageEntities[*v2.MarketEdge](markets, pagination)

	m.allCache[key] = cacheValue{markets: markets, pageInfo: pageInfo}
	return markets, pageInfo, nil
}

func (m *Markets) ListSuccessorMarkets(ctx context.Context, marketID string, fullHistory bool, pagination entities.CursorPagination) ([]entities.SuccessorMarket, entities.PageInfo, error) {
	if marketID == "" {
		return nil, entities.PageInfo{}, errors.New("invalid market ID. Market ID cannot be empty")
	}

	// We paginate by market, so first we have to get all the markets and apply pagination to those first

	args := make([]interface{}, 0)

	lineageFilter := ""

	if !fullHistory {
		lineageFilter = "and vega_time >= (select vega_time from lineage_root)"
	}

	preQuery := fmt.Sprintf(`
with lineage_root(root_id, vega_time) as (
	select root_id, vega_time
	from market_lineage
	where market_id = %s
), lineage(successor_market_id, parent_id, root_id) as (
	select market_id, parent_market_id, root_id
	from market_lineage
	where root_id = (select root_id from lineage_root)
  %s
) `, nextBindVar(&args, entities.MarketID(marketID)), lineageFilter)

	query := `select m.*, s.successor_market_id
from markets_current m
join lineage l on l.successor_market_id = m.id
left join lineage s on l.successor_market_id = s.parent_id
`
	var markets []entities.Market
	var pageInfo entities.PageInfo
	var err error
	query, args, err = PaginateQuery[entities.SuccessorMarketCursor](query, args, lineageOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}

	query = fmt.Sprintf("%s %s", preQuery, query)

	if err = pgxscan.Select(ctx, m.ConnectionSource, &markets, query, args...); err != nil {
		return nil, entities.PageInfo{}, m.wrapE(err)
	}

	markets, pageInfo = entities.PageEntities[*v2.MarketEdge](markets, pagination)

	// Now that we have the markets we are going to return, we need to get all the related proposals where the parent market
	// is one of the markets we are returning. We will do this in one query and process the results in memory
	// rather than making a separate database query for each market in case the succession line becomes very long.

	parentMarketList := make([]string, 0)
	for _, m := range markets {
		parentMarketList = append(parentMarketList, m.ID.String())
	}

	var proposals []entities.Proposal

	proposalsQuery := fmt.Sprintf(`select * from proposals_current where terms->'newMarket'->'changes'->'successor'->>'parentMarketId' in ('%s') order by vega_time, id`, strings.Join(parentMarketList, "', '"))

	if err = pgxscan.Select(ctx, m.ConnectionSource, &proposals, proposalsQuery); err != nil {
		return nil, entities.PageInfo{}, m.wrapE(err)
	}

	edges := []entities.SuccessorMarket{}

	// Now we have the proposals, we need to create the successor market edges and add them to the market
	for _, m := range markets {
		edge := entities.SuccessorMarket{
			Market: m,
		}

		for i, p := range proposals {
			if p.Terms.ProposalTerms.GetNewMarket().Changes.Successor.ParentMarketId == m.ID.String() {
				edge.Proposals = append(edge.Proposals, &proposals[i])
			}
		}

		edges = append(edges, edge)
	}

	if len(markets) == 0 {
		// We do not have any markets in the given succession line, so we need to return the market
		// associated with the given market ID, which should be the parent market.
		market, err := m.GetByID(ctx, marketID)
		if err != nil {
			return nil, entities.PageInfo{}, err
		}

		edge := entities.SuccessorMarket{
			Market: market,
		}

		edges = append(edges, edge)
	}

	return edges, pageInfo, nil
}
