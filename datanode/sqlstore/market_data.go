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
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
)

var marketdataOrdering = TableOrdering{
	ColumnOrdering{Name: "synthetic_time", Sorting: ASC},
}

type MarketData struct {
	*ConnectionSource
	columns    []string
	marketData []*entities.MarketData
}

var ErrInvalidDateRange = errors.New("invalid date range, end date must be after start date")

const selectMarketDataColumns = `synthetic_time, tx_hash, vega_time, seq_num,
			market, mark_price, best_bid_price, best_bid_volume,
			best_offer_price, best_offer_volume, best_static_bid_price, best_static_bid_volume,
			best_static_offer_price, best_static_offer_volume, mid_price, static_mid_price,
			open_interest, auction_end, auction_start, indicative_price, indicative_volume,
			market_trading_mode, auction_trigger, extension_trigger, target_stake,
			supplied_stake, price_monitoring_bounds, market_value_proxy, liquidity_provider_fee_shares,
			market_state, next_mark_to_market, coalesce(market_growth, 0) as market_growth,
			coalesce(last_traded_price, 0) as last_traded_price, product_data, liquidity_provider_sla, next_network_closeout, mark_price_type, mark_price_state, active_protocol_automated_purchase`

func NewMarketData(connectionSource *ConnectionSource) *MarketData {
	return &MarketData{
		ConnectionSource: connectionSource,
		columns: []string{
			"synthetic_time", "tx_hash", "vega_time", "seq_num",
			"market", "mark_price", "best_bid_price", "best_bid_volume",
			"best_offer_price", "best_offer_volume", "best_static_bid_price", "best_static_bid_volume",
			"best_static_offer_price", "best_static_offer_volume", "mid_price", "static_mid_price",
			"open_interest", "auction_end", "auction_start", "indicative_price", "indicative_volume",
			"market_trading_mode", "auction_trigger", "extension_trigger", "target_stake",
			"supplied_stake", "price_monitoring_bounds", "market_value_proxy", "liquidity_provider_fee_shares",
			"market_state", "next_mark_to_market", "market_growth", "last_traded_price", "product_data",
			"liquidity_provider_sla", "next_network_closeout", "mark_price_type", "mark_price_state", "active_protocol_automated_purchase",
		},
	}
}

func (md *MarketData) Add(data *entities.MarketData) error {
	md.marketData = append(md.marketData, data)
	return nil
}

func (md *MarketData) Flush(ctx context.Context) ([]*entities.MarketData, error) {
	rows := make([][]interface{}, 0, len(md.marketData))
	for _, data := range md.marketData {
		rows = append(rows, []interface{}{
			data.SyntheticTime, data.TxHash, data.VegaTime, data.SeqNum,
			data.Market, data.MarkPrice,
			data.BestBidPrice, data.BestBidVolume, data.BestOfferPrice, data.BestOfferVolume,
			data.BestStaticBidPrice, data.BestStaticBidVolume, data.BestStaticOfferPrice, data.BestStaticOfferVolume,
			data.MidPrice, data.StaticMidPrice, data.OpenInterest, data.AuctionEnd,
			data.AuctionStart, data.IndicativePrice, data.IndicativeVolume, data.MarketTradingMode,
			data.AuctionTrigger, data.ExtensionTrigger, data.TargetStake, data.SuppliedStake,
			data.PriceMonitoringBounds, data.MarketValueProxy, data.LiquidityProviderFeeShares, data.MarketState,
			data.NextMarkToMarket, data.MarketGrowth, data.LastTradedPrice,
			data.ProductData, data.LiquidityProviderSLA, data.NextNetworkCloseout, data.MarkPriceType, data.MarkPriceState, data.ActiveProtocolAutomatedPurchase,
		})
	}
	defer metrics.StartSQLQuery("MarketData", "Flush")()
	if rows != nil {
		copyCount, err := md.CopyFrom(
			ctx,
			pgx.Identifier{"market_data"}, md.columns, pgx.CopyFromRows(rows),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to copy market data into database:%w", err)
		}

		if copyCount != int64(len(rows)) {
			return nil, fmt.Errorf("copied %d market data rows into the database, expected to copy %d", copyCount, len(rows))
		}
	}

	flushed := md.marketData
	md.marketData = nil

	return flushed, nil
}

func (md *MarketData) GetMarketDataByID(ctx context.Context, marketID string) (entities.MarketData, error) {
	defer metrics.StartSQLQuery("MarketData", "GetMarketDataByID")()
	md.log.Debug("Retrieving market data from Postgres", logging.String("market-id", marketID))

	var marketData entities.MarketData
	query := fmt.Sprintf("select %s from current_market_data where market = $1", selectMarketDataColumns)
	return marketData, md.wrapE(pgxscan.Get(ctx, md.ConnectionSource, &marketData, query, entities.MarketID(marketID)))
}

func (md *MarketData) GetMarketsData(ctx context.Context) ([]entities.MarketData, error) {
	md.log.Debug("Retrieving markets data from Postgres")

	var marketData []entities.MarketData
	query := fmt.Sprintf("select %s from current_market_data", selectMarketDataColumns)

	defer metrics.StartSQLQuery("MarketData", "GetMarketsData")()
	err := pgxscan.Select(ctx, md.ConnectionSource, &marketData, query)

	return marketData, err
}

func (md *MarketData) GetHistoricMarketData(ctx context.Context, marketID string, start, end *time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error) {
	if start != nil && end != nil && end.Before(*start) {
		return nil, entities.PageInfo{}, ErrInvalidDateRange
	}

	switch p := pagination.(type) {
	case entities.CursorPagination:
		return md.getHistoricMarketData(ctx, marketID, start, end, p)
	default:
		panic("unsupported pagination")
	}
}

func (md *MarketData) getHistoricMarketData(ctx context.Context, marketID string, start, end *time.Time, pagination entities.CursorPagination) ([]entities.MarketData, entities.PageInfo, error) {
	defer metrics.StartSQLQuery("MarketData", "getHistoricMarketData")()
	market := entities.MarketID(marketID)

	selectStatement := fmt.Sprintf(`select %s from market_data`, selectMarketDataColumns)
	args := make([]interface{}, 0)

	var (
		query    string
		err      error
		pageInfo entities.PageInfo
	)

	switch {
	case start != nil && end != nil:
		query = fmt.Sprintf(`%s where market = %s and vega_time between %s and %s`, selectStatement,
			nextBindVar(&args, market),
			nextBindVar(&args, *start),
			nextBindVar(&args, *end),
		)
	case start != nil:
		query = fmt.Sprintf(`%s where market = %s and vega_time >= %s`, selectStatement,
			nextBindVar(&args, market),
			nextBindVar(&args, *start))
	case end != nil:
		query = fmt.Sprintf(`%s where market = %s and vega_time <= %s`, selectStatement,
			nextBindVar(&args, market),
			nextBindVar(&args, *end))
	default:
		query = fmt.Sprintf(`%s where market = %s`, selectStatement,
			nextBindVar(&args, market))
		// We want to restrict to just the last price update so we can override the pagination and force it to return just the 1 result
		first := ptr.From(int32(1))
		if pagination, err = entities.NewCursorPagination(first, nil, nil, nil, true); err != nil {
			return nil, pageInfo, err
		}
	}

	query, args, err = PaginateQuery[entities.MarketDataCursor](query, args, marketdataOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}
	var pagedData []entities.MarketData

	if err = pgxscan.Select(ctx, md.ConnectionSource, &pagedData, query, args...); err != nil {
		return pagedData, pageInfo, err
	}

	pagedData, pageInfo = entities.PageEntities[*v2.MarketDataEdge](pagedData, pagination)

	return pagedData, pageInfo, nil
}
