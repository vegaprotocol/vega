package sqlstore

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/entities"
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

// Add inserts a row to the trades table.
func (ts *Trades) Add(t *entities.Trade) error {
	ctx := context.Background()

	_, err := ts.pool.Exec(ctx,
		`INSERT INTO trades(vega_time, seq_num, id, market_id, price, size, buyer, seller, aggressor, buy_order, 
				sell_order, type, buyer_maker_fee, buyer_infrastructure_fee, buyer_liquidity_fee, 
                seller_maker_fee, seller_infrastructure_fee, seller_liquidity_fee,
				buyer_auction_batch, seller_auction_batch)
         		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
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
		t.SellerAuctionBatch)

	return err
}

func (ts *Trades) GetByMarket(ctx context.Context, market string, p entities.Pagination) ([]entities.Trade, error) {
	query := `SELECT * from trades WHERE market_id=$1`
	args := []interface{}{entities.NewMarketID(market)}
	trades, err := ts.queryTrades(ctx, query, args, &p)
	if err != nil {
		return nil, fmt.Errorf("failed to get trade by market:%w", err)
	}

	return trades, nil
}

func (ts *Trades) GetByParty(ctx context.Context, party string, market *string, pagination entities.Pagination) ([]entities.Trade, error) {
	args := []interface{}{entities.NewPartyID(party)}
	query := `SELECT * from trades WHERE buyer=$1 or seller=$1`

	return ts.queryTradesWithMarketFilter(ctx, query, args, market, pagination)
}

func (ts *Trades) GetByOrderID(ctx context.Context, order string, market *string, pagination entities.Pagination) ([]entities.Trade, error) {
	args := []interface{}{entities.NewOrderID(order)}
	query := `SELECT * from trades WHERE buy_order=$1 or sell_order=$1`
	return ts.queryTradesWithMarketFilter(ctx, query, args, market, pagination)
}

func (ts *Trades) queryTradesWithMarketFilter(ctx context.Context, query string, args []interface{}, market *string, p entities.Pagination) ([]entities.Trade, error) {
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

func (os *Trades) queryTrades(ctx context.Context, query string, args []interface{}, p *entities.Pagination) ([]entities.Trade, error) {
	if p != nil {
		query, args = paginateTradeQuery(query, args, *p)
	}

	var trades []entities.Trade
	err := pgxscan.Select(ctx, os.pool, &trades, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying trades: %w", err)
	}
	return trades, nil
}

func paginateTradeQuery(query string, args []interface{}, p entities.Pagination) (string, []interface{}) {
	dir := "ASC"
	if p.Descending {
		dir = "DESC"
	}

	var limit interface{} = nil
	if p.Limit != 0 {
		limit = p.Limit
	}

	query = fmt.Sprintf(" %s ORDER BY vega_time %s, seq_num %s LIMIT %s OFFSET %s",
		query, dir, dir, nextBindVar(&args, limit), nextBindVar(&args, p.Skip))

	return query, args
}
