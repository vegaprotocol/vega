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
	"strings"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
)

const tradesFilterDateColumn = "synthetic_time"

type Trades struct {
	*ConnectionSource
	trades []*entities.Trade
}

var tradesOrdering = TableOrdering{
	ColumnOrdering{Name: "synthetic_time", Sorting: ASC},
}

func NewTrades(connectionSource *ConnectionSource) *Trades {
	t := &Trades{
		ConnectionSource: connectionSource,
	}
	return t
}

func (ts *Trades) Flush(ctx context.Context) ([]*entities.Trade, error) {
	rows := make([][]interface{}, 0, len(ts.trades))
	for _, t := range ts.trades {
		rows = append(rows, []interface{}{
			t.SyntheticTime,
			t.TxHash,
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
				"synthetic_time", "tx_hash", "vega_time", "seq_num", "id", "market_id", "price", "size", "buyer", "seller",
				"aggressor", "buy_order", "sell_order", "type", "buyer_maker_fee", "buyer_infrastructure_fee",
				"buyer_liquidity_fee", "seller_maker_fee", "seller_infrastructure_fee", "seller_liquidity_fee",
				"buyer_auction_batch", "seller_auction_batch",
			},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to copy trades into database:%w", err)
		}

		if copyCount != int64(len(rows)) {
			return nil, fmt.Errorf("copied %d trade rows into the database, expected to copy %d", copyCount, len(rows))
		}
	}

	flushed := ts.trades
	ts.trades = nil

	return flushed, nil
}

func (ts *Trades) Add(t *entities.Trade) error {
	ts.trades = append(ts.trades, t)
	return nil
}

func (ts *Trades) List(ctx context.Context,
	marketID entities.MarketID,
	partyID entities.PartyID,
	orderID entities.OrderID,
	pagination entities.CursorPagination,
	dateRange entities.DateRange,
) ([]entities.Trade, entities.PageInfo, error) {
	args := []interface{}{}

	conditions := []string{}
	if marketID.String() != "" {
		conditions = append(conditions, fmt.Sprintf("market_id=%s", nextBindVar(&args, marketID)))
	}

	if partyID.String() != "" {
		bindVar := nextBindVar(&args, partyID)
		conditions = append(conditions, fmt.Sprintf("(buyer=%s or seller=%s)", bindVar, bindVar))
	}

	if orderID.String() != "" {
		bindVar := nextBindVar(&args, orderID)
		conditions = append(conditions, fmt.Sprintf("(buy_order=%s or sell_order=%s)", bindVar, bindVar))
	}

	query := `SELECT * from trades`
	first := true
	if len(conditions) > 0 {
		query = fmt.Sprintf("%s WHERE %s", query, strings.Join(conditions, " AND "))
		first = false
	}
	query, args = filterDateRange(query, tradesFilterDateColumn, dateRange, first, args...)

	trades, pageInfo, err := ts.queryTradesWithCursorPagination(ctx, query, args, pagination)
	if err != nil {
		return nil, pageInfo, fmt.Errorf("failed to get trade by market:%w", err)
	}

	return trades, pageInfo, nil
}

func (ts *Trades) GetByMarket(ctx context.Context, market string, p entities.OffsetPagination) ([]entities.Trade, error) {
	query := `SELECT * from trades WHERE market_id=$1`
	args := []interface{}{entities.MarketID(market)}
	defer metrics.StartSQLQuery("Trades", "GetByMarket")()
	trades, err := ts.queryTrades(ctx, query, args, &p)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade by market:%w", err)
	}

	return trades, nil
}

func (ts *Trades) GetByParty(ctx context.Context, party string, market *string, pagination entities.OffsetPagination) ([]entities.Trade, error) {
	args := []interface{}{entities.PartyID(party)}
	query := `SELECT * from trades WHERE buyer=$1 or seller=$1`

	defer metrics.StartSQLQuery("Trades", "GetByParty")()
	return ts.queryTradesWithMarketFilter(ctx, query, args, market, pagination)
}

func (ts *Trades) GetByOrderID(ctx context.Context, order string, market *string, pagination entities.OffsetPagination) ([]entities.Trade, error) {
	args := []interface{}{entities.OrderID(order)}
	query := `SELECT * from trades WHERE buy_order=$1 or sell_order=$1`

	defer metrics.StartSQLQuery("Trades", "GetByOrderID")()
	return ts.queryTradesWithMarketFilter(ctx, query, args, market, pagination)
}

func (ts *Trades) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Trade, error) {
	defer metrics.StartSQLQuery("Trades", "GetByTxHash")()
	query := `SELECT * from trades WHERE tx_hash=$1`

	var trades []entities.Trade
	err := pgxscan.Select(ctx, ts.Connection, &trades, query, txHash)
	if err != nil {
		return nil, fmt.Errorf("querying trades: %w", err)
	}

	return trades, nil
}

func (ts *Trades) queryTradesWithMarketFilter(ctx context.Context, query string, args []interface{}, market *string, p entities.OffsetPagination) ([]entities.Trade, error) {
	if market != nil && *market != "" {
		marketID := nextBindVar(&args, entities.MarketID(*market))
		query += ` AND market_id=` + marketID
	}

	trades, err := ts.queryTrades(ctx, query, args, &p)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades:%w", err)
	}

	return trades, nil
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

func (ts *Trades) queryTradesWithCursorPagination(ctx context.Context, query string, args []interface{}, pagination entities.CursorPagination) ([]entities.Trade, entities.PageInfo, error) {
	var (
		err      error
		pageInfo entities.PageInfo
	)

	query, args, err = PaginateQuery[entities.TradeCursor](query, args, tradesOrdering, pagination)
	if err != nil {
		return nil, pageInfo, err
	}
	var trades []entities.Trade

	err = pgxscan.Select(ctx, ts.Connection, &trades, query, args...)
	if err != nil {
		return trades, pageInfo, fmt.Errorf("querying trades: %w", err)
	}

	trades, pageInfo = entities.PageEntities[*v2.TradeEdge](trades, pagination)
	return trades, pageInfo, nil
}
