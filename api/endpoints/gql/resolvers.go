package gql

import (
	"context"
	"vega/api"
	"vega/msg"
	"errors"
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

func (r *resolverRoot) Query() QueryResolver {
	return (*MyQueryResolver)(r)
}
func (r *resolverRoot) Candle() CandleResolver {
	return (*MyCandleResolver)(r)
}
func (r *resolverRoot) MarketDepth() MarketDepthResolver {
	return (*MyMarketDepthResolver)(r)
}
func (r *resolverRoot) PriceLevel() PriceLevelResolver {
	return (*MyPriceLevelResolver)(r)
}
func (r *resolverRoot) Market() MarketResolver {
	return (*MyMarketResolver)(r)
}
func (r *resolverRoot) Order() OrderResolver {
	return (*MyOrderResolver)(r)
}
func (r *resolverRoot) Trade() TradeResolver {
	return (*MyTradeResolver)(r)
}
func (r *resolverRoot) Vega() VegaResolver {
	return (*MyVegaResolver)(r)
}

// BEGIN: Query Resolver

type MyQueryResolver resolverRoot

func (r *MyQueryResolver) Vega(ctx context.Context) (Vega, error) {
	var vega = Vega{}
	return vega, nil
}

// END: Query Resolver

type MyVegaResolver resolverRoot

func (r *MyVegaResolver) Markets(ctx context.Context, obj *Vega, name *string) ([]Market, error) {
	if name == nil {
		return nil, errors.New("All markets for VEGA platform not implemented")
	}

	// Look for orders for market (will validate market internally)
	orders, err := r.orderService.GetByMarket(ctx, *name, 1000)
	if err != nil {
		return nil, err
	}

	trades, err := r.tradeService.GetByMarket(ctx, *name, 1000)
	if err != nil {
		return nil, err
	}

	valOrders := make([]msg.Order, 0)
	for _, v := range orders {
		valOrders = append(valOrders, *v)
	}

	valTrades := make([]msg.Trade, 0)
	for _, v := range trades {
		valTrades = append(valTrades, *v)
	}

	var markets = make([]Market, 0)
	var market = Market{
		Name: *name,
		Orders: valOrders,
		Trades: valTrades,
	}
	markets = append(markets, market)
	
	return markets, nil
}

func (r *MyVegaResolver) Parties(ctx context.Context, obj *Vega, name *string) ([]Party, error) {
	if name == nil {
		return nil, errors.New("All parties for VEGA platform not implemented")
	}

	// Look for orders for party (will validate market internally)
	orders, err := r.orderService.GetByParty(ctx, *name, 1000)
	if err != nil {
		return nil, err
	}

	valOrders := make([]msg.Order, 0)
	for _, v := range orders {
		valOrders = append(valOrders, *v)
	}
	var parties = make([]Party, 0)
	var party = Party{
		Name: *name,
		Orders: valOrders,
	}
	parties = append(parties, party)
	
	return parties, nil
}

// BEGIN: Market Resolver

type MyMarketResolver resolverRoot

func (r *MyMarketResolver) Depth(ctx context.Context, obj *Market) (msg.MarketDepth, error) {

	// Look for market depth for the given market (will validate market internally)
	// FYI: Market depth is also known as OrderBook depth within the matching-engine
	depth, err := r.orderService.GetMarketDepth(ctx, obj.Name)
	if err != nil {
		return msg.MarketDepth{}, err
	}

	return *depth, nil
}

// END: Market Resolver

// BEGIN: Market Depth Resolver

type MyMarketDepthResolver resolverRoot

func (r *MyMarketDepthResolver) Buy(ctx context.Context, obj *msg.MarketDepth) ([]msg.PriceLevel, error) {
	valBuyLevels := make([]msg.PriceLevel, 0)
	for _, v := range obj.Buy {
		valBuyLevels = append(valBuyLevels, *v)
	}
	return valBuyLevels, nil
}
func (r *MyMarketDepthResolver) Sell(ctx context.Context, obj *msg.MarketDepth) ([]msg.PriceLevel, error) {
	valBuyLevels := make([]msg.PriceLevel, 0)
	for _, v := range obj.Sell {
		valBuyLevels = append(valBuyLevels, *v)
	}
	return valBuyLevels, nil
}

// END: Market Depth Resolver

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
	return Market {
		Name: obj.Market,
	}, nil
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
func (r *MyOrderResolver) Status(ctx context.Context, obj *msg.Order) (OrderStatus, error) {
	return OrderStatus(obj.Status.String()), nil
}

// END: Order Resolver

// BEGIN: Trade Resolver

type MyTradeResolver resolverRoot

func (r *MyTradeResolver) Market(ctx context.Context, obj *msg.Trade) (Market, error) {
	return Market{Name: obj.Market}, nil
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

// BEGIN: Price Level Resolver

type MyPriceLevelResolver resolverRoot

func (r *MyPriceLevelResolver) Price(ctx context.Context, obj *msg.PriceLevel) (int, error) {
	return int(obj.Price), nil
}

func (r *MyPriceLevelResolver) Volume(ctx context.Context, obj *msg.PriceLevel) (int, error) {
	return int(obj.Volume), nil
}

func (r *MyPriceLevelResolver) NumberOfOrders(ctx context.Context, obj *msg.PriceLevel) (int, error) {
	return int(obj.Price), nil
}

func (r *MyPriceLevelResolver) CumulativeVolume(ctx context.Context, obj *msg.PriceLevel) (int, error) {
	return int(obj.CumulativeVolume), nil
}

// END: Price Level Resolver