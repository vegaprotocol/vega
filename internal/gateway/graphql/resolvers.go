package gql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"code.vegaprotocol.io/vega/internal/gateway"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
)

var (
	ErrNilPendingOrder = errors.New("mil pending order")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_client_mock.go -package mocks code.vegaprotocol.io/vega/internal/gateway/graphql TradingClient
type TradingClient interface {
	// unary calls - writes
	SubmitOrder(ctx context.Context, in *types.OrderSubmission, opts ...grpc.CallOption) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, in *types.OrderCancellation, opts ...grpc.CallOption) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, in *types.OrderAmendment, opts ...grpc.CallOption) (*protoapi.OrderResponse, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_data_client_mock.go -package mocks code.vegaprotocol.io/vega/internal/gateway/graphql TradingDataClient
type TradingDataClient interface {
	// orders
	OrdersByMarket(ctx context.Context, in *protoapi.OrdersByMarketRequest, opts ...grpc.CallOption) (*protoapi.OrdersByMarketResponse, error)
	OrdersByParty(ctx context.Context, in *protoapi.OrdersByPartyRequest, opts ...grpc.CallOption) (*protoapi.OrdersByPartyResponse, error)
	OrderByMarketAndId(ctx context.Context, in *protoapi.OrderByMarketAndIdRequest, opts ...grpc.CallOption) (*protoapi.OrderByMarketAndIdResponse, error)
	// markets
	MarketByID(ctx context.Context, in *protoapi.MarketByIDRequest, opts ...grpc.CallOption) (*protoapi.MarketByIDResponse, error)
	Markets(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*protoapi.MarketsResponse, error)
	MarketDepth(ctx context.Context, in *protoapi.MarketDepthRequest, opts ...grpc.CallOption) (*protoapi.MarketDepthResponse, error)
	LastTrade(ctx context.Context, in *protoapi.LastTradeRequest, opts ...grpc.CallOption) (*protoapi.LastTradeResponse, error)
	// parties
	PartyByID(ctx context.Context, in *protoapi.PartyByIDRequest, opts ...grpc.CallOption) (*protoapi.PartyByIDResponse, error)
	Parties(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*protoapi.PartiesResponse, error)
	// trades
	TradesByMarket(ctx context.Context, in *protoapi.TradesByMarketRequest, opts ...grpc.CallOption) (*protoapi.TradesByMarketResponse, error)
	TradesByParty(ctx context.Context, in *protoapi.TradesByPartyRequest, opts ...grpc.CallOption) (*protoapi.TradesByPartyResponse, error)
	TradesByOrder(ctx context.Context, in *protoapi.TradesByOrderRequest, opts ...grpc.CallOption) (*protoapi.TradesByOrderResponse, error)
	// positions
	PositionsByParty(ctx context.Context, in *protoapi.PositionsByPartyRequest, opts ...grpc.CallOption) (*protoapi.PositionsByPartyResponse, error)
	// candles
	Candles(ctx context.Context, in *protoapi.CandlesRequest, opts ...grpc.CallOption) (*protoapi.CandlesResponse, error)
	// metrics
	Statistics(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*types.Statistics, error)
	GetVegaTime(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*protoapi.VegaTimeResponse, error)
	// streams
	OrdersSubscribe(ctx context.Context, in *protoapi.OrdersSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_OrdersSubscribeClient, error)
	TradesSubscribe(ctx context.Context, in *protoapi.TradesSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_TradesSubscribeClient, error)
	CandlesSubscribe(ctx context.Context, in *protoapi.CandlesSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_CandlesSubscribeClient, error)
	MarketDepthSubscribe(ctx context.Context, in *protoapi.MarketDepthSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_MarketDepthSubscribeClient, error)
	PositionsSubscribe(ctx context.Context, in *protoapi.PositionsSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_PositionsSubscribeClient, error)
}

type resolverRoot struct {
	gateway.Config

	log               *logging.Logger
	tradingClient     TradingClient
	tradingDataClient TradingDataClient
}

func NewResolverRoot(
	log *logging.Logger,
	config gateway.Config,
	tradingClient TradingClient,
	tradingDataClient TradingDataClient,
) *resolverRoot {

	return &resolverRoot{
		log:               log,
		Config:            config,
		tradingClient:     tradingClient,
		tradingDataClient: tradingDataClient,
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
func (r *resolverRoot) Position() PositionResolver {
	return (*MyPositionResolver)(r)
}
func (r *resolverRoot) Party() PartyResolver {
	return (*MyPartyResolver)(r)
}
func (r *resolverRoot) Subscription() SubscriptionResolver {
	return (*MySubscriptionResolver)(r)
}
func (r *resolverRoot) PendingOrder() PendingOrderResolver {
	return (*MyPendingOrderResolver)(r)
}

// BEGIN: Query Resolver

type MyQueryResolver resolverRoot

func (r *MyQueryResolver) Markets(ctx context.Context, id *string) ([]Market, error) {
	if id != nil {
		mkt, err := r.Market(ctx, *id)
		if err != nil {
			return nil, err
		}
		return []Market{*mkt}, nil

	}
	res, err := r.tradingDataClient.Markets(ctx, &empty.Empty{})
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	m := make([]Market, 0, len(res.Markets))
	for _, pmarket := range res.Markets {
		market, err := MarketFromProto(pmarket)
		if err != nil {
			r.log.Error("unable to convert market from proto", logging.Error(err))
			return nil, err
		}
		m = append(m, *market)
	}

	return m, nil
}

func (r *MyQueryResolver) Market(ctx context.Context, id string) (*Market, error) {
	req := protoapi.MarketByIDRequest{Id: id}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	market, err := MarketFromProto(res.Market)
	if err != nil {
		r.log.Error("unable to convert market from proto", logging.Error(err))
		return nil, err
	}
	return market, nil
}

func (r *MyQueryResolver) Parties(ctx context.Context, name *string) ([]Party, error) {
	if name == nil {
		return nil, errors.New("all parties not implemented")
	}
	pty, err := r.Party(ctx, *name)
	if err != nil {
		return nil, err
	}
	return []Party{
		{ID: pty.ID},
	}, nil
}

func (r *MyQueryResolver) Party(ctx context.Context, name string) (*Party, error) {
	req := protoapi.PartyByIDRequest{Id: name}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return &Party{ID: res.Party.Id}, nil
}

// END: Root Resolver

// BEGIN: Market Resolver

type MyMarketResolver resolverRoot

func (r *MyMarketResolver) Orders(
	ctx context.Context, market *Market, open *bool, skip *int, first *int, last *int,
) ([]types.Order, error) {
	p := makePagination(skip, first, last)
	openOnly := open != nil && *open
	req := protoapi.OrdersByMarketRequest{
		MarketID:   market.ID,
		Open:       openOnly,
		Pagination: p,
	}
	res, err := r.tradingDataClient.OrdersByMarket(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	outorders := make([]types.Order, 0, len(res.Orders))
	for _, v := range res.Orders {
		v := v
		outorders = append(outorders, *v)
	}
	return outorders, nil
}

func (r *MyMarketResolver) Trades(ctx context.Context, market *Market,
	skip *int, first *int, last *int) ([]types.Trade, error) {
	p := makePagination(skip, first, last)
	req := protoapi.TradesByMarketRequest{
		MarketID:   market.ID,
		Pagination: p,
	}
	res, err := r.tradingDataClient.TradesByMarket(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	outtrades := make([]types.Trade, 0, len(res.Trades))
	for _, v := range res.Trades {
		v := v
		outtrades = append(outtrades, *v)
	}
	return outtrades, nil
}

func (r *MyMarketResolver) Depth(ctx context.Context, market *Market) (*types.MarketDepth, error) {

	if market == nil {
		return nil, errors.New("market missing or empty")

	}

	req := protoapi.MarketDepthRequest{Market: market.ID}
	// Look for market depth for the given market (will validate market internally)
	// Note: Market depth is also known as OrderBook depth within the matching-engine
	res, err := r.tradingDataClient.MarketDepth(ctx, &req)
	if err != nil {
		r.log.Error("trading data client", logging.Error(err))
		return nil, err
	}

	return &types.MarketDepth{
		MarketID: res.MarketID,
		Buy:      res.Buy,
		Sell:     res.Sell,
	}, nil
}

func (r *MyMarketResolver) Candles(ctx context.Context, market *Market,
	sinceRaw string, interval Interval) ([]*types.Candle, error) {
	pinterval, err := convertInterval(interval)
	if err != nil {
		r.log.Debug("interval convert error", logging.Error(err))
	}

	since, err := vegatime.Parse(sinceRaw)
	if err != nil {
		return nil, err
	}

	var mkt string
	if market != nil {
		mkt = market.ID
	}

	req := protoapi.CandlesRequest{
		Market:         mkt,
		SinceTimestamp: since.UnixNano(),
		Interval:       pinterval,
	}
	res, err := r.tradingDataClient.Candles(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Candles, nil
}

// END: Market Resolver

// BEGIN: Party Resolver

type MyPartyResolver resolverRoot

func makePagination(skip, first, last *int) *protoapi.Pagination {
	var (
		offset, limit uint64
		descending    bool
	)
	if skip != nil {
		offset = uint64(*skip)
	}
	if last != nil {
		limit = uint64(*last)
		descending = true
	} else if first != nil {
		limit = uint64(*first)
	}
	return &protoapi.Pagination{
		Skip:       offset,
		Limit:      limit,
		Descending: descending,
	}
}

func (r *MyPartyResolver) Orders(ctx context.Context, party *Party,
	open *bool, skip *int, first *int, last *int) ([]types.Order, error) {

	p := makePagination(skip, first, last)
	openOnly := open != nil && *open
	req := protoapi.OrdersByPartyRequest{
		PartyID:    party.ID,
		Open:       openOnly,
		Pagination: p,
	}
	res, err := r.tradingDataClient.OrdersByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	outorders := make([]types.Order, 0, len(res.Orders))
	for _, v := range res.Orders {
		v := v
		outorders = append(outorders, *v)
	}
	return outorders, nil
}

func (r *MyPartyResolver) Trades(ctx context.Context, party *Party,
	market *string, skip *int, first *int, last *int) ([]types.Trade, error) {

	var mkt string
	if market != nil {
		mkt = *market
	}

	p := makePagination(skip, first, last)
	req := protoapi.TradesByPartyRequest{
		PartyID:    party.ID,
		MarketID:   mkt,
		Pagination: p,
	}

	res, err := r.tradingDataClient.TradesByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	outtrades := make([]types.Trade, 0, len(res.Trades))
	for _, v := range res.Trades {
		v := v
		outtrades = append(outtrades, *v)
	}
	return outtrades, nil
}

func (r *MyPartyResolver) Positions(ctx context.Context, pty *Party) ([]types.MarketPosition, error) {
	if pty == nil {
		return nil, errors.New("nil party")
	}
	req := protoapi.PositionsByPartyRequest{PartyID: pty.ID}
	res, err := r.tradingDataClient.PositionsByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	retpos := make([]types.MarketPosition, 0, len(res.Positions))
	for _, v := range res.Positions {
		v := v
		retpos = append(retpos, *v)
	}
	return retpos, nil

}

// END: Party Resolver

// BEGIN: Market Depth Resolver

type MyMarketDepthResolver resolverRoot

func (r *MyMarketDepthResolver) Buy(ctx context.Context, obj *types.MarketDepth) ([]types.PriceLevel, error) {
	valBuyLevels := make([]types.PriceLevel, 0)
	for _, v := range obj.Buy {
		valBuyLevels = append(valBuyLevels, *v)
	}
	return valBuyLevels, nil
}
func (r *MyMarketDepthResolver) Sell(ctx context.Context, obj *types.MarketDepth) ([]types.PriceLevel, error) {
	valBuyLevels := make([]types.PriceLevel, 0)
	for _, v := range obj.Sell {
		valBuyLevels = append(valBuyLevels, *v)
	}
	return valBuyLevels, nil
}

func (r *MyMarketDepthResolver) LastTrade(ctx context.Context, md *types.MarketDepth) (*types.Trade, error) {
	if md == nil {
		return nil, errors.New("invalid market depth")
	}

	req := protoapi.LastTradeRequest{MarketID: md.MarketID}
	res, err := r.tradingDataClient.LastTrade(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return res.Trade, nil
}

func (r *MyMarketDepthResolver) Market(ctx context.Context, md *types.MarketDepth) (*Market, error) {
	if md == nil {
		return nil, errors.New("invalid market depth")
	}

	req := protoapi.MarketByIDRequest{Id: md.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return MarketFromProto(res.Market)
}

// END: Market Depth Resolver

// BEGIN: Order Resolver

type MyOrderResolver resolverRoot

func (r *MyOrderResolver) Price(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}
func (r *MyOrderResolver) Type(ctx context.Context, obj *types.Order) (OrderType, error) {
	return OrderType(obj.Type.String()), nil
}
func (r *MyOrderResolver) Side(ctx context.Context, obj *types.Order) (Side, error) {
	return Side(obj.Side.String()), nil
}
func (r *MyOrderResolver) Market(ctx context.Context, obj *types.Order) (*Market, error) {
	return &Market{
		ID: obj.MarketID,
	}, nil
}
func (r *MyOrderResolver) Size(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *MyOrderResolver) Remaining(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}
func (r *MyOrderResolver) Timestamp(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatInt(obj.CreatedAt, 10), nil
}
func (r *MyOrderResolver) Status(ctx context.Context, obj *types.Order) (OrderStatus, error) {
	return OrderStatus(obj.Status.String()), nil
}
func (r *MyOrderResolver) Datetime(ctx context.Context, obj *types.Order) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}
func (r *MyOrderResolver) CreatedAt(ctx context.Context, obj *types.Order) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}
func (r *MyOrderResolver) ExpiresAt(ctx context.Context, obj *types.Order) (*string, error) {
	if obj.ExpiresAt <= 0 {
		return nil, nil
	}
	expiresAt := vegatime.Format(vegatime.UnixNano(obj.ExpiresAt))
	return &expiresAt, nil
}
func (r *MyOrderResolver) Trades(ctx context.Context, ord *types.Order) ([]*types.Trade, error) {
	if ord == nil {
		return nil, errors.New("nil order")
	}

	req := protoapi.TradesByOrderRequest{OrderID: ord.Id}
	res, err := r.tradingDataClient.TradesByOrder(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Trades, nil
}
func (r *MyOrderResolver) Party(ctx context.Context, ord *types.Order) (*Party, error) {
	if ord == nil {
		return nil, errors.New("nil order")
	}
	return &Party{
		ID: ord.PartyID,
	}, nil
}

// END: Order Resolver

// BEGIN: Trade Resolver

type MyTradeResolver resolverRoot

func (r *MyTradeResolver) Market(ctx context.Context, obj *types.Trade) (*Market, error) {
	return &Market{ID: obj.MarketID}, nil
}
func (r *MyTradeResolver) Aggressor(ctx context.Context, obj *types.Trade) (Side, error) {
	return Side(obj.Aggressor.String()), nil
}
func (r *MyTradeResolver) Price(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}
func (r *MyTradeResolver) Size(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *MyTradeResolver) Timestamp(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatInt(obj.Timestamp, 10), nil
}
func (r *MyTradeResolver) Datetime(ctx context.Context, obj *types.Trade) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}
func (r *MyTradeResolver) CreatedAt(ctx context.Context, obj *types.Trade) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}

// END: Trade Resolver

// BEGIN: Candle Resolver

type MyCandleResolver resolverRoot

func (r *MyCandleResolver) High(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.High, 10), nil
}
func (r *MyCandleResolver) Low(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Low, 10), nil
}
func (r *MyCandleResolver) Open(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Open, 10), nil
}
func (r *MyCandleResolver) Close(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Close, 10), nil
}
func (r *MyCandleResolver) Volume(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}
func (r *MyCandleResolver) Datetime(ctx context.Context, obj *types.Candle) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}
func (r *MyCandleResolver) Timestamp(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatInt(obj.Timestamp, 10), nil
}
func (r *MyCandleResolver) Interval(ctx context.Context, obj *types.Candle) (Interval, error) {
	interval := Interval(obj.Interval.String())
	if interval.IsValid() {
		return interval, nil
	} else {
		r.log.Warn("Interval conversion from proto to gql type failed, falling back to default: I15M",
			logging.String("interval", interval.String()))
		return IntervalI15m, nil
	}
}

// END: Candle Resolver

// BEGIN: Price Level Resolver

type MyPriceLevelResolver resolverRoot

func (r *MyPriceLevelResolver) Price(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}

func (r *MyPriceLevelResolver) Volume(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

func (r *MyPriceLevelResolver) NumberOfOrders(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}

func (r *MyPriceLevelResolver) CumulativeVolume(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.CumulativeVolume, 10), nil
}

// END: Price Level Resolver

// BEGIN: Position Resolver

type MyPositionResolver resolverRoot

func (r *MyPositionResolver) RealisedVolume(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return strconv.FormatInt(obj.RealisedVolume, 10), nil
}

func (r *MyPositionResolver) RealisedProfitValue(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return r.absInt64Str(obj.RealisedPNL), nil
}

func (r *MyPositionResolver) RealisedProfitDirection(ctx context.Context, obj *types.MarketPosition) (ValueDirection, error) {
	return r.direction(obj.RealisedPNL), nil
}

func (r *MyPositionResolver) UnrealisedVolume(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return strconv.FormatInt(obj.UnrealisedVolume, 10), nil
}

func (r *MyPositionResolver) UnrealisedProfitValue(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return r.absInt64Str(obj.UnrealisedPNL), nil
}

func (r *MyPositionResolver) UnrealisedProfitDirection(ctx context.Context, obj *types.MarketPosition) (ValueDirection, error) {
	return r.direction(obj.UnrealisedPNL), nil
}

func (r *MyPositionResolver) AverageEntryPrice(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return strconv.FormatUint(obj.AverageEntryPrice, 10), nil
}

func (r *MyPositionResolver) MinimumMargin(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return strconv.FormatInt(obj.MinimumMargin, 10), nil
}

func (r *MyPositionResolver) Market(ctx context.Context, obj *types.MarketPosition) (*Market, error) {
	return &Market{ID: obj.MarketID}, nil
}

func (r *MyPositionResolver) absInt64Str(val int64) string {
	if val < 0 {
		return strconv.FormatInt(val*-1, 10)
	}
	return strconv.FormatInt(val, 10)
}

func (r *MyPositionResolver) direction(val int64) ValueDirection {
	if val < 0 {
		return ValueDirectionNegative
	}
	return ValueDirectionPositive
}

// END: Position Resolver

// BEGIN: Mutation Resolver

type MyMutationResolver resolverRoot

func (r *MyMutationResolver) OrderCreate(ctx context.Context, market string, party string, price string,
	size string, side Side, type_ OrderType, expiration *string) (*types.PendingOrder, error) {

	order := &types.OrderSubmission{}

	// We need to convert strings to uint64 (JS doesn't yet support uint64)
	p, err := safeStringUint64(price)
	if err != nil {
		return nil, err
	}
	order.Price = p
	s, err := safeStringUint64(size)
	if err != nil {
		return nil, err
	}
	order.Size = s
	if len(market) <= 0 {
		return nil, errors.New("market missing or empty")
	}
	order.MarketID = market
	if len(party) <= 0 {
		return nil, errors.New("party missing or empty")
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	order.PartyID = party
	order.Type, err = parseOrderType(&type_)
	if err != nil {
		return nil, err
	}
	order.Side, err = parseSide(&side)
	if err != nil {
		return nil, err
	}

	// GTT must have an expiration value
	if order.Type == types.Order_GTT && expiration != nil {
		expiresAt, err := vegatime.Parse(*expiration)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("cannot parse expiration time: %s - invalid format sent to create order (example: 2018-01-02T15:04:05Z)", *expiration))
		}

		// move to pure timestamps or convert an RFC format shortly
		order.ExpiresAt = expiresAt.UnixNano()
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	pendingOrder, err := r.tradingClient.SubmitOrder(ctx, order)
	if err != nil {
		r.log.Error("Failed to create order using rpc client in graphQL resolver", logging.Error(err))
		return nil, err
	}

	return pendingOrder, nil

}

func (r *MyMutationResolver) OrderCancel(ctx context.Context, id string, market string, party string) (*types.PendingOrder, error) {
	order := &types.OrderCancellation{}

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	if len(market) <= 0 {
		return nil, errors.New("market missing or empty")
	}
	order.MarketID = market
	if len(id) == 0 {
		return nil, errors.New("id missing or empty")
	}
	order.Id = id
	if len(party) == 0 {
		return nil, errors.New("party missing or empty")
	}

	order.PartyID = party

	// Pass the cancellation over for consensus (service layer will use RPC client internally and handle errors etc)
	pendingOrder, err := r.tradingClient.CancelOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	return pendingOrder, nil

}

// END: Mutation Resolver

// BEGIN: Subscription Resolver

type MySubscriptionResolver resolverRoot

func (r *MySubscriptionResolver) Orders(ctx context.Context, market *string, party *string) (<-chan []types.Order, error) {
	var (
		mkt, pty string
	)
	if market != nil {
		mkt = *market
	}
	if party != nil {
		pty = *party
	}

	req := &api.OrdersSubscribeRequest{
		MarketID: mkt,
		PartyID:  pty,
	}
	stream, err := r.tradingDataClient.OrdersSubscribe(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []types.Order)
	go func() {
		for {
			o, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("orders: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("orders: stream closed", logging.Error(err))
				break
			}
			out := make([]types.Order, 0, len(o.Orders))
			for _, v := range o.Orders {
				out = append(out, *v)
			}
			c <- out
		}
	}()

	return c, nil
}

func (r *MySubscriptionResolver) Trades(ctx context.Context, market *string, party *string) (<-chan []types.Trade, error) {
	var (
		mkt, pty string
	)
	if market != nil {
		mkt = *market
	}
	if party != nil {
		pty = *party
	}

	req := &api.TradesSubscribeRequest{
		MarketID: mkt,
		PartyID:  pty,
	}
	stream, err := r.tradingDataClient.TradesSubscribe(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan []types.Trade)
	go func() {
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("orders: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("orders: stream closed", logging.Error(err))
				break
			}
			out := make([]types.Trade, 0, len(t.Trades))
			for _, v := range t.Trades {
				out = append(out, *v)
			}

			c <- out
		}
	}()

	return c, nil
}

func (r *MySubscriptionResolver) Positions(ctx context.Context, party string) (<-chan *types.MarketPosition, error) {
	req := &api.PositionsSubscribeRequest{
		PartyID: party,
	}
	stream, err := r.tradingDataClient.PositionsSubscribe(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan *types.MarketPosition)
	go func() {
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("orders: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("orders: stream closed", logging.Error(err))
				break
			}
			c <- t
		}
	}()

	return c, nil
}

func (r *MySubscriptionResolver) MarketDepth(ctx context.Context, market string) (<-chan *types.MarketDepth, error) {
	req := &api.MarketDepthSubscribeRequest{
		MarketID: market,
	}
	stream, err := r.tradingDataClient.MarketDepthSubscribe(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan *types.MarketDepth)
	go func() {
		for {
			md, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("orders: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("orders: stream closed", logging.Error(err))
				break
			}
			c <- md
		}
	}()

	return c, nil
}

func (r *MySubscriptionResolver) Candles(ctx context.Context, market string, interval Interval) (<-chan *types.Candle, error) {

	pinterval, err := convertInterval(interval)
	if err != nil {
		r.log.Debug("invalid interval for candles subscriptions", logging.Error(err))
	}

	req := &api.CandlesSubscribeRequest{
		MarketID: market,
		Interval: pinterval,
	}
	stream, err := r.tradingDataClient.CandlesSubscribe(ctx, req)
	if err != nil {
		return nil, err
	}

	c := make(chan *types.Candle)
	go func() {
		for {
			cdl, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("orders: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("orders: stream closed", logging.Error(err))
				break
			}
			c <- cdl
		}
	}()

	return c, nil

}

type MyPendingOrderResolver resolverRoot

func (r *MyPendingOrderResolver) Price(ctx context.Context, obj *proto.PendingOrder) (*string, error) {
	if obj != nil {
		str := fmt.Sprintf("%v", obj.Price)
		return &str, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *MyPendingOrderResolver) Type(ctx context.Context, obj *proto.PendingOrder) (*OrderType, error) {
	if obj != nil {
		ot := OrderType(obj.Type.String())
		return &ot, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *MyPendingOrderResolver) Side(ctx context.Context, obj *proto.PendingOrder) (*Side, error) {
	if obj != nil {
		s := Side(obj.Side.String())
		return &s, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *MyPendingOrderResolver) Market(ctx context.Context, pord *proto.PendingOrder) (*Market, error) {
	if pord == nil {
		return nil, nil
	}

	req := protoapi.MarketByIDRequest{Id: pord.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return MarketFromProto(res.Market)
}

func (r *MyPendingOrderResolver) Party(ctx context.Context, pord *proto.PendingOrder) (*Party, error) {
	if pord == nil {
		return nil, nil
	}
	return &Party{ID: pord.PartyID}, nil
}

func (r *MyPendingOrderResolver) Size(ctx context.Context, obj *proto.PendingOrder) (*string, error) {
	if obj != nil {
		str := fmt.Sprintf("%v", obj.Size)
		return &str, nil
	}
	return nil, ErrNilPendingOrder
}
func (r *MyPendingOrderResolver) Status(ctx context.Context, obj *proto.PendingOrder) (*OrderStatus, error) {
	if obj != nil {
		os := OrderStatus(obj.Status.String())
		return &os, nil
	}
	return nil, ErrNilPendingOrder
}

// END: Subscription Resolver
