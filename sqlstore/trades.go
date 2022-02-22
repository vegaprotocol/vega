package sqlstore

import (
	"context"
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/protos/vega"

	"github.com/georgysavva/scany/pgxscan"
)

type Trades struct {
	*SQLStore
}

func NewTrades(sqlStore *SQLStore) *Trades {
	t := &Trades{
		SQLStore: sqlStore,
	}
	return t
}

const sqlColumns = `vega_time, sequence_num, id, market_id, price, size, buyer, seller, aggressor, buy_order, sell_order,
				type, buyer_maker_fee, buyer_infrastructure_fee, buyer_liquidity_fee, 
                seller_maker_fee, seller_infrastructure_fee, seller_liquidity_fee,
                buyer_auction_batch, seller_auction_batch`

// Add inserts a row to the trades table.
func (ts *Trades) Add(t *entities.Trade) error {
	ctx := context.Background()

	_, err := ts.pool.Exec(ctx,
		`INSERT INTO trades(`+sqlColumns+`)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
		t.VegaTime,
		t.SequenceNum,
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
		t.SellerAuctionBatch)

	return err
}

func (ts *Trades) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) ([]*vega.Trade, error) {
	p := entities.Pagination{
		Skip:       skip,
		Limit:      limit,
		Descending: descending,
	}

	marketId, err := hex.DecodeString(market)
	if err != nil {
		return nil, err
	}

	query := `SELECT * from trades WHERE market_id=$1`
	args := []interface{}{marketId}
	trades, err := ts.queryTrades(ctx, query, args, &p)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade by market:%w", err)
	}

	return toProtoTrades(trades), nil
}

func (ts *Trades) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, market *string) ([]*vega.Trade, error) {
	query := `SELECT * from trades WHERE buyer=$1 or seller=$1`

	return ts.queryTradesWithMarketFilter(ctx, party, skip, limit, descending, market, query)
}

func (ts *Trades) GetByOrderID(ctx context.Context, order string, skip, limit uint64, descending bool, market *string) ([]*vega.Trade, error) {
	query := `SELECT * from trades WHERE buy_order=$1 or sell_order=$1`

	return ts.queryTradesWithMarketFilter(ctx, order, skip, limit, descending, market, query)
}

func (ts *Trades) queryTradesWithMarketFilter(ctx context.Context, idString string, skip uint64, limit uint64, descending bool, market *string, query string) ([]*vega.Trade, error) {
	p := entities.Pagination{
		Skip:       skip,
		Limit:      limit,
		Descending: descending,
	}

	id, err := hex.DecodeString(idString)
	if err != nil {
		return nil, err
	}

	args := []interface{}{id}

	if market != nil && *market != "" {

		marketId, err := hex.DecodeString(*market)
		if err != nil {
			return nil, err
		}

		query += ` AND market_id=$2`
		args = append(args, marketId)
	}

	trades, err := ts.queryTrades(ctx, query, args, &p)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades:%w", err)
	}

	return toProtoTrades(trades), nil
}

func toProtoTrades(trades []entities.Trade) []*vega.Trade {
	var protos []*vega.Trade
	for _, t := range trades {
		protos = append(protos, entities.TradeToProto(&t))
	}

	return protos
}

func (os *Trades) queryTrades(ctx context.Context, query string, args []interface{}, p *entities.Pagination) ([]entities.Trade, error) {
	if p != nil {
		query, args = paginateOrderQuery(query, args, *p)
	}

	trades := []entities.Trade{}
	err := pgxscan.Select(ctx, os.pool, &trades, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying trades: %w", err)
	}
	return trades, nil
}

func paginateOrderQuery(query string, args []interface{}, p entities.Pagination) (string, []interface{}) {
	dir := "ASC"
	if p.Descending {
		dir = "DESC"
	}

	var limit interface{} = nil
	if p.Limit != 0 {
		limit = p.Limit
	}

	query = fmt.Sprintf(" %s ORDER BY vega_time %s, sequence_num %s LIMIT %s OFFSET %s",
		query, dir, dir, nextBindVar(&args, limit), nextBindVar(&args, p.Skip))

	return query, args
}
