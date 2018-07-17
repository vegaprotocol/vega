package graphql

import (
	"context"
	"fmt"
	"time"
	"vega/api"
	"vega/msg"
)

type resolverRoot struct {
	orderService api.OrderService
	tradeService api.TradeService
	//observers    map[string]chan msg.Candle
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
func (r *resolverRoot) Subscription() SubscriptionResolver {
	return (*MySubscriptionResolver)(r)
}

func (r *MyQueryResolver) Orders(ctx context.Context) ([]msg.Order, error) {
	orders, err := r.orderService.GetOrders(ctx, "BTC/DEC18", "", 99999)
	return orders, err
}

func (r *MyQueryResolver) Trades(ctx context.Context) ([]msg.Trade, error) {
	_, err := r.tradeService.GetTrades(ctx, "BTC/DEC18", 99999)
	return nil, err
}

func (r *MyQueryResolver) Candles(ctx context.Context) ([]msg.Candle, error) {
	const genesisTimeStr = "2018-07-09T12:00:00Z"
	genesisT, _ := time.Parse(time.RFC3339, genesisTimeStr)
	nowT := genesisT.Add(6 * time.Minute)
	since := nowT.Add(-5 * time.Minute)
	interval := uint64(60)

	res, err := r.tradeService.GetCandlesChart(ctx, "BTC/DEC18", since, interval)
	if err != nil {
		return nil, err
	}

	candles := make([]msg.Candle, 0)
	for _, v := range res.Candles {
		candles = append(candles, msg.Candle{
			Volume:           v.Volume,
			High:             v.High,
			Low:              v.Low,
			Date:             v.Date,
			Open:             v.Open,
			Close:            v.Close,
			OpenBlockNumber:  v.OpenBlockNumber,
			CloseBlockNumber: v.CloseBlockNumber,
		})
	}
	return candles, err
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
	return Market{obj.Market}, nil
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

func (r *MyCandleResolver) High(ctx context.Context, obj *msg.Candle) (int, error) {
	return int(obj.High), nil
}
func (r *MyCandleResolver) Low(ctx context.Context, obj *msg.Candle) (int, error) {
	return int(obj.Low), nil
}
func (r *MyCandleResolver) Open(ctx context.Context, obj *msg.Candle) (int, error) {
	return int(obj.Open), nil
}
func (r *MyCandleResolver) Close(ctx context.Context, obj *msg.Candle) (int, error) {
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
	return Market{obj.Market}, nil
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

// BEGIN: Subscription Resolver

type MySubscriptionResolver resolverRoot

func (r *MySubscriptionResolver) TradeCandles(ctx context.Context, market string, interval int) (<-chan []msg.Candle, error) {
	events := make(chan []msg.Candle, 1)

	go func(channel chan[]msg.Candle) {
		subscribed := true
		fmt.Println("new subscriber on channel")
		select {
			case <-ctx.Done():
				subscribed = false
				fmt.Println("subscriber on channel left the party :(")
			default:
				fmt.Println("start generating candles at interval")
				for subscribed {
					currentTime := time.Now()

					fmt.Printf("market: %s interval: %d", market, interval)
					fmt.Println()

					count :=int64(interval)
					since := currentTime.Add(time.Duration(-count) * time.Second)


					fmt.Printf("%+v, %+v", since, currentTime)

					
					res, err := r.tradeService.GetCandlesChart(ctx, market, since, 1)
					if err != nil {
						fmt.Errorf("there was an error when getting candles charts: %v", err)
					}

					fmt.Printf("Candles holder: %+v", res)
					fmt.Println()
					fmt.Printf("Candles returned: %+v", res.Candles)
					fmt.Println()

					candles := make([]msg.Candle, 0)

					for _, v := range res.Candles {
						candles = append(candles, msg.Candle{
							Volume:           v.Volume,
							High:             v.High,
							Low:              v.Low,
							Date:             v.Date,
							Open:             v.Open,
							Close:            v.Close,
							OpenBlockNumber:  v.OpenBlockNumber,
							CloseBlockNumber: v.CloseBlockNumber,
						})
					}

					channel <- candles

					time.Sleep(time.Duration(count) * time.Second)
					fmt.Println("pong")
				}
			}
	}(events)
	
	return events, nil
}