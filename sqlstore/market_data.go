package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
)

type MarketData struct {
	*ConnectionSource
	columns    []string
	marketData []*entities.MarketData
}

var ErrInvalidDateRange = errors.New("invalid date range, end date must be after start date")

func NewMarketData(connectionSource *ConnectionSource) *MarketData {
	return &MarketData{
		ConnectionSource: connectionSource,
		columns: []string{
			"synthetic_time", "vega_time", "seq_num",
			"market", "mark_price", "best_bid_price", "best_bid_volume",
			"best_offer_price", "best_offer_volume", "best_static_bid_price", "best_static_bid_volume",
			"best_static_offer_price", "best_static_offer_volume", "mid_price", "static_mid_price",
			"open_interest", "auction_end", "auction_start", "indicative_price", "indicative_volume",
			"market_trading_mode", "auction_trigger", "extension_trigger", "target_stake",
			"supplied_stake", "price_monitoring_bounds", "market_value_proxy", "liquidity_provider_fee_shares",
		},
	}
}

func (md *MarketData) Add(data *entities.MarketData) error {
	md.marketData = append(md.marketData, data)
	return nil
}

func (md *MarketData) Flush(ctx context.Context) ([]*entities.MarketData, error) {
	var rows [][]interface{}
	for _, data := range md.marketData {
		rows = append(rows, []interface{}{
			data.SyntheticTime, data.VegaTime, data.SeqNum,
			data.Market, data.MarkPrice,
			data.BestBidPrice, data.BestBidVolume, data.BestOfferPrice, data.BestOfferVolume,
			data.BestStaticBidPrice, data.BestStaticBidVolume, data.BestStaticOfferPrice, data.BestStaticOfferVolume,
			data.MidPrice, data.StaticMidPrice, data.OpenInterest, data.AuctionEnd,
			data.AuctionStart, data.IndicativePrice, data.IndicativeVolume, data.MarketTradingMode,
			data.AuctionTrigger, data.ExtensionTrigger, data.TargetStake, data.SuppliedStake,
			data.PriceMonitoringBounds, data.MarketValueProxy, data.LiquidityProviderFeeShares,
		})
	}
	defer metrics.StartSQLQuery("MarketData", "Flush")()
	if rows != nil {
		copyCount, err := md.Connection.CopyFrom(
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
	md.log.Debug("Retrieving market data from Postgres", logging.String("market-id", marketID))

	var marketData entities.MarketData
	query := "select * from market_data_snapshot where market = $1"

	err := pgxscan.Get(ctx, md.Connection, &marketData, query, entities.NewMarketID(marketID))

	return marketData, err
}

func (md *MarketData) GetMarketsData(ctx context.Context) ([]entities.MarketData, error) {
	md.log.Debug("Retrieving markets data from Postgres")

	var marketData []entities.MarketData
	query := "select * from market_data_snapshot"

	defer metrics.StartSQLQuery("MarketData", "GetMarketsData")()
	err := pgxscan.Select(ctx, md.Connection, &marketData, query)

	return marketData, err
}

func (md *MarketData) GetBetweenDatesByID(ctx context.Context, marketID string, start, end time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error) {
	if end.Before(start) {
		return nil, ErrInvalidDateRange
	}

	return md.getBetweenDatesByID(ctx, marketID, &start, &end, pagination)
}

func (md *MarketData) GetFromDateByID(ctx context.Context, marketID string, start time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error) {
	return md.getBetweenDatesByID(ctx, marketID, &start, nil, pagination)
}

func (md *MarketData) GetToDateByID(ctx context.Context, marketID string, end time.Time, pagination entities.OffsetPagination) ([]entities.MarketData, error) {
	return md.getBetweenDatesByID(ctx, marketID, nil, &end, pagination)
}

func (md *MarketData) getBetweenDatesByID(ctx context.Context, marketID string, start, end *time.Time, pagination entities.OffsetPagination) (results []entities.MarketData, err error) {
	defer metrics.StartSQLQuery("MarketData", "getBetweenDatesByID")()
	market := entities.NewMarketID(marketID)

	selectStatement := `select * from market_data`

	if start != nil && end != nil {
		query, args := orderAndPaginateQuery(
			fmt.Sprintf(`%s where market = $1 and synthetic_time between $2 and $3`, selectStatement),
			[]string{"synthetic_time"}, pagination,
			market, *start, *end)

		err = pgxscan.Select(ctx, md.Connection, &results, query, args...)
	} else if start != nil && end == nil {
		query, args := orderAndPaginateQuery(fmt.Sprintf(`%s where market = $1 and vega_time >= $2`, selectStatement),
			[]string{"synthetic_time"}, pagination,
			market, *start)

		err = pgxscan.Select(ctx, md.Connection, &results, query, args...)
	} else if start == nil && end != nil {
		query, args := orderAndPaginateQuery(fmt.Sprintf(`%s where market = $1 and vega_time <= $2`, selectStatement),
			[]string{"synthetic_time"}, pagination,
			market, *end)

		err = pgxscan.Select(ctx, md.Connection, &results, query, args...)
	}

	return results, err
}
