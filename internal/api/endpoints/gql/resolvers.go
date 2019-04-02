package gql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrChainNotConnected = errors.New("chain not connected")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/gql_order_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/gql OrderService
type OrderService interface {
	CreateOrder(ctx context.Context, order *types.OrderSubmission) (success bool, orderReference string, err error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) (success bool, err error)
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	ObserveOrders(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Order, ref uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/gql_trade_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/gql TradeService
type TradeService interface {
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (trades []*types.Trade, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, market *string) (trades []*types.Trade, err error)
	GetByOrderId(ctx context.Context, orderId string) (trades []*types.Trade, err error)
	GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error)
	ObservePositions(ctx context.Context, retries int, party string) (positions <-chan *types.MarketPosition, ref uint64)
	ObserveTrades(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Trade, ref uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/gql_candle_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/gql CandleService
type CandleService interface {
	ObserveCandles(ctx context.Context, retries int, market *string, interval *types.Interval) (candleCh <-chan *types.Candle, ref uint64)
	GetCandles(ctx context.Context, market string, since time.Time, interval types.Interval) (candles []*types.Candle, err error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/gql_market_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/gql MarketService
type MarketService interface {
	GetAll(ctx context.Context) ([]*types.Market, error)
	GetByID(ctx context.Context, name string) (*types.Market, error)
	GetDepth(ctx context.Context, market string) (marketDepth *types.MarketDepth, err error)
	ObserveDepth(ctx context.Context, retries int, market string) (depth <-chan *types.MarketDepth, ref uint64)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/gql_party_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/gql PartyService
type PartyService interface {
	GetAll(ctx context.Context) ([]*types.Party, error)
	GetByID(ctx context.Context, name string) (*types.Party, error)
}

type resolverRoot struct {
	*api.Config
	orderService  OrderService
	tradeService  TradeService
	candleService CandleService
	marketService MarketService
	partyService  PartyService
	statusChecker *monitoring.Status
}

func NewResolverRoot(
	config *api.Config,
	orderService OrderService,
	tradeService TradeService,
	candleService CandleService,
	marketService MarketService,
	partyService PartyService,
	statusChecker *monitoring.Status,
) *resolverRoot {

	return &resolverRoot{
		Config:        config,
		orderService:  orderService,
		tradeService:  tradeService,
		candleService: candleService,
		marketService: marketService,
		partyService:  partyService,
		statusChecker: statusChecker,
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

// BEGIN: Query Resolver

type MyQueryResolver resolverRoot

func (r *MyQueryResolver) Markets(ctx context.Context, id *string) ([]Market, error) {
	if id != nil {
		mkt, err := validateMarket(ctx, id, r.marketService)
		if err != nil {
			return nil, err
		}
		market, err := MarketFromProto(mkt)
		if err != nil {
			r.GetLogger().Error("unable to convert market from proto", logging.Error(err))
			return nil, err
		}
		return []Market{
			*market,
		}, nil
	}
	pmkts, err := r.marketService.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	m := make([]Market, 0, len(pmkts))
	for _, pmarket := range pmkts {
		market, err := MarketFromProto(pmarket)
		if err != nil {
			r.GetLogger().Error("unable to convert market from proto", logging.Error(err))
			return nil, err
		}
		m = append(m, *market)
	}

	return m, nil
}

func (r *MyQueryResolver) Market(ctx context.Context, id string) (*Market, error) {
	mkt, err := validateMarket(ctx, &id, r.marketService)
	if err != nil {
		return nil, err
	}
	market, err := MarketFromProto(mkt)
	if err != nil {
		r.GetLogger().Error("unable to convert market from proto", logging.Error(err))
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
		{Name: pty.Name},
	}, nil
}

func (r *MyQueryResolver) Party(ctx context.Context, name string) (*Party, error) {
	pty, err := validateParty(ctx, &name, r.partyService)
	if err != nil {
		return nil, err
	}
	return &Party{Name: pty.Name}, nil
}

// END: Root Resolver

// BEGIN: Market Resolver

type MyMarketResolver resolverRoot

func (r *MyMarketResolver) Orders(ctx context.Context, market *Market,
	open *bool, skip *int, first *int, last *int) ([]types.Order, error) {
	_, err := validateMarket(ctx, &market.ID, r.marketService)
	if err != nil {
		return nil, err
	}
	var (
		offset, limit uint64
		descending    bool
	)
	if skip != nil {
		offset = uint64(*skip)
	}
	if last != nil {
		descending = true
		limit = uint64(*last)
	} else if first != nil {
		limit = uint64(*first)
	}
	o, err := r.orderService.GetByMarket(ctx, market.ID, limit, offset, descending, open)
	if err != nil {
		return nil, err
	}
	valOrders := make([]types.Order, 0, len(o))
	for _, v := range o {
		valOrders = append(valOrders, *v)
	}
	return valOrders, nil
}

func (r *MyMarketResolver) Trades(ctx context.Context, market *Market,
	skip *int, first *int, last *int) ([]types.Trade, error) {
	_, err := validateMarket(ctx, &market.ID, r.marketService)
	if err != nil {
		return nil, err
	}
	var (
		offset, limit uint64
		descending    bool
	)
	if skip != nil {
		offset = uint64(*skip)
	}
	if last != nil {
		descending = true
		limit = uint64(*last)
	} else if first != nil {
		limit = uint64(*first)
	}
	t, err := r.tradeService.GetByMarket(ctx, market.ID, offset, limit, descending)
	if err != nil {
		return nil, err
	}
	valTrades := make([]types.Trade, 0, len(t))
	for _, v := range t {
		valTrades = append(valTrades, *v)
	}
	return valTrades, nil
}

func (r *MyMarketResolver) Depth(ctx context.Context, market *Market) (*types.MarketDepth, error) {
	if market == nil {
		return nil, errors.New("market missing or empty")

	}
	_, err := validateMarket(ctx, &market.ID, r.marketService)
	if err != nil {
		return nil, err
	}
	// Look for market depth for the given market (will validate market internally)
	// Note: Market depth is also known as OrderBook depth within the matching-engine
	depth, err := r.marketService.GetDepth(ctx, market.ID)
	if err != nil {
		return nil, err
	}

	return depth, nil
}

func (r *MyMarketResolver) Candles(ctx context.Context, market *Market,
	sinceTimestampRaw string, interval Interval) ([]*types.Candle, error) {

	// Validate interval, map to protobuf enum
	var pbInterval types.Interval
	switch interval {
	case IntervalI15M:
		pbInterval = types.Interval_I15M
	case IntervalI1D:
		pbInterval = types.Interval_I1D
	case IntervalI1H:
		pbInterval = types.Interval_I1H
	case IntervalI1M:
		pbInterval = types.Interval_I1M
	case IntervalI5M:
		pbInterval = types.Interval_I5M
	case IntervalI6H:
		pbInterval = types.Interval_I6H
	default:
		logger := *r.GetLogger()
		logger.Warn("Invalid interval when subscribing to candles, falling back to default: I15M",
			logging.String("interval", interval.String()))
		pbInterval = types.Interval_I15M
	}

	// Convert javascript string representation of int epoch+nano timestamp
	sinceTimestamp, err := strconv.ParseInt(sinceTimestampRaw, 10, 64)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error converting %s into a valid timestamp", sinceTimestampRaw))
	}
	if len(sinceTimestampRaw) < 19 {
		return nil, errors.New("timestamp should be in epoch+nanoseconds format, eg. 1545158175835902621")
	}

	// Retrieve candles from store/service
	c, err := r.candleService.GetCandles(ctx, market.ID, vegatime.UnixNano(sinceTimestamp), pbInterval)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// END: Market Resolver

// BEGIN: Party Resolver

type MyPartyResolver resolverRoot

func (r *MyPartyResolver) Orders(ctx context.Context, party *Party,
	open *bool, skip *int, first *int, last *int) ([]types.Order, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

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
	o, err := r.orderService.GetByParty(ctx, party.Name, offset, limit, descending, open)
	if err != nil {
		return nil, err
	}
	valOrders := make([]types.Order, 0)
	for _, v := range o {
		valOrders = append(valOrders, *v)
	}
	return valOrders, nil
}

func (r *MyPartyResolver) Trades(ctx context.Context, party *Party,
	market *string, skip *int, first *int, last *int) ([]types.Trade, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

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
	t, err := r.tradeService.GetByParty(ctx, party.Name, offset, limit, descending, market)
	if err != nil {
		return nil, err
	}
	valTrades := make([]types.Trade, 0, len(t))
	for _, v := range t {
		valTrades = append(valTrades, *v)
	}
	return valTrades, nil
}

func (r *MyPartyResolver) Positions(ctx context.Context, obj *Party) ([]types.MarketPosition, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	positions, err := r.tradeService.GetPositionsByParty(ctx, obj.Name)
	if err != nil {
		return nil, err
	}
	var valPositions = make([]types.MarketPosition, 0)
	for _, v := range positions {
		valPositions = append(valPositions, *v)
	}
	return valPositions, nil
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

func (r *MyMarketDepthResolver) LastTrade(ctx context.Context, obj *types.MarketDepth) (*types.Trade, error) {
	_, err := validateMarket(ctx, &obj.Name, r.marketService)
	if err != nil {
		return nil, err
	}
	// skip 0, descending, get one trade
	t, err := r.tradeService.GetByMarket(ctx, obj.Name, 0, 1, true)
	if err != nil {
		return nil, err
	}
	if t != nil && len(t) > 0 && t[0] != nil {
		return t[0], nil
	}
	// No trades found on the market yet (and no errors)
	// this can happen at the beginning of a new market
	return nil, nil
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
		ID: obj.Market,
	}, nil
}
func (r *MyOrderResolver) Size(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *MyOrderResolver) Remaining(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}
func (r *MyOrderResolver) Timestamp(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatInt(obj.Timestamp, 10), nil
}
func (r *MyOrderResolver) Status(ctx context.Context, obj *types.Order) (OrderStatus, error) {
	return OrderStatus(obj.Status.String()), nil
}
func (r *MyOrderResolver) Datetime(ctx context.Context, obj *types.Order) (string, error) {
	return vegatime.UnixNano(obj.Timestamp).Format(time.RFC3339Nano), nil
}
func (r *MyOrderResolver) Trades(ctx context.Context, obj *types.Order) ([]*types.Trade, error) {
	relatedTrades, err := r.tradeService.GetByOrderId(ctx, obj.Id)
	if err != nil {
		return nil, err
	}
	return relatedTrades, nil
}

// END: Order Resolver

// BEGIN: Trade Resolver

type MyTradeResolver resolverRoot

func (r *MyTradeResolver) Market(ctx context.Context, obj *types.Trade) (*Market, error) {
	return &Market{ID: obj.Market}, nil
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
	return vegatime.UnixNano(obj.Timestamp).Format(time.RFC3339Nano), nil
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
	return vegatime.UnixNano(obj.Timestamp).Format(time.RFC3339Nano), nil
}
func (r *MyCandleResolver) Timestamp(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatInt(obj.Timestamp, 10), nil
}
func (r *MyCandleResolver) Interval(ctx context.Context, obj *types.Candle) (Interval, error) {
	interval := Interval(obj.Interval.String())
	if interval.IsValid() {
		return interval, nil
	} else {
		logger := *r.GetLogger()
		logger.Warn("Interval conversion from proto to gql type failed, falling back to default: I15M",
			logging.String("interval", interval.String()))
		return IntervalI15M, nil
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
	return &Market{ID: obj.Market}, nil
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
	size string, side Side, type_ OrderType, expiration *string) (*PreConsensus, error) {
	order := &types.OrderSubmission{}
	res := PreConsensus{}
	if r.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return &res, ErrChainNotConnected
	}

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
	_, err = validateMarket(ctx, &market, r.marketService)
	if err != nil {
		return nil, err
	}
	order.MarketId = market
	if len(party) == 0 {
		return nil, errors.New("party missing or empty")
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	order.Party = party
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

		// Validate RFC3339 with no milli or nanosecond (@matt has chosen this strategy, good enough until unix epoch timestamp)
		layout := "2006-01-02T15:04:05Z"
		_, err := time.Parse(layout, *expiration)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("cannot parse expiration time: %s - invalid format sent to create order (example: 2018-01-02T15:04:05Z)", *expiration))
		}

		// move to pure timestamps or convert an RFC format shortly
		order.ExpirationDatetime = *expiration
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	accepted, reference, err := r.orderService.CreateOrder(ctx, order)
	if err != nil {
		logger := *r.GetLogger()
		logger.Error("Failed to create order using rpc client in graphQL resolver", logging.Error(err))
		return nil, err
	}

	res.Accepted = accepted
	res.Reference = reference
	return &res, nil
}

func (r *MyMutationResolver) OrderCancel(ctx context.Context, id string, market string, party string) (*PreConsensus, error) {
	order := &types.OrderCancellation{}
	res := PreConsensus{}

	if r.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return &res, ErrChainNotConnected
	}

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	_, err := validateMarket(ctx, &market, r.marketService)
	if err != nil {
		return nil, err
	}
	order.MarketId = market
	if len(id) == 0 {
		return nil, errors.New("id missing or empty")
	}
	order.Id = id
	if len(party) == 0 {
		return nil, errors.New("party missing or empty")
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	order.Party = party

	// Pass the cancellation over for consensus (service layer will use RPC client internally and handle errors etc)
	accepted, err := r.orderService.CancelOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	res.Accepted = accepted
	return &res, nil
}

// END: Mutation Resolver

// BEGIN: Subscription Resolver

type MySubscriptionResolver resolverRoot

func (r *MySubscriptionResolver) Orders(ctx context.Context, market *string, party *string) (<-chan []types.Order, error) {
	_, err := validateMarket(ctx, market, r.marketService)
	if err != nil {
		return nil, err
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	c, ref := r.orderService.ObserveOrders(ctx, r.Config.GraphQLSubscriptionRetries, market, party)
	logger := *r.GetLogger()
	logger.Debug("Orders: new subscriber", logging.Uint64("ref", ref))
	return c, nil
}

func (r *MySubscriptionResolver) Trades(ctx context.Context, market *string, party *string) (<-chan []types.Trade, error) {
	_, err := validateMarket(ctx, market, r.marketService)
	if err != nil {
		return nil, err
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	c, ref := r.tradeService.ObserveTrades(ctx, r.Config.GraphQLSubscriptionRetries, market, party)
	logger := *r.GetLogger()
	logger.Debug("Trades: new subscriber", logging.Uint64("ref", ref))
	return c, nil
}

func (r *MySubscriptionResolver) Positions(ctx context.Context, party string) (<-chan *types.MarketPosition, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	c, ref := r.tradeService.ObservePositions(ctx, r.Config.GraphQLSubscriptionRetries, party)
	logger := *r.GetLogger()
	logger.Debug("Positions: new subscriber", logging.Uint64("ref", ref))
	return c, nil
}

func (r *MySubscriptionResolver) MarketDepth(ctx context.Context, market string) (<-chan *types.MarketDepth, error) {
	_, err := validateMarket(ctx, &market, r.marketService)
	if err != nil {
		return nil, err
	}
	c, ref := r.marketService.ObserveDepth(ctx, r.Config.GraphQLSubscriptionRetries, market)
	logger := *r.GetLogger()
	logger.Debug("Market Depth: new subscriber", logging.Uint64("ref", ref))
	return c, nil
}

func (r *MySubscriptionResolver) Candles(ctx context.Context, market string, interval Interval) (<-chan *types.Candle, error) {
	_, err := validateMarket(ctx, &market, r.marketService)
	if err != nil {
		return nil, err
	}

	logger := *r.GetLogger()

	var pbInterval types.Interval
	switch interval {
	case IntervalI15M:
		pbInterval = types.Interval_I15M
	case IntervalI1D:
		pbInterval = types.Interval_I1D
	case IntervalI1H:
		pbInterval = types.Interval_I1H
	case IntervalI1M:
		pbInterval = types.Interval_I1M
	case IntervalI5M:
		pbInterval = types.Interval_I5M
	case IntervalI6H:
		pbInterval = types.Interval_I6H
	default:
		logger.Warn("Invalid interval when subscribing to candles in gql, falling back to default: I15M",
			logging.String("interval", interval.String()))
		pbInterval = types.Interval_I15M
	}

	// Observe new candles for interval
	// --------------------------------

	c, ref := r.candleService.ObserveCandles(ctx, r.Config.GraphQLSubscriptionRetries, &market, &pbInterval)

	logger.Debug("Candles: New subscriber",
		logging.String("interval", pbInterval.String()),
		logging.String("market", market),
		logging.Uint64("ref", ref))

	return c, nil
}

func validateMarket(ctx context.Context, marketId *string, marketService MarketService) (*types.Market, error) {
	var mkt *types.Market
	var err error
	if marketId != nil {
		if len(*marketId) == 0 {
			return nil, errors.New("market must not be empty")
		}
		mkt, err = marketService.GetByID(ctx, *marketId)
		if err != nil {
			return nil, err
		}
	}
	return mkt, nil
}

func validateParty(ctx context.Context, partyId *string, partyService PartyService) (*types.Party, error) {
	var pty *types.Party
	var err error
	if partyId != nil {
		if len(*partyId) == 0 {
			return nil, errors.New("party must not be empty")
		}
		pty, err = partyService.GetByID(ctx, *partyId)
		if err != nil {
			return nil, err
		}
	}
	return pty, err
}

// END: Subscription Resolver
