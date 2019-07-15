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

// Errors
var (
	ErrFailedToLoadAccounts     = errors.New("failed to load accounts from store")
	ErrFailedToLoadCandles      = errors.New("failed to load candles from store")
	ErrFailedToLoadMarkets      = errors.New("failed to load markets from store")
	ErrFailedToLoadMarketDepth  = errors.New("failed to load market depth from store")
	ErrFailedToLoadOrders       = errors.New("failed to load orders from store")
	ErrFailedToLoadParties      = errors.New("failed to load parties from store")
	ErrFailedToLoadPositions    = errors.New("failed to load positions from store")
	ErrFailedToLoadStatistics   = errors.New("failed to load statistics")
	ErrFailedToLoadTrades       = errors.New("failed to load trades from store")
	ErrFailedToAmendOrder       = errors.New("failed to amend order")
	ErrFailedToCancelOrder      = errors.New("failed to cancel order")
	ErrFailedToSubmitOrder      = errors.New("failed to submit order")
	ErrFailedToSignIn           = errors.New("failed to sign in")
	ErrFailedToSubscribe        = errors.New("failed to subscribe")
	ErrInvalidExpirationTime    = errors.New("invalid expiration time")
	ErrInvalidPrice             = errors.New("invalid price")
	ErrInvalidSize              = errors.New("invalid size")
	ErrNilCandle                = errors.New("no candle supplied")
	ErrNilOrder                 = errors.New("no order supplied")
	ErrNilOrderID               = errors.New("no order ID supplied")
	ErrNilMarketDepth           = errors.New("no market depth supplied")
	ErrNilMarketPosition        = errors.New("no market position supplied")
	ErrNilParty                 = errors.New("no party supplied")
	ErrNilPendingOrder          = errors.New("no pending order supplied")
	ErrNotImplementedAccounts   = errors.New("not implemented: accounts")
	ErrNotImplementedAllParties = errors.New("not implemented: all parties")
	ErrUnknownAccountType       = errors.New("unknown account type")
)

// Log messages
var (
	failedToLoad    = "failed to load object(s) from store"
	failedToConvert = "failed to convert object(s) from proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_client_mock.go -package mocks code.vegaprotocol.io/vega/internal/gateway/graphql TradingClient
type TradingClient interface {
	// unary calls - writes
	SubmitOrder(ctx context.Context, in *protoapi.SubmitOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, in *protoapi.CancelOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, in *protoapi.AmendOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	SignIn(ctx context.Context, in *protoapi.SignInRequest, opts ...grpc.CallOption) (*protoapi.SignInResponse, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_data_client_mock.go -package mocks code.vegaprotocol.io/vega/internal/gateway/graphql TradingDataClient
type TradingDataClient interface {
	// orders
	OrdersByMarket(ctx context.Context, in *protoapi.OrdersByMarketRequest, opts ...grpc.CallOption) (*protoapi.OrdersByMarketResponse, error)
	OrderByReference(ctx context.Context, in *protoapi.OrderByReferenceRequest, opts ...grpc.CallOption) (*protoapi.OrderByReferenceResponse, error)
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
	AccountsSubscribe(ctx context.Context, in *protoapi.AccountsSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_AccountsSubscribeClient, error)
	OrdersSubscribe(ctx context.Context, in *protoapi.OrdersSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_OrdersSubscribeClient, error)
	TradesSubscribe(ctx context.Context, in *protoapi.TradesSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_TradesSubscribeClient, error)
	CandlesSubscribe(ctx context.Context, in *protoapi.CandlesSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_CandlesSubscribeClient, error)
	MarketDepthSubscribe(ctx context.Context, in *protoapi.MarketDepthSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_MarketDepthSubscribeClient, error)
	PositionsSubscribe(ctx context.Context, in *protoapi.PositionsSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_PositionsSubscribeClient, error)
	// collateral
	TraderAccounts(ctx context.Context, req *protoapi.CollateralRequest, opts ...grpc.CallOption) (*protoapi.CollateralResponse, error)
	TraderMarketAccounts(ctx context.Context, req *protoapi.CollateralRequest, opts ...grpc.CallOption) (*protoapi.CollateralResponse, error)
	TraderMarketBalance(ctx context.Context, req *protoapi.CollateralRequest, opts ...grpc.CallOption) (*protoapi.CollateralResponse, error)
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
func (r *resolverRoot) Account() AccountResolver {
	return (*MyAccountResolver)(r)
}

func (r *resolverRoot) Statistics() StatisticsResolver {
	return (*MyStatisticsResolver)(r)
}

func objLoadFailError(store, objid string) error {
	return fmt.Errorf("failed to load %s from store: %s", store, objid)
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
		r.log.Debug(failedToLoad, logging.Error(err))
		return nil, ErrFailedToLoadMarkets
	}

	m := make([]Market, 0, len(res.Markets))
	for _, pmarket := range res.Markets {
		market, err := MarketFromProto(pmarket)
		if err != nil {
			r.log.Debug(failedToConvert, logging.Error(err))
			return nil, ErrFailedToLoadMarkets
		}
		m = append(m, *market)
	}

	return m, nil
}

func (r *MyQueryResolver) Market(ctx context.Context, id string) (*Market, error) {
	req := protoapi.MarketByIDRequest{MarketID: id}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad, logging.Error(err))
		return nil, objLoadFailError("Market", id)
	}

	market, err := MarketFromProto(res.Market)
	if err != nil {
		r.log.Debug(failedToConvert, logging.Error(err))
		return nil, objLoadFailError("Market", id)
	}
	return market, nil
}

func (r *MyQueryResolver) Parties(ctx context.Context, name *string) ([]Party, error) {
	if name == nil {
		return nil, ErrNotImplementedAllParties
	}
	pty, err := r.Party(ctx, *name)
	if err != nil {
		r.log.Debug(failedToLoad, logging.Error(err))
		return nil, ErrFailedToLoadParties
	}
	return []Party{
		{ID: pty.ID},
	}, nil
}

func (r *MyQueryResolver) Party(ctx context.Context, name string) (*Party, error) {
	req := protoapi.PartyByIDRequest{Id: name}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad, logging.Error(err))
		return nil, objLoadFailError("Party", name)
	}

	return &Party{ID: res.Party.Id}, nil
}

func (r *MyQueryResolver) Statistics(ctx context.Context) (*types.Statistics, error) {
	res, err := r.tradingDataClient.Statistics(ctx, &empty.Empty{})
	if err != nil {
		r.log.Debug(failedToLoad, logging.Error(err))
		return nil, ErrFailedToLoadStatistics
	}
	return res, nil
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
		r.log.Debug(failedToLoad,
			logging.Error(err),
			logging.MarketID(market.ID),
		)
		return nil, ErrFailedToLoadOrders
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
		r.log.Debug(failedToLoad,
			logging.Error(err),
			logging.MarketID(market.ID),
		)
		return nil, ErrFailedToLoadTrades
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
		return nil, ErrNilMarket
	}

	req := protoapi.MarketDepthRequest{MarketID: market.ID}
	// Look for market depth for the given market (will validate market internally)
	// Note: Market depth is also known as OrderBook depth within the matching-engine
	res, err := r.tradingDataClient.MarketDepth(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.Error(err),
			logging.MarketID(market.ID),
		)
		return nil, ErrFailedToLoadMarketDepth
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
		MarketID:       mkt,
		SinceTimestamp: since.UnixNano(),
		Interval:       pinterval,
	}
	res, err := r.tradingDataClient.Candles(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.Error(err),
			logging.MarketID(mkt),
		)
		return nil, ErrFailedToLoadCandles
	}
	return res.Candles, nil
}

func (r *MyMarketResolver) OrderByReference(ctx context.Context, market *Market,
	ref string) (*types.Order, error) {

	req := protoapi.OrderByReferenceRequest{
		Reference: ref,
	}
	res, err := r.tradingDataClient.OrderByReference(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.Error(err),
			logging.MarketID(market.ID),
		)
		return nil, objLoadFailError("Order", ref)
	}
	return res.Order, nil
}

func (r *MyMarketResolver) Accounts(ctx context.Context, market *Market, accType *AccountType) ([]types.Account, error) {
	return nil, ErrNotImplementedAccounts
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
		r.log.Debug(failedToLoad,
			logging.Error(err),
			logging.PartyID(party.ID),
		)
		return nil, ErrFailedToLoadOrders
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
		r.log.Debug(failedToLoad,
			logging.Error(err),
			logging.MarketID(mkt),
			logging.PartyID(party.ID),
		)
		return nil, ErrFailedToLoadTrades
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
		return nil, ErrNilParty
	}
	req := protoapi.PositionsByPartyRequest{PartyID: pty.ID}
	res, err := r.tradingDataClient.PositionsByParty(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.PartyID(pty.ID),
			logging.Error(err),
		)
		return nil, ErrFailedToLoadPositions
	}

	retpos := make([]types.MarketPosition, 0, len(res.Positions))
	for _, v := range res.Positions {
		v := v
		retpos = append(retpos, *v)
	}
	return retpos, nil

}

func (r *MyPartyResolver) Accounts(ctx context.Context, pty *Party, marketID *string, accType *AccountType) ([]types.Account, error) {
	if pty == nil {
		return nil, ErrNilParty
	}
	// the call we'll be making
	call := r.tradingDataClient.TraderAccounts
	var (
		market string
		at     types.AccountType
	)
	if marketID != nil {
		market = *marketID
		// if a market was given, assume we want the market accounts
		call = r.tradingDataClient.TraderMarketAccounts
	}
	if accType != nil {
		// if an account type was specified, we'll be getting the balance (hacky, but simplifies this temp API)
		switch *accType {
		case AccountTypeMargin:
			at = types.AccountType_MARGIN
		case AccountTypeGeneral:
			at = types.AccountType_GENERAL
		case AccountTypeInsurance:
			at = types.AccountType_INSURANCE
		case AccountTypeSettlement:
			at = types.AccountType_SETTLEMENT
		}
		call = r.tradingDataClient.TraderMarketBalance
	}
	req := protoapi.CollateralRequest{
		Party:    pty.ID,
		MarketID: market,
		Type:     at,
	}
	resp, err := call(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.AccountType(at),
			logging.MarketID(market),
			logging.PartyID(pty.ID),
			logging.Error(err),
		)
		return nil, ErrFailedToLoadAccounts
	}
	accounts := make([]types.Account, 0, len(resp.Accounts))
	for _, acc := range resp.Accounts {
		accounts = append(accounts, *acc)
	}
	return accounts, nil
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
		return nil, ErrNilMarketDepth
	}

	req := protoapi.LastTradeRequest{MarketID: md.MarketID}
	res, err := r.tradingDataClient.LastTrade(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.MarketID(md.MarketID),
			logging.Error(err),
		)
		return nil, err
	}

	return res.Trade, nil
}

func (r *MyMarketDepthResolver) Market(ctx context.Context, md *types.MarketDepth) (*Market, error) {
	if md == nil {
		return nil, ErrNilMarketDepth
	}

	req := protoapi.MarketByIDRequest{MarketID: md.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.MarketID(md.MarketID),
			logging.Error(err),
		)
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
	if obj == nil {
		return nil, ErrNilOrder
	}

	req := protoapi.MarketByIDRequest{MarketID: obj.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.MarketID(obj.MarketID),
			logging.OrderID(obj.Id),
			logging.Error(err),
		)
		return nil, objLoadFailError("Market", obj.MarketID)
	}
	return MarketFromProto(res.Market)
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
		return nil, ErrNilOrder
	}

	req := protoapi.TradesByOrderRequest{OrderID: ord.Id}
	res, err := r.tradingDataClient.TradesByOrder(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.OrderID(ord.Id),
			logging.Error(err),
		)
		return nil, ErrFailedToLoadTrades
	}
	return res.Trades, nil
}
func (r *MyOrderResolver) Party(ctx context.Context, ord *types.Order) (*Party, error) {
	if ord == nil {
		return nil, ErrNilOrder
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
	if obj == nil {
		return IntervalI15m, ErrNilCandle
	}
	interval := Interval(obj.Interval.String())
	if interval.IsValid() {
		return interval, nil
	} else {
		r.log.Debug("Interval conversion from proto to gql type failed, falling back to default: I15m",
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
	if obj == nil {
		return nil, ErrNilMarketPosition
	}

	req := protoapi.MarketByIDRequest{MarketID: obj.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.MarketID(obj.MarketID),
			logging.Error(err),
		)
		return nil, objLoadFailError("Market", obj.MarketID)
	}
	return MarketFromProto(res.Market)
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

	tkn := gateway.TokenFromContext(ctx)

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
		return nil, ErrNilMarket
	}
	order.MarketID = market

	if len(party) <= 0 {
		return nil, ErrNilParty
	}
	order.PartyID = party

	// todo: add party-store/party-service validation (gitlab.com/vega-protocol/trading-core/issues/175)

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
			r.log.Debug("Failed to parse expiration time",
				logging.String("expiration_string", *expiration),
				logging.Error(err),
			)
			return nil, ErrInvalidExpirationTime
		}

		// move to pure timestamps or convert an RFC format shortly
		order.ExpiresAt = expiresAt.UnixNano()
	}

	req := protoapi.SubmitOrderRequest{
		Submission: order,
		Token:      tkn,
	}

	// Pass the order over for consensus (service layer will use RPC client internally and handle errors etc)
	pendingOrder, err := r.tradingClient.SubmitOrder(ctx, &req)
	if err != nil {
		r.log.Error("Failed to submit order",
			logging.Error(err),
		)
		return nil, ErrFailedToSubmitOrder
	}

	return pendingOrder, nil
}

func (r *MyMutationResolver) OrderCancel(ctx context.Context, id string, party string, market string) (*types.PendingOrder, error) {
	order := &types.OrderCancellation{}

	tkn := gateway.TokenFromContext(ctx)

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	if len(market) <= 0 {
		return nil, ErrNilMarket
	}
	order.MarketID = market
	if len(id) == 0 {
		return nil, ErrNilOrderID
	}
	order.OrderID = id
	if len(party) == 0 {
		return nil, ErrNilParty
	}
	order.PartyID = party

	// Pass the cancellation over for consensus (service layer will use RPC client internally and handle errors etc)

	req := protoapi.CancelOrderRequest{
		Cancellation: order,
		Token:        tkn,
	}
	pendingOrder, err := r.tradingClient.CancelOrder(ctx, &req)
	if err != nil {
		r.log.Error("Failed to cancel order",
			logging.Error(err),
		)
		return nil, ErrFailedToCancelOrder
	}

	return pendingOrder, nil
}

func (r *MyMutationResolver) OrderAmend(ctx context.Context, id string, party string, price, size int, expiration *string) (*types.PendingOrder, error) {
	order := &types.OrderAmendment{}

	tkn := gateway.TokenFromContext(ctx)

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	if len(id) == 0 {
		return nil, ErrNilOrderID
	}
	order.OrderID = id
	if len(party) == 0 {
		return nil, ErrNilParty
	}
	order.PartyID = party
	if price < 0 {
		return nil, ErrInvalidPrice
	}
	order.Price = uint64(price)
	if size < 0 {
		return nil, ErrInvalidSize
	}
	order.Size = uint64(size)
	if expiration != nil {
		expiresAt, err := vegatime.Parse(*expiration)
		if err != nil {
			r.log.Debug("Failed to parse expiration time",
				logging.String("expiration_string", *expiration),
				logging.Error(err),
			)
			return nil, ErrInvalidExpirationTime
		}
		// move to pure timestamps or convert an RFC format shortly
		order.ExpiresAt = expiresAt.UnixNano()
	}

	req := protoapi.AmendOrderRequest{
		Amendment: order,
		Token:     tkn,
	}
	pendingOrder, err := r.tradingClient.AmendOrder(ctx, &req)
	if err != nil {
		r.log.Error("Failed to amend order",
			logging.Error(err),
		)
		return nil, ErrFailedToAmendOrder
	}

	return pendingOrder, nil
}

func (r *MyMutationResolver) Signin(ctx context.Context, id string, password string) (string, error) {
	req := protoapi.SignInRequest{
		Id:       id,
		Password: password,
	}

	res, err := r.tradingClient.SignIn(ctx, &req)
	if err != nil {
		r.log.Debug("Failed to sign in",
			logging.String("id", id),
			logging.Error(err),
		)
		// DO NOT leak details from a signin failure.
		return "", ErrFailedToSignIn
	}

	return res.Token, nil
}

// END: Mutation Resolver

// BEGIN: Subscription Resolver

type MySubscriptionResolver resolverRoot

func (r *MySubscriptionResolver) Accounts(ctx context.Context, market *string, party *string, typeArg *AccountType) (<-chan *proto.Account, error) {
	var (
		mkt, pty string
		ty       types.AccountType
	)

	if market != nil {
		mkt = *market
	}
	if party != nil {
		pty = *party
	}
	if typeArg != nil {
		ty = typeArg.IntoProto()
	}

	req := &api.AccountsSubscribeRequest{
		MarketID: mkt,
		PartyID:  pty,
		Type:     ty,
	}
	stream, err := r.tradingDataClient.AccountsSubscribe(ctx, req)
	if err != nil {
		r.log.Debug("Failed to subscribe to accounts",
			logging.MarketID(mkt),
			logging.PartyID(pty),
			logging.Error(err),
		)
		return nil, ErrFailedToSubscribe
	}

	c := make(chan *types.Account)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			a, err := stream.Recv()
			if err == io.EOF {
				r.log.Debug("accounts: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Debug("accounts: stream closed", logging.Error(err))
				break
			}
			c <- a
		}
	}()

	return c, nil
}

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
		r.log.Debug("Failed to subscribe to orders",
			logging.MarketID(mkt),
			logging.PartyID(pty),
			logging.Error(err),
		)
		return nil, ErrFailedToSubscribe
	}

	c := make(chan []types.Order)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			o, err := stream.Recv()
			if err == io.EOF {
				r.log.Debug("orders: stream closed by server",
					logging.MarketID(mkt),
					logging.PartyID(pty),
					logging.Error(err),
				)
				break
			}
			if err != nil {
				r.log.Debug("orders: stream closed",
					logging.MarketID(mkt),
					logging.PartyID(pty),
					logging.Error(err),
				)
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
		r.log.Debug("Failed to subscribe to trades",
			logging.MarketID(mkt),
			logging.PartyID(pty),
			logging.Error(err),
		)
		return nil, ErrFailedToSubscribe
	}

	c := make(chan []types.Trade)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				r.log.Debug("trades: stream closed by server",
					logging.MarketID(mkt),
					logging.PartyID(pty),
					logging.Error(err),
				)
				break
			}
			if err != nil {
				r.log.Debug("trades: stream closed",
					logging.MarketID(mkt),
					logging.PartyID(pty),
					logging.Error(err),
				)
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
		r.log.Debug("Failed to subscribe to positions",
			logging.PartyID(party),
			logging.Error(err),
		)
		return nil, ErrFailedToSubscribe
	}

	c := make(chan *types.MarketPosition)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				r.log.Debug("positions: stream closed by server",
					logging.PartyID(party),
					logging.Error(err),
				)
				break
			}
			if err != nil {
				r.log.Debug("positions: stream closed",
					logging.PartyID(party),
					logging.Error(err),
				)
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
		r.log.Debug("Failed to subscribe to market depth",
			logging.MarketID(market),
			logging.Error(err),
		)
		return nil, ErrFailedToSubscribe
	}

	c := make(chan *types.MarketDepth)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			md, err := stream.Recv()
			if err == io.EOF {
				r.log.Debug("marketDepth: stream closed by server",
					logging.MarketID(market),
					logging.Error(err),
				)
				break
			}
			if err != nil {
				r.log.Debug("marketDepth: stream closed",
					logging.MarketID(market),
					logging.Error(err),
				)
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
		r.log.Debug("Failed to subscribe to candles",
			logging.MarketID(market),
			logging.Error(err),
		)
		return nil, ErrFailedToSubscribe
	}

	c := make(chan *types.Candle)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			cdl, err := stream.Recv()
			if err == io.EOF {
				r.log.Debug("candles: stream closed by server",
					logging.MarketID(market),
					logging.Error(err),
				)
				break
			}
			if err != nil {
				r.log.Debug("candles: stream closed",
					logging.MarketID(market),
					logging.Error(err),
				)
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
		return nil, ErrNilPendingOrder
	}

	req := protoapi.MarketByIDRequest{MarketID: pord.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Debug(failedToLoad,
			logging.PendingOrder(*pord),
			logging.Error(err),
		)
		return nil, objLoadFailError("Market", pord.MarketID)
	}
	return MarketFromProto(res.Market)
}

func (r *MyPendingOrderResolver) Party(ctx context.Context, pord *proto.PendingOrder) (*Party, error) {
	if pord == nil {
		return nil, ErrNilPendingOrder
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

// START: Account Resolver

// MyAccountResolver - seems to be required by gqlgen, but we're not using this ATM
type MyAccountResolver resolverRoot

func (r *MyAccountResolver) Balance(ctx context.Context, acc *proto.Account) (string, error) {
	bal := fmt.Sprintf("%d", acc.Balance)
	return bal, nil
}

func (r *MyAccountResolver) Type(ctx context.Context, obj *proto.Account) (AccountType, error) {
	var t AccountType
	switch obj.Type {
	case types.AccountType_MARGIN:
		return AccountTypeMargin, nil
	case types.AccountType_GENERAL:
		return AccountTypeGeneral, nil
	case types.AccountType_INSURANCE:
		return AccountTypeInsurance, nil
	case types.AccountType_SETTLEMENT:
		return AccountTypeSettlement, nil
	}
	return t, ErrUnknownAccountType
}

// END: Account Resolver

type MyStatisticsResolver resolverRoot

func (r *MyStatisticsResolver) BlockHeight(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.BlockHeight), nil
}

func (r *MyStatisticsResolver) BacklogLength(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.BacklogLength), nil
}

func (r *MyStatisticsResolver) TotalPeers(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalPeers), nil
}

func (r *MyStatisticsResolver) Status(ctx context.Context, obj *proto.Statistics) (string, error) {
	return obj.Status.String(), nil
}

func (r *MyStatisticsResolver) TxPerBlock(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TxPerBlock), nil
}

func (r *MyStatisticsResolver) AverageTxBytes(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.AverageTxBytes), nil
}

func (r *MyStatisticsResolver) AverageOrdersPerBlock(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.AverageOrdersPerBlock), nil
}

func (r *MyStatisticsResolver) TradesPerSecond(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TradesPerSecond), nil
}

func (r *MyStatisticsResolver) OrdersPerSecond(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.OrdersPerSecond), nil
}

func (r *MyStatisticsResolver) TotalMarkets(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalMarkets), nil
}

func (r *MyStatisticsResolver) TotalParties(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalParties), nil
}

func (r *MyStatisticsResolver) TotalAmendOrder(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalAmendOrder), nil
}

func (r *MyStatisticsResolver) TotalCancelOrder(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalCancelOrder), nil
}

func (r *MyStatisticsResolver) TotalCreateOrder(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalCreateOrder), nil
}

func (r *MyStatisticsResolver) TotalOrders(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalOrders), nil
}

func (r *MyStatisticsResolver) TotalTrades(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalTrades), nil
}

func (r *MyStatisticsResolver) BlockDuration(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.BlockDuration), nil
}
