package gql

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"code.vegaprotocol.io/vega/internal/gateway"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"github.com/golang/protobuf/ptypes/empty"
)

var (
	ErrNilPendingOrder = errors.New("mil pending order")
)

type resolverRoot struct {
	gateway.Config

	log               *logging.Logger
	tradingClient     protoapi.TradingClient
	tradingDataClient protoapi.TradingDataClient
}

func NewResolverRoot(
	log *logging.Logger,
	config gateway.Config,
	tradingClient protoapi.TradingClient,
	tradingDataClient protoapi.TradingDataClient,
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
		{Name: pty.Name},
	}, nil
}

func (r *MyQueryResolver) Party(ctx context.Context, name string) (*Party, error) {
	req := protoapi.PartyByIDRequest{Id: name}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return &Party{Name: res.Party.Name}, nil
}

// END: Root Resolver

// BEGIN: Market Resolver

type MyMarketResolver resolverRoot

func (r *MyMarketResolver) Orders(ctx context.Context, market *Market,
	open *bool, skip *int, first *int, last *int) ([]types.Order, error) {
	/*
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
	*/
	return nil, nil
}

func (r *MyMarketResolver) Trades(ctx context.Context, market *Market,
	skip *int, first *int, last *int) ([]types.Trade, error) {
	/*
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
	*/
	return nil, nil
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
		Name: res.MarketID,
		Buy:  res.Buy,
		Sell: res.Sell,
	}, nil
}

func (r *MyMarketResolver) Candles(ctx context.Context, market *Market,
	sinceRaw string, interval Interval) ([]*types.Candle, error) {
	pinterval, err := convertInterval(interval)
	if err != nil {
		r.log.Warn("interval convert error", logging.Error(err))
	}

	since, err := vegatime.Parse(sinceRaw)
	if err != nil {
		return nil, err
	}

	req := protoapi.CandlesRequest{
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

func (r *MyPartyResolver) Orders(ctx context.Context, party *Party,
	open *bool, skip *int, first *int, last *int) ([]types.Order, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)
	/*
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
	*/
	return nil, nil
}

func (r *MyPartyResolver) Trades(ctx context.Context, party *Party,
	market *string, skip *int, first *int, last *int) ([]types.Trade, error) {

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)
	/*
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
	*/
	return nil, nil
}

func (r *MyPartyResolver) Positions(ctx context.Context, pty *Party) ([]types.MarketPosition, error) {
	if pty == nil {
		return nil, errors.New("nil party")
	}
	req := protoapi.PositionsByPartyRequest{PartyID: pty.Name}
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

	req := protoapi.LastTradeRequest{MarketID: md.Name}
	res, err := r.tradingDataClient.LastTrade(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return res.Trade, nil
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
	size string, side Side, type_ OrderType, expiration *string) (*types.PendingOrder, error) {
	/*
		order := &types.OrderSubmission{}

		if r.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
			return nil, ErrChainNotConnected
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
			//layout := "2006-01-02T15:04:05Z"
			// _, err := time.Parse(layout, *expiration)
			expiresAt, err := vegatime.Parse(*expiration)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("cannot parse expiration time: %s - invalid format sent to create order (example: 2018-01-02T15:04:05Z)", *expiration))
			}

			// move to pure timestamps or convert an RFC format shortly
			order.ExpiresAt = expiresAt.UnixNano()
		}

		// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
		pendingOrder, err := r.orderService.CreateOrder(ctx, order)
		if err != nil {
			r.log.Error("Failed to create order using rpc client in graphQL resolver", logging.Error(err))
			return nil, err
		}

		return pendingOrder, nil
	*/
	return nil, nil

}

func (r *MyMutationResolver) OrderCancel(ctx context.Context, id string, market string, party string) (*types.PendingOrder, error) {
	//order := &types.OrderCancellation{}
	/*

		if r.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
			return nil, ErrChainNotConnected
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
		pendingOrder, err := r.orderService.CancelOrder(ctx, order)
		if err != nil {
			return nil, err
		}

		return pendingOrder, nil
	*/
	return nil, nil
}

// END: Mutation Resolver

// BEGIN: Subscription Resolver

type MySubscriptionResolver resolverRoot

func (r *MySubscriptionResolver) Orders(ctx context.Context, market *string, party *string) (<-chan []types.Order, error) {
	/*
		_, err := validateMarket(ctx, market, r.marketService)
		if err != nil {
			return nil, err
		}

		// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

		c, ref := r.orderService.ObserveOrders(ctx, r.Config.GraphQLSubscriptionRetries, market, party)
		r.log.Debug("Orders: new subscriber", logging.Uint64("ref", ref))
		return c, nil
	*/
	return nil, nil
}

func (r *MySubscriptionResolver) Trades(ctx context.Context, market *string, party *string) (<-chan []types.Trade, error) {
	/*
		_, err := validateMarket(ctx, market, r.marketService)
		if err != nil {
			return nil, err
		}

		// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

		c, ref := r.tradeService.ObserveTrades(ctx, r.Config.GraphQLSubscriptionRetries, market, party)
		r.log.Debug("Trades: new subscriber", logging.Uint64("ref", ref))
		return c, nil
	*/
	return nil, nil
}

func (r *MySubscriptionResolver) Positions(ctx context.Context, party string) (<-chan *types.MarketPosition, error) {

	/*
		// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

		c, ref := r.tradeService.ObservePositions(ctx, r.Config.GraphQLSubscriptionRetries, party)
		r.log.Debug("Positions: new subscriber", logging.Uint64("ref", ref))
		return c, nil
	*/
	return nil, nil
}

func (r *MySubscriptionResolver) MarketDepth(ctx context.Context, market string) (<-chan *types.MarketDepth, error) {
	/*
		_, err := validateMarket(ctx, &market, r.marketService)
		if err != nil {
			return nil, err
		}
		c, ref := r.marketService.ObserveDepth(ctx, r.Config.GraphQLSubscriptionRetries, market)
		r.log.Debug("Market Depth: new subscriber", logging.Uint64("ref", ref))
		return c, nil
	*/
	return nil, nil
}

func (r *MySubscriptionResolver) Candles(ctx context.Context, market string, interval Interval) (<-chan *types.Candle, error) {
	/*
		_, err := validateMarket(ctx, &market, r.marketService)
		if err != nil {
			return nil, err
		}

		var pbInterval types.Interval
		switch interval {
		case IntervalI15m:
			pbInterval = types.Interval_I15M
		case IntervalI1d:
			pbInterval = types.Interval_I1D
		case IntervalI1h:
			pbInterval = types.Interval_I1H
		case IntervalI1m:
			pbInterval = types.Interval_I1M
		case IntervalI5m:
			pbInterval = types.Interval_I5M
		case IntervalI6h:
			pbInterval = types.Interval_I6H
		default:
			r.log.Warn("Invalid interval when subscribing to candles in gql, falling back to default: I15M",
				logging.String("interval", interval.String()))
			pbInterval = types.Interval_I15M
		}

		// Observe new candles for interval
		// --------------------------------

		c, ref := r.candleService.ObserveCandles(ctx, r.Config.GraphQLSubscriptionRetries, &market, &pbInterval)

		r.log.Debug("Candles: New subscriber",
			logging.String("interval", pbInterval.String()),
			logging.String("market", market),
			logging.Uint64("ref", ref))

		return c, nil
	*/
	return nil, nil
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
