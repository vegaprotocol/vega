package graphql

import (
	"vega/msg"
	"context"
	"vega/api"
	"time"
)

type resolverRoot struct {
	orderService api.OrderService
	tradeService api.TradeService
}

func NewResolverRoot(orderService api.OrderService, tradeService api.TradeService) *resolverRoot {
	return &resolverRoot{
		orderService: orderService,
		tradeService: tradeService,
	}
}

// BEGIN: Query Resolver

type MyQueryResolver resolverRoot

func (r *resolverRoot) Query() QueryResolver {
	return (*MyQueryResolver)(r)
}
func (r *resolverRoot) Order() OrderResolver {
	return (*MyOrderResolver)(r)
}
func (r *resolverRoot) Trade() TradeResolver {
	return (*MyTradeResolver)(r)
}
func (r *resolverRoot) Candle() CandleResolver {
	return (*MyCandleResolver)(r)
}

func (r *MyQueryResolver) Orders(ctx context.Context) ([]msg.Order, error) {
	orders, err := r.orderService.GetOrders(ctx,"BTC/DEC18", "", 99999)
	return orders, err
}

func (r *MyQueryResolver) Trades(ctx context.Context) ([]msg.Trade, error) {
	_, err := r.tradeService.GetTrades(ctx,"BTC/DEC18", 99999)
	return nil, err
}

func (r *MyQueryResolver) Candles(ctx context.Context) ([]msg.Candle, error) {
	const genesisTimeStr = "2018-07-09T12:00:00Z"
	genesisT, _ := time.Parse(time.RFC3339, genesisTimeStr)
	nowT := genesisT.Add(6 * time.Minute)
	since := nowT.Add(-5 * time.Minute)
	interval := uint64(60)

	_, err := r.tradeService.GetCandlesChart(ctx,"BTC/DEC18", since, interval)
	return nil, err
}

// END: Query Resolver



// BEGIN: Order Resolver

type MyOrderResolver resolverRoot

func (r *MyOrderResolver) Price(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Price), nil
}
func (r *MyOrderResolver) Type(ctx context.Context, obj *msg.Order) (OrderType, error) {
	return OrderType(obj.Type.String()), nil
}
func (r *MyOrderResolver) Side(ctx context.Context, obj *msg.Order) (Side, error) {
	return Side(obj.Side.String()), nil
}
func (r *MyOrderResolver) Market(ctx context.Context, obj *msg.Order) (Market, error) {
	return Market { obj.Market }, nil
}
func (r *MyOrderResolver) Size(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Size), nil
}
func (r *MyOrderResolver) Remaining(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Remaining), nil
}
func (r *MyOrderResolver) Timestamp(ctx context.Context, obj *msg.Order) (int, error) {
	return int(obj.Timestamp), nil
}

// END: Order Resolver



// BEGIN: Candle Resolver

type MyCandleResolver resolverRoot

func (r *MyCandleResolver) High(ctx context.Context, obj *msg.Candle) (int, error)   {
	 return int(obj.High), nil
}
func (r *MyCandleResolver) Low(ctx context.Context, obj *msg.Candle) (int, error)  {
	return int(obj.Low), nil
}
func (r *MyCandleResolver) Open(ctx context.Context, obj *msg.Candle) (int, error)  {
	return int(obj.Open), nil
}
func (r *MyCandleResolver) Close(ctx context.Context, obj *msg.Candle) (int, error)  {
	return int(obj.Close), nil
}
func (r *MyCandleResolver) Volume(ctx context.Context, obj *msg.Candle) (int, error) {
	return int(obj.Volume), nil
}
func (r *MyCandleResolver) OpenBlockNumber(ctx context.Context, obj *msg.Candle) (int, error) {
	return int(obj.OpenBlockNumber), nil
}
func (r *MyCandleResolver) CloseBlockNumber(ctx context.Context, obj *msg.Candle) (int, error) {
	return int(obj.CloseBlockNumber), nil
}

// END: Candle Resolver



// BEGIN: Trade Resolver

type MyTradeResolver resolverRoot

func (r *MyTradeResolver) Market(ctx context.Context, obj *msg.Trade) (Market, error) {
	return Market{ obj.Market }, nil
}
func (r *MyTradeResolver) Aggressor(ctx context.Context, obj *msg.Trade) (Side, error) {
	return Side(obj.Aggressor.String()), nil
}
func (r *MyTradeResolver) Price(ctx context.Context, obj *msg.Trade) (int, error) {
	return int(obj.Price), nil
}
func (r *MyTradeResolver) Size(ctx context.Context, obj *msg.Trade) (int, error) {
	return int(obj.Size), nil
}
func (r *MyTradeResolver) Timestamp(ctx context.Context, obj *msg.Trade) (int, error) {
	return int(obj.Timestamp), nil
}

// END: Trade Resolver


