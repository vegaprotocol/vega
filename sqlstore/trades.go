package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
)

type Trades struct {
	*ConnectionSource
	trades []*entities.Trade
}

func NewTrades(connectionSource *ConnectionSource) *Trades {
	t := &Trades{
		ConnectionSource: connectionSource,
	}
	return t
}

func (ts *Trades) OnTimeUpdateEvent(ctx context.Context) error {
	var rows [][]interface{}
	for _, t := range ts.trades {
		rows = append(rows, []interface{}{
			t.SyntheticTime,
			t.VegaTime,
			t.SeqNum,
			t.ID,
			t.MarketID,
			t.Price,
			t.Size,
			t.Buyer,
			t.Seller,
			t.Aggressor,
			t.BuyOrder,
			t.SellOrder,
			t.Type,
			t.BuyerMakerFee,
			t.BuyerInfrastructureFee,
			t.BuyerLiquidityFee,
			t.SellerMakerFee,
			t.SellerInfrastructureFee,
			t.SellerLiquidityFee,
			t.BuyerAuctionBatch,
			t.SellerAuctionBatch,
		})
	}

	defer metrics.StartSQLQuery("Trades", "Flush")()

	if rows != nil {
		copyCount, err := ts.Connection.CopyFrom(
			ctx,
			pgx.Identifier{"trades"},
			[]string{
				"synthetic_time", "vega_time", "seq_num", "id", "market_id", "price", "size", "buyer", "seller",
				"aggressor", "buy_order", "sell_order", "type", "buyer_maker_fee", "buyer_infrastructure_fee",
				"buyer_liquidity_fee", "seller_maker_fee", "seller_infrastructure_fee", "seller_liquidity_fee",
				"buyer_auction_batch", "seller_auction_batch",
			},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("failed to copy trades into database:%w", err)
		}

		if copyCount != int64(len(rows)) {
			return fmt.Errorf("copied %d trade rows into the database, expected to copy %d", copyCount, len(rows))
		}
	}

	ts.trades = nil

	return nil
}

func (ts *Trades) Add(t *entities.Trade) error {
	ts.trades = append(ts.trades, t)
	return nil
}

func (ts *Trades) GetByMarket(ctx context.Context, market string, p entities.OffsetPagination) ([]entities.Trade, error) {
	query := `SELECT * from trades WHERE market_id=$1`
	args := []interface{}{entities.NewMarketID(market)}
	defer metrics.StartSQLQuery("Trades", "GetByMarket")()
	trades, err := ts.queryTrades(ctx, query, args, &p)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade by market:%w", err)
	}

	return trades, nil
}

func (ts *Trades) GetByMarketWithCursor(ctx context.Context, market string, pagination entities.Pagination) ([]entities.Trade, entities.PageInfo, error) {
	query := `SELECT * from trades WHERE market_id=$1`
	args := []interface{}{entities.NewMarketID(market)}
	trades, pageInfo, err := ts.queryTradesWithCursor(ctx, query, args, pagination)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("failed to get trade by market:%w", err)
	}

	return trades, pageInfo, nil
}

func (ts *Trades) GetByParty(ctx context.Context, party string, market *string, pagination entities.OffsetPagination) ([]entities.Trade, error) {
	args := []interface{}{entities.NewPartyID(party)}
	query := `SELECT * from trades WHERE buyer=$1 or seller=$1`

	defer metrics.StartSQLQuery("Trades", "GetByParty")()
	return ts.queryTradesWithMarketFilter(ctx, query, args, market, pagination)
}

func (ts *Trades) GetByPartyWithCursor(ctx context.Context, party string, market *string, pagination entities.Pagination) ([]entities.Trade, entities.PageInfo, error) {
	args := []interface{}{entities.NewPartyID(party)}
	query := `SELECT * from trades WHERE (buyer=$1 or seller=$1)`

	return ts.queryTradesWithMarketFilterAndCursor(ctx, query, args, market, pagination)
}

func (ts *Trades) GetByOrderID(ctx context.Context, order string, market *string, pagination entities.OffsetPagination) ([]entities.Trade, error) {
	args := []interface{}{entities.NewOrderID(order)}
	query := `SELECT * from trades WHERE buy_order=$1 or sell_order=$1`

	defer metrics.StartSQLQuery("Trades", "GetByOrderID")()
	return ts.queryTradesWithMarketFilter(ctx, query, args, market, pagination)
}

func (ts *Trades) queryTradesWithMarketFilter(ctx context.Context, query string, args []interface{}, market *string, p entities.OffsetPagination) ([]entities.Trade, error) {
	if market != nil && *market != "" {
		marketID := nextBindVar(&args, entities.NewMarketID(*market))
		query += ` AND market_id=` + marketID
	}

	trades, err := ts.queryTrades(ctx, query, args, &p)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades:%w", err)
	}

	return trades, nil
}

func (ts *Trades) queryTradesWithMarketFilterAndCursor(ctx context.Context, query string, args []interface{},
	market *string, cursor entities.Pagination,
) ([]entities.Trade, entities.PageInfo, error) {
	if market != nil && *market != "" {
		marketID := nextBindVar(&args, entities.NewMarketID(*market))
		query += ` AND market_id=` + marketID
	}

	trades, pageInfo, err := ts.queryTradesWithCursor(ctx, query, args, cursor)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("failed to query trades:%w", err)
	}

	return trades, pageInfo, nil
}

func (ts *Trades) queryTrades(ctx context.Context, query string, args []interface{}, p *entities.OffsetPagination) ([]entities.Trade, error) {
	if p != nil {
		query, args = orderAndPaginateQuery(query, []string{"synthetic_time"}, *p, args...)
	}

	var trades []entities.Trade
	err := pgxscan.Select(ctx, ts.Connection, &trades, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying trades: %w", err)
	}
	return trades, nil
}

func (ts *Trades) queryTradesWithCursor(ctx context.Context, query string, args []interface{}, pagination entities.Pagination) ([]entities.Trade, entities.PageInfo, error) {
	var err error

	query, args = orderAndPaginateWithCursor(query, pagination, "synthetic_time", args...)

	var trades []entities.Trade
	var pageInfo entities.PageInfo
	var pagedTrades []entities.Trade

	err = pgxscan.Select(ctx, ts.Connection, &trades, query, args...)
	if err != nil {
		return pagedTrades, pageInfo, fmt.Errorf("querying trades: %w", err)
	}

	pagedTrades, pageInfo = entities.PageEntities(trades, pagination)
	return pagedTrades, pageInfo, nil
}
