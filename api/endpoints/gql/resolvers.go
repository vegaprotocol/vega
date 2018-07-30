package gql

import (
	"context"
	"vega/api"
	"vega/msg"
	"errors"
	"strconv"
	"fmt"
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
func (r *resolverRoot) Mutation() MutationResolver {
	return (*MyMutationResolver)(r)
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
func (r *resolverRoot) Position() PositionResolver {
	return (*MyPositionResolver)(r)
}
func (r *resolverRoot) Party() PartyResolver {
	return (*MyPartyResolver)(r)
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
		return nil, errors.New("all markets for VEGA platform not implemented")
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
		return nil, errors.New("all parties for VEGA platform not implemented")
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


// BEGIN: Party Resolver

type MyPartyResolver resolverRoot

func (r *MyPartyResolver) Positions(ctx context.Context, obj *Party) ([]msg.MarketPosition, error) {
	positions, err := r.tradeService.GetPositionsByParty(ctx, obj.Name)
	if err != nil {
		return nil, err
	}
	var valPositions = make([]msg.MarketPosition, 0)
	for _, v := range positions {
		valPositions = append(valPositions, *v)
	}
	return valPositions, nil
}

// END: Party Resolver


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

func (r *MyOrderResolver) Price(ctx context.Context, obj *msg.Order) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
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
func (r *MyOrderResolver) Size(ctx context.Context, obj *msg.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *MyOrderResolver) Remaining(ctx context.Context, obj *msg.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}
func (r *MyOrderResolver) Timestamp(ctx context.Context, obj *msg.Order) (string, error) {
	return strconv.FormatUint(obj.Timestamp, 10), nil
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
func (r *MyTradeResolver) Price(ctx context.Context, obj *msg.Trade) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}
func (r *MyTradeResolver) Size(ctx context.Context, obj *msg.Trade) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *MyTradeResolver) Timestamp(ctx context.Context, obj *msg.Trade) (string, error) {
	return strconv.FormatUint(obj.Timestamp, 10), nil
}

// END: Trade Resolver

// BEGIN: Candle Resolver

type MyCandleResolver resolverRoot

func (r *MyCandleResolver) High(ctx context.Context, obj *msg.Candle) (string, error) {
	return strconv.FormatUint(obj.High, 10), nil
}
func (r *MyCandleResolver) Low(ctx context.Context, obj *msg.Candle) (string, error) {
	return strconv.FormatUint(obj.Low, 10), nil
}
func (r *MyCandleResolver) Open(ctx context.Context, obj *msg.Candle) (string, error) {
	return strconv.FormatUint(obj.Open, 10), nil
}
func (r *MyCandleResolver) Close(ctx context.Context, obj *msg.Candle) (string, error) {
	return strconv.FormatUint(obj.Close, 10), nil
}
func (r *MyCandleResolver) Volume(ctx context.Context, obj *msg.Candle) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}
func (r *MyCandleResolver) OpenBlockNumber(ctx context.Context, obj *msg.Candle) (string, error) {
	return strconv.FormatUint(obj.OpenBlockNumber, 10), nil
}
func (r *MyCandleResolver) CloseBlockNumber(ctx context.Context, obj *msg.Candle) (string, error) {
	return strconv.FormatUint(obj.CloseBlockNumber, 10), nil
}

// END: Candle Resolver

// BEGIN: Price Level Resolver

type MyPriceLevelResolver resolverRoot

func (r *MyPriceLevelResolver) Price(ctx context.Context, obj *msg.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}

func (r *MyPriceLevelResolver) Volume(ctx context.Context, obj *msg.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

func (r *MyPriceLevelResolver) NumberOfOrders(ctx context.Context, obj *msg.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}

func (r *MyPriceLevelResolver) CumulativeVolume(ctx context.Context, obj *msg.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.CumulativeVolume, 10), nil
}

// END: Price Level Resolver


// BEGIN: Position Resolver

type MyPositionResolver resolverRoot

func (r *MyPositionResolver) RealisedVolume(ctx context.Context, obj *msg.MarketPosition) (string, error) {
	return string(obj.RealisedVolume), nil
}

func (r *MyPositionResolver) RealisedProfitValue(ctx context.Context, obj *msg.MarketPosition) (string, error) {
	return r.absInt64Str(obj.RealisedPNL), nil
}

func (r *MyPositionResolver) RealisedProfitDirection(ctx context.Context, obj *msg.MarketPosition) (ValueDirection, error) {
	return r.direction(obj.RealisedPNL), nil
}

func (r *MyPositionResolver) UnrealisedVolume(ctx context.Context, obj *msg.MarketPosition) (string, error) {
	return string(obj.UnrealisedVolume), nil
}

func (r *MyPositionResolver) UnrealisedProfitValue(ctx context.Context, obj *msg.MarketPosition) (string, error) {
	return r.absInt64Str(obj.UnrealisedPNL), nil
}

func (r *MyPositionResolver) UnrealisedProfitDirection(ctx context.Context, obj *msg.MarketPosition) (ValueDirection, error) {
	return r.direction(obj.UnrealisedPNL), nil
}

func (r *MyPositionResolver) absInt64Str(val int64) (string) {
	if val < 0 {
		return strconv.FormatInt(val * -1, 10)
	}
	return strconv.FormatInt(val, 10)
}

func (r *MyPositionResolver) direction(val int64) (ValueDirection) {
	if val < 0 {
		return ValueDirectionNegative
	}
	return ValueDirectionPositive
}

// END: Position Resolver


// BEGIN: Mutation Resolver

type MyMutationResolver resolverRoot

func (r *MyMutationResolver) OrderCreate(ctx context.Context, market string, party string, price string,
	size string, side Side, type_ OrderType) (PreConsensus, error) {
	order := &msg.Order{}
	res := PreConsensus{}

	// We need to convert strings to uint64 (JS doesn't yet support uint64)
	p, err := SafeStringUint64(price)
	if err != nil {
		return res, err
	}
	order.Price = p
	s, err := SafeStringUint64(size)
	if err != nil {
		return res, err
	}
	order.Size = s
	if len(market) == 0 {
		return res, errors.New("market missing or empty")
	}
	order.Market = market
	if len(party) == 0 {
		return res, errors.New("party missing or empty")
	}
	order.Party = party
	switch type_ {
		case OrderTypeGtc:
			order.Type = msg.Order_GTC
		case OrderTypeGtt:
			order.Type = msg.Order_GTT
		case OrderTypeEne:
			order.Type = msg.Order_ENE
		case OrderTypeFok:
			order.Type = msg.Order_FOK
		default:
			return res, errors.New(fmt.Sprintf("unknown type: %s", type_.String()))
	}
	switch side {
		case SideBuy:
			order.Side = msg.Side_Buy
		case SideSell:
			order.Side = msg.Side_Sell
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	accepted, err := r.orderService.CreateOrder(ctx, order)
	if err != nil {
		return res, err
	}

	res.Accepted = accepted
	return res, nil
}

func (r *MyMutationResolver) OrderCancel(ctx context.Context, id string, market string, party string) (PreConsensus, error) {
	order := &msg.Order{}
	res := PreConsensus{}

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	if len(market) == 0 {
		return res, errors.New("market missing or empty")
	}
	order.Market = market
	if len(id) == 0 {
		return res, errors.New("id missing or empty")
	}
	order.Id = id
	if len(party) == 0 {
		return res, errors.New("party missing or empty")
	}
	order.Party = party

	// Pass the cancellation over for consensus (service layer will use RPC client internally and handle errors etc)
	accepted, err := r.orderService.CancelOrder(ctx, order)
	if err != nil {
		return res, err
	}

	res.Accepted = accepted
	return res, nil
}

// END: Mutation Resolver

