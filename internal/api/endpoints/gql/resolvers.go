package gql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/internal/monitoring"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/filtering"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"
)

var (
	ErrChainNotConnected = errors.New("chain not connected")
)

type resolverRoot struct {
	*api.Config
	orderService  orders.Service
	tradeService  trades.Service
	timeService   vegatime.Service
	candleService candles.Service
	marketService markets.Service
	partyService  parties.Service
	statusChecker *monitoring.Status
}

func NewResolverRoot(
	config *api.Config,
	orderService orders.Service,
	tradeService trades.Service,
	candleService candles.Service,
	timeService vegatime.Service,
	marketService markets.Service,
	partyService parties.Service,
	statusChecker *monitoring.Status,
) *resolverRoot {

	return &resolverRoot{
		Config:        config,
		timeService:   timeService,
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
func (r *resolverRoot) Vega() VegaResolver {
	return (*MyVegaResolver)(r)
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

func (r *MyQueryResolver) Vega(ctx context.Context) (*Vega, error) {
	var vega = Vega{}
	return &vega, nil
}

// END: Query Resolver

// BEGIN: Root Resolver

type MyVegaResolver resolverRoot

func (r *MyVegaResolver) Markets(ctx context.Context, obj *Vega, name *string) ([]Market, error) {
	if name == nil {
		return nil, errors.New("all markets on VEGA query not implemented")
	}
	err := validateMarket(ctx, name, r.marketService)
	if err != nil {
		return nil, err
	}
	var m = make([]Market, 0)
	var market = Market{
		Name: *name,
	}
	m = append(m, market)

	return m, nil
}

func (r *MyVegaResolver) Market(ctx context.Context, obj *Vega, name string) (*Market, error) {
	err := validateMarket(ctx, &name, r.marketService)
	if err != nil {
		return nil, err
	}
	var market = Market{
		Name: name,
	}
	return &market, nil
}

func (r *MyVegaResolver) Parties(ctx context.Context, obj *Vega, name *string) ([]Party, error) {
	if name == nil {
		return nil, errors.New("all parties on VEGA query not implemented")
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)
	var p = make([]Party, 0)
	var party = Party{
		Name: *name,
	}
	p = append(p, party)

	return p, nil
}

func (r *MyVegaResolver) Party(ctx context.Context, obj *Vega, name string) (*Party, error) {
	var party = Party{
		Name: name,
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	return &party, nil
}

// END: Root Resolver

// BEGIN: Market Resolver

type MyMarketResolver resolverRoot

func (r *MyMarketResolver) Orders(ctx context.Context, market *Market,
	where *OrderFilter, skip *int, first *int, last *int) ([]types.Order, error) {
	err := validateMarket(ctx, &market.Name, r.marketService)
	if err != nil {
		return nil, err
	}
	queryFilters, err := buildOrderQueryFilters(where, skip, first, last)
	if err != nil {
		return nil, err
	}
	o, err := r.orderService.GetByMarket(ctx, market.Name, queryFilters)
	if err != nil {
		return nil, err
	}
	valOrders := make([]types.Order, 0)
	for _, v := range o {
		valOrders = append(valOrders, *v)
	}
	return valOrders, nil
}

func (r *MyMarketResolver) Trades(ctx context.Context, market *Market,
	where *TradeFilter, skip *int, first *int, last *int) ([]types.Trade, error) {
	err := validateMarket(ctx, &market.Name, r.marketService)
	if err != nil {
		return nil, err
	}
	queryFilters, err := buildTradeQueryFilters(where, skip, first, last)
	if err != nil {
		return nil, err
	}
	t, err := r.tradeService.GetByMarket(ctx, market.Name, queryFilters)
	if err != nil {
		return nil, err
	}
	valTrades := make([]types.Trade, 0)
	for _, v := range t {
		valTrades = append(valTrades, *v)
	}
	return valTrades, nil
}

func (r *MyMarketResolver) Depth(ctx context.Context, market *Market) (*types.MarketDepth, error) {
	if market == nil {
		return nil, errors.New("market missing or empty")

	}
	err := validateMarket(ctx, &market.Name, r.marketService)
	if err != nil {
		return nil, err
	}
	// Look for market depth for the given market (will validate market internally)
	// Note: Market depth is also known as OrderBook depth within the matching-engine
	depth, err := r.marketService.GetDepth(ctx, market.Name)
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
	sinceTimestamp, err := strconv.ParseUint(sinceTimestampRaw, 10, 64)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error converting %s into a valid timestamp", sinceTimestampRaw))
	}
	if len(sinceTimestampRaw) < 19 {
		return nil, errors.New("timestamp should be in epoch+nanoseconds format, eg. 1545158175835902621")
	}

	// Retrieve candles from store/service
	c, err := r.candleService.GetCandles(ctx, market.Name, sinceTimestamp, pbInterval)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// END: Market Resolver

// BEGIN: Party Resolver

type MyPartyResolver resolverRoot

func (r *MyPartyResolver) Orders(ctx context.Context, party *Party,
	where *OrderFilter, skip *int, first *int, last *int) ([]types.Order, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	queryFilters, err := buildOrderQueryFilters(where, skip, first, last)
	if err != nil {
		return nil, err
	}
	o, err := r.orderService.GetByParty(ctx, party.Name, queryFilters)
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
	where *TradeFilter, skip *int, first *int, last *int) ([]types.Trade, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	queryFilters, err := buildTradeQueryFilters(where, skip, first, last)
	if err != nil {
		return nil, err
	}
	t, err := r.tradeService.GetByParty(ctx, party.Name, queryFilters)
	if err != nil {
		return nil, err
	}
	valTrades := make([]types.Trade, 0)
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
	err := validateMarket(ctx, &obj.Name, r.marketService)
	if err != nil {
		return nil, err
	}
	queryFilters := &filtering.TradeQueryFilters{}
	last := uint64(1)
	queryFilters.Last = &last
	t, err := r.tradeService.GetByMarket(ctx, obj.Name, queryFilters)
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
		Name: obj.Market,
	}, nil
}
func (r *MyOrderResolver) Size(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *MyOrderResolver) Remaining(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}
func (r *MyOrderResolver) Timestamp(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Timestamp, 10), nil
}
func (r *MyOrderResolver) Status(ctx context.Context, obj *types.Order) (OrderStatus, error) {
	return OrderStatus(obj.Status.String()), nil
}
func (r *MyOrderResolver) Datetime(ctx context.Context, obj *types.Order) (string, error) {
	vegaTimestamp := vegatime.Stamp(obj.Timestamp)
	return vegaTimestamp.Rfc3339Nano(), nil
}
func (r *MyOrderResolver) Trades(ctx context.Context, obj *types.Order) ([]*types.Trade, error) {
	f := filtering.TradeQueryFilters{}
	relatedTrades, err := r.tradeService.GetByOrderId(ctx, obj.Id, &f)
	if err != nil {
		return nil, err
	}
	return relatedTrades, nil
}

// END: Order Resolver

// BEGIN: Trade Resolver

type MyTradeResolver resolverRoot

func (r *MyTradeResolver) Market(ctx context.Context, obj *types.Trade) (*Market, error) {
	return &Market{Name: obj.Market}, nil
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
	return strconv.FormatUint(obj.Timestamp, 10), nil
}
func (r *MyTradeResolver) Datetime(ctx context.Context, obj *types.Trade) (string, error) {
	vegaTimestamp := vegatime.Stamp(obj.Timestamp)
	return vegaTimestamp.Rfc3339Nano(), nil
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
	return vegatime.Stamp(obj.Timestamp).Rfc3339Nano(), nil
}
func (r *MyCandleResolver) Timestamp(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Timestamp, 10), nil
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
	return &Market{Name: obj.Market}, nil
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
	order := &types.Order{}
	res := PreConsensus{}
	if r.statusChecker.Blockchain.Status() != types.ChainStatus_CONNECTED {
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
	err = validateMarket(ctx, &market, r.marketService)
	if err != nil {
		return nil, err
	}
	order.Market = market
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
	order := &types.Order{}
	res := PreConsensus{}

	if r.statusChecker.Blockchain.Status() != types.ChainStatus_CONNECTED {
		return &res, ErrChainNotConnected
	}

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	err := validateMarket(ctx, &market, r.marketService)
	if err != nil {
		return nil, err
	}
	order.Market = market
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
	err := validateMarket(ctx, market, r.marketService)
	if err != nil {
		return nil, err
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	c, ref := r.orderService.ObserveOrders(ctx, market, party)
	logger := *r.GetLogger()
	logger.Debug("Orders: new subscriber", logging.Uint64("ref", ref))
	return c, nil
}

func (r *MySubscriptionResolver) Trades(ctx context.Context, market *string, party *string) (<-chan []types.Trade, error) {
	err := validateMarket(ctx, market, r.marketService)
	if err != nil {
		return nil, err
	}

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	c, ref := r.tradeService.ObserveTrades(ctx, market, party)
	logger := *r.GetLogger()
	logger.Debug("Trades: new subscriber", logging.Uint64("ref", ref))
	return c, nil
}

func (r *MySubscriptionResolver) Positions(ctx context.Context, party string) (<-chan *types.MarketPosition, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

	c, ref := r.tradeService.ObservePositions(ctx, party)
	logger := *r.GetLogger()
	logger.Debug("Positions: new subscriber", logging.Uint64("ref", ref))
	return c, nil
}

func (r *MySubscriptionResolver) MarketDepth(ctx context.Context, market string) (<-chan *types.MarketDepth, error) {
	err := validateMarket(ctx, &market, r.marketService)
	if err != nil {
		return nil, err
	}
	c, ref := r.marketService.ObserveDepth(ctx, market)
	logger := *r.GetLogger()
	logger.Debug("Market Depth: new subscriber", logging.Uint64("ref", ref))
	return c, nil
}

func (r *MySubscriptionResolver) Candles(ctx context.Context, market string, interval Interval) (<-chan *types.Candle, error) {
	err := validateMarket(ctx, &market, r.marketService)
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

	c, ref := r.candleService.ObserveCandles(ctx, &market, &pbInterval)

	logger.Debug("Candles: New subscriber",
		logging.String("interval", pbInterval.String()),
		logging.String("market", market),
		logging.Uint64("ref", ref))

	return c, nil
}

func validateMarket(ctx context.Context, marketId *string, marketService markets.Service) error {
	if marketId != nil {
		if len(*marketId) == 0 {
			return errors.New("market must not be empty")
		}
		_, err := marketService.GetByName(ctx, *marketId)
		if err != nil {
			return err
		}
	}
	return nil
}

// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)
//func validateParty(ctx context.Context, partyId *string, partyService parties.Service) error {
//	if partyId != nil {
//		if len(*partyId) == 0 {
//			return errors.New("party must not be empty")
//		}
//		_, err := partyService.GetByName(*partyId)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

// END: Subscription Resolver
