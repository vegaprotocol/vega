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
			t.BuyerMakerFeeReferralDiscount,
			t.BuyerInfrastructureFeeReferralDiscount,
			t.BuyerLiquidityFeeReferralDiscount,
			t.BuyerMakerFeeVolumeDiscount,
			t.BuyerInfrastructureFeeVolumeDiscount,
			t.BuyerLiquidityFeeVolumeDiscount,
			t.SellerMakerFeeReferralDiscount,
			t.SellerInfrastructureFeeReferralDiscount,
			t.SellerLiquidityFeeReferralDiscount,
			t.SellerMakerFeeVolumeDiscount,
			t.SellerInfrastructureFeeVolumeDiscount,
			t.SellerLiquidityFeeVolumeDiscount,
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
				"buyer_auction_batch", "seller_auction_batch", "buyer_maker_fee_referral_discount", "buyer_infrastructure_fee_referral_discount",
				"buyer_liquidity_fee_referral_discount", "buyer_maker_fee_volume_discount", "buyer_infrastructure_fee_volume_discount", "buyer_liquidity_fee_volume_discount",
				"seller_maker_fee_referral_discount", "seller_infrastructure_fee_referral_discount", "seller_liquidity_fee_referral_discount",
				"seller_maker_fee_volume_discount", "seller_infrastructure_fee_volume_discount", "seller_liquidity_fee_volume_discount",
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
	marketIDs []entities.MarketID,
	partyIDs []entities.PartyID,
	orderIDs []entities.OrderID,
	pagination entities.CursorPagination,
	dateRange entities.DateRange,
) ([]entities.Trade, entities.PageInfo, error) {
	args := []interface{}{}

	conditions := []string{}
	if len(marketIDs) > 0 {
		markets := make([][]byte, 0)
		for _, m := range marketIDs {
			bs, err := m.Bytes()
			if err != nil {
				return nil, entities.PageInfo{}, fmt.Errorf("received invalid market ID: %w", err)
			}
			markets = append(markets, bs)
		}
		conditions = append(conditions, fmt.Sprintf("market_id = ANY(%s::bytea[])", nextBindVar(&args, markets)))
	}

	if len(partyIDs) > 0 {
		parties := make([][]byte, 0)
		for _, p := range partyIDs {
			bs, err := p.Bytes()
			if err != nil {
				return nil, entities.PageInfo{}, fmt.Errorf("received invalid party ID: %w", err)
			}
			parties = append(parties, bs)
		}
		bindVar := nextBindVar(&args, parties)

		conditions = append(conditions, fmt.Sprintf("(buyer = ANY(%s::bytea[]) or seller = ANY(%s::bytea[]))", bindVar, bindVar))
	}

	if len(orderIDs) > 0 {
		orders := make([][]byte, 0)
		for _, o := range orderIDs {
			bs, err := o.Bytes()
			if err != nil {
				return nil, entities.PageInfo{}, fmt.Errorf("received invalid order ID: %w", err)
			}
			orders = append(orders, bs)
		}
		bindVar := nextBindVar(&args, orders)
		conditions = append(conditions, fmt.Sprintf("(buy_order = ANY(%s::bytea[]) or sell_order = ANY(%s::bytea[]))", bindVar, bindVar))
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

func (ts *Trades) GetLastTradeByMarket(ctx context.Context, market string) ([]entities.Trade, error) {
	query := `SELECT * from trades WHERE market_id=$1`
	args := []interface{}{entities.MarketID(market)}
	defer metrics.StartSQLQuery("Trades", "GetByMarket")()
	trades, err := ts.queryTrades(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade by market:%w", err)
	}

	return trades, nil
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

func (ts *Trades) queryTrades(ctx context.Context, query string, args []interface{}) ([]entities.Trade, error) {
	query, args = queryTradesLast(query, []string{"synthetic_time"}, args...)

	var trades []entities.Trade
	err := pgxscan.Select(ctx, ts.Connection, &trades, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying trades: %w", err)
	}
	return trades, nil
}

func queryTradesLast(query string, orderColumns []string, args ...interface{}) (string, []interface{}) {
	ordering := "DESC"

	sbOrderBy := strings.Builder{}

	if len(orderColumns) > 0 {
		sbOrderBy.WriteString("ORDER BY")

		sep := ""

		for _, column := range orderColumns {
			sbOrderBy.WriteString(fmt.Sprintf("%s %s %s", sep, column, ordering))
			sep = ","
		}
	}

	var paging string
	paging = fmt.Sprintf("%sOFFSET %s ", paging, nextBindVar(&args, 0))
	paging = fmt.Sprintf("%sLIMIT %s ", paging, nextBindVar(&args, 1))
	query = fmt.Sprintf("%s %s %s", query, sbOrderBy.String(), paging)

	return query, args
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
