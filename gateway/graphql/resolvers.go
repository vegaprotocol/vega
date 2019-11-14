package gql

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/proto/api"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/vegatime"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
)

var (
	// ErrNilPendingOrder a pending order was nil when expected not to be
	ErrNilPendingOrder = errors.New("nil pending order")
	// ErrUnknownAccountType a account type specified does not exist
	ErrUnknownAccountType = errors.New("unknown account type")
)

// TradingClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_client_mock.go -package mocks code.vegaprotocol.io/vega/gateway/graphql TradingClient
type TradingClient interface {
	// unary calls - writes
	SubmitOrder(ctx context.Context, in *protoapi.SubmitOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, in *protoapi.CancelOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, in *protoapi.AmendOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	SignIn(ctx context.Context, in *protoapi.SignInRequest, opts ...grpc.CallOption) (*protoapi.SignInResponse, error)
	// unary calls - reads
	CheckToken(context.Context, *api.CheckTokenRequest, ...grpc.CallOption) (*api.CheckTokenResponse, error)
}

// TradingDataClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_data_client_mock.go -package mocks code.vegaprotocol.io/vega/gateway/graphql TradingDataClient
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
	// accounts
	AccountsByParty(ctx context.Context, req *protoapi.AccountsByPartyRequest, opts ...grpc.CallOption) (*protoapi.AccountsByPartyResponse, error)
	AccountsByPartyAndMarket(ctx context.Context, req *protoapi.AccountsByPartyAndMarketRequest, opts ...grpc.CallOption) (*protoapi.AccountsByPartyAndMarketResponse, error)
	AccountsByPartyAndType(ctx context.Context, req *protoapi.AccountsByPartyAndTypeRequest, opts ...grpc.CallOption) (*protoapi.AccountsByPartyAndTypeResponse, error)
	AccountsByPartyAndAsset(ctx context.Context, req *protoapi.AccountsByPartyAndAssetRequest, opts ...grpc.CallOption) (*protoapi.AccountsByPartyAndAssetResponse, error)
}

// VegaResolverRoot is the root resolver for all graphql types
type VegaResolverRoot struct {
	gateway.Config

	log               *logging.Logger
	tradingClient     TradingClient
	tradingDataClient TradingDataClient
}

// NewResolverRoot instanciate a graphql root resolver
func NewResolverRoot(
	log *logging.Logger,
	config gateway.Config,
	tradingClient TradingClient,
	tradingDataClient TradingDataClient,
) *VegaResolverRoot {

	return &VegaResolverRoot{
		log:               log,
		Config:            config,
		tradingClient:     tradingClient,
		tradingDataClient: tradingDataClient,
	}
}

// Query returns the query resolver
func (r *VegaResolverRoot) Query() QueryResolver {
	return (*myQueryResolver)(r)
}

// Mutation returns the mutations resolver
func (r *VegaResolverRoot) Mutation() MutationResolver {
	return (*myMutationResolver)(r)
}

// Candle returns the candles resolver
func (r *VegaResolverRoot) Candle() CandleResolver {
	return (*myCandleResolver)(r)
}

// MarketDepth returns the market depth resolver
func (r *VegaResolverRoot) MarketDepth() MarketDepthResolver {
	return (*myMarketDepthResolver)(r)
}

// PriceLevel returns the price levels resolver
func (r *VegaResolverRoot) PriceLevel() PriceLevelResolver {
	return (*myPriceLevelResolver)(r)
}

// Market returns the markets resolver
func (r *VegaResolverRoot) Market() MarketResolver {
	return (*myMarketResolver)(r)
}

// Order returns the order resolver
func (r *VegaResolverRoot) Order() OrderResolver {
	return (*myOrderResolver)(r)
}

// Trade returns the trades resolver
func (r *VegaResolverRoot) Trade() TradeResolver {
	return (*myTradeResolver)(r)
}

// Position returns the positions resolver
func (r *VegaResolverRoot) Position() PositionResolver {
	return (*myPositionResolver)(r)
}

// Party returns the parties resolver
func (r *VegaResolverRoot) Party() PartyResolver {
	return (*myPartyResolver)(r)
}

// Subscription returns the subscriptions resolver
func (r *VegaResolverRoot) Subscription() SubscriptionResolver {
	return (*mySubscriptionResolver)(r)
}

// PendingOrder returns the pending orders resolver
func (r *VegaResolverRoot) PendingOrder() PendingOrderResolver {
	return (*myPendingOrderResolver)(r)
}

// Account returns the accounts resolver
func (r *VegaResolverRoot) Account() AccountResolver {
	return (*myAccountResolver)(r)
}

// Statistics returns the statistics resolver
func (r *VegaResolverRoot) Statistics() StatisticsResolver {
	return (*myStatisticsResolver)(r)
}

// BEGIN: Query Resolver

type myQueryResolver VegaResolverRoot

func (r *myQueryResolver) Markets(ctx context.Context, id *string) ([]*Market, error) {
	if id != nil {
		mkt, err := r.Market(ctx, *id)
		if err != nil {
			return nil, err
		}
		return []*Market{mkt}, nil

	}
	res, err := r.tradingDataClient.Markets(ctx, &empty.Empty{})
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	m := make([]*Market, 0, len(res.Markets))
	for _, pmarket := range res.Markets {
		market, err := MarketFromProto(pmarket)
		if err != nil {
			r.log.Error("unable to convert market from proto", logging.Error(err))
			return nil, err
		}
		m = append(m, market)
	}

	return m, nil
}

func (r *myQueryResolver) Market(ctx context.Context, id string) (*Market, error) {
	req := protoapi.MarketByIDRequest{MarketID: id}
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

func (r *myQueryResolver) Parties(ctx context.Context, name *string) ([]*Party, error) {
	if name == nil {
		return nil, errors.New("all parties not implemented")
	}
	pty, err := r.Party(ctx, *name)
	if err != nil {
		return nil, err
	}
	return []*Party{
		&Party{ID: pty.ID},
	}, nil
}

func (r *myQueryResolver) Party(ctx context.Context, name string) (*Party, error) {
	req := protoapi.PartyByIDRequest{PartyID: name}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}

	return &Party{ID: res.Party.Id}, nil
}

func (r *myQueryResolver) Statistics(ctx context.Context) (*types.Statistics, error) {
	res, err := r.tradingDataClient.Statistics(ctx, &empty.Empty{})
	if err != nil {
		r.log.Error("tradingCore client", logging.Error(err))
		return nil, err
	}
	return res, nil
}

func (r *myQueryResolver) CheckToken(ctx context.Context, partyID string, token string) (*CheckTokenResponse, error) {
	req := &protoapi.CheckTokenRequest{
		PartyID: partyID,
		Token:   token,
	}

	response, err := r.tradingClient.CheckToken(ctx, req)
	if err != nil {
		return nil, err
	}

	return &CheckTokenResponse{Ok: response.Ok}, nil
}

// END: Root Resolver

// BEGIN: Market Resolver

type myMarketResolver VegaResolverRoot

func (r *myMarketResolver) Orders(
	ctx context.Context, market *Market, open *bool, skip *int, first *int, last *int,
) ([]*types.Order, error) {
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
	return res.Orders, nil
}

func (r *myMarketResolver) Trades(ctx context.Context, market *Market,
	skip *int, first *int, last *int) ([]*types.Trade, error) {
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

	return res.Trades, nil
}

func (r *myMarketResolver) Depth(ctx context.Context, market *Market) (*types.MarketDepth, error) {

	if market == nil {
		return nil, errors.New("market missing or empty")

	}

	req := protoapi.MarketDepthRequest{MarketID: market.ID}
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

func (r *myMarketResolver) Candles(ctx context.Context, market *Market,
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
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Candles, nil
}

func (r *myMarketResolver) OrderByReference(ctx context.Context, market *Market,
	ref string) (*types.Order, error) {

	req := protoapi.OrderByReferenceRequest{
		Reference: ref,
	}
	res, err := r.tradingDataClient.OrderByReference(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Order, nil
}

func (r *myMarketResolver) Accounts(ctx context.Context, market *Market, accType *AccountType) ([]*types.Account, error) {
	return nil, errors.New("not implemented yet")
}

// END: Market Resolver

// BEGIN: Party Resolver

type myPartyResolver VegaResolverRoot

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

func (r *myPartyResolver) Orders(ctx context.Context, party *Party,
	open *bool, skip *int, first *int, last *int) ([]*types.Order, error) {

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

	outorders := make([]*types.Order, 0, len(res.Orders))
	for _, v := range res.Orders {
		v := v
		outorders = append(outorders, v)
	}
	return outorders, nil
}

func (r *myPartyResolver) Trades(ctx context.Context, party *Party,
	market *string, skip *int, first *int, last *int) ([]*types.Trade, error) {

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
	return res.Trades, nil
}

func (r *myPartyResolver) Positions(ctx context.Context, pty *Party) ([]*types.MarketPosition, error) {
	if pty == nil {
		return nil, errors.New("nil party")
	}
	req := protoapi.PositionsByPartyRequest{PartyID: pty.ID}
	res, err := r.tradingDataClient.PositionsByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return res.Positions, nil
}

func (r *myPartyResolver) Accounts(ctx context.Context, pty *Party, marketID *string, asset *string, accType *AccountType) ([]*types.Account, error) {
	if pty == nil {
		return nil, errors.New("a party must be specified when querying accounts")
	}
	// Ensure default account types values
	general := AccountTypeGeneral
	margin := AccountTypeMargin
	if accType == nil {
		if marketID != nil {
			accType = &margin
		} else {
			accType = &general
		}
	}
	// Depending on Accounts params, select the correct gRPC call
	switch *accType {
	case AccountTypeMargin:
		if marketID == nil {
			return nil, errors.New("a market must be specified when querying for a margin account")
		}
		req := protoapi.AccountsByPartyAndMarketRequest{
			PartyID:  pty.ID,
			MarketID: *marketID,
			Type:     types.AccountType_MARGIN,
		}
		resp, err := r.tradingDataClient.AccountsByPartyAndMarket(ctx, &req)
		if err != nil {
			return nil, err
		}
		accounts := make([]*types.Account, 0, len(resp.Accounts))
		for _, acc := range resp.Accounts {
			if asset == nil || acc.Asset == *asset {
				accounts = append(accounts, acc)
			}
		}
		return accounts, nil
	case AccountTypeGeneral:
		req := protoapi.AccountsByPartyRequest{
			PartyID: pty.ID,
			Type:    types.AccountType_GENERAL,
		}
		resp, err := r.tradingDataClient.AccountsByParty(ctx, &req)
		if err != nil {
			return nil, err
		}
		accounts := make([]*types.Account, 0, len(resp.Accounts))
		for _, acc := range resp.Accounts {
			acc := acc
			if asset == nil || acc.Asset == *asset {
				accounts = append(accounts, acc)
			}
		}
		return accounts, nil
	}
	//Note: there's currently no read store for the following account types
	// AccountTypeInsurance
	// AccountTypeSettlement
	return nil, errors.New("account type specified is not supported")
}

// END: Party Resolver

// BEGIN: Market Depth Resolver

type myMarketDepthResolver VegaResolverRoot

func (r *myMarketDepthResolver) Buy(ctx context.Context, obj *types.MarketDepth) ([]types.PriceLevel, error) {
	valBuyLevels := make([]types.PriceLevel, 0)
	for _, v := range obj.Buy {
		valBuyLevels = append(valBuyLevels, *v)
	}
	return valBuyLevels, nil
}
func (r *myMarketDepthResolver) Sell(ctx context.Context, obj *types.MarketDepth) ([]types.PriceLevel, error) {
	valBuyLevels := make([]types.PriceLevel, 0)
	for _, v := range obj.Sell {
		valBuyLevels = append(valBuyLevels, *v)
	}
	return valBuyLevels, nil
}

func (r *myMarketDepthResolver) LastTrade(ctx context.Context, md *types.MarketDepth) (*types.Trade, error) {
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

func (r *myMarketDepthResolver) Market(ctx context.Context, md *types.MarketDepth) (*Market, error) {
	if md == nil {
		return nil, errors.New("invalid market depth")
	}

	req := protoapi.MarketByIDRequest{MarketID: md.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return MarketFromProto(res.Market)
}

// END: Market Depth Resolver

// BEGIN: Order Resolver

type myOrderResolver VegaResolverRoot

func (r *myOrderResolver) Price(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}
func (r *myOrderResolver) TimeInForce(ctx context.Context, obj *types.Order) (OrderTimeInForce, error) {
	return OrderTimeInForce(obj.TimeInForce.String()), nil
}

func (r *myOrderResolver) Type(ctx context.Context, obj *types.Order) (*OrderType, error) {
	t := OrderType(obj.Type.String())
	return &t, nil
}

func (r *myOrderResolver) Side(ctx context.Context, obj *types.Order) (Side, error) {
	return Side(obj.Side.String()), nil
}
func (r *myOrderResolver) Market(ctx context.Context, obj *types.Order) (*Market, error) {
	if obj == nil {
		return nil, errors.New("invalid order")
	}

	req := protoapi.MarketByIDRequest{MarketID: obj.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return MarketFromProto(res.Market)
}
func (r *myOrderResolver) Size(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *myOrderResolver) Remaining(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}
func (r *myOrderResolver) Timestamp(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatInt(obj.CreatedAt, 10), nil
}
func (r *myOrderResolver) Status(ctx context.Context, obj *types.Order) (OrderStatus, error) {
	return OrderStatus(obj.Status.String()), nil
}
func (r *myOrderResolver) Datetime(ctx context.Context, obj *types.Order) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}
func (r *myOrderResolver) CreatedAt(ctx context.Context, obj *types.Order) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedAt)), nil
}
func (r *myOrderResolver) ExpiresAt(ctx context.Context, obj *types.Order) (*string, error) {
	if obj.ExpiresAt <= 0 {
		return nil, nil
	}
	expiresAt := vegatime.Format(vegatime.UnixNano(obj.ExpiresAt))
	return &expiresAt, nil
}
func (r *myOrderResolver) Trades(ctx context.Context, ord *types.Order) ([]*types.Trade, error) {
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
func (r *myOrderResolver) Party(ctx context.Context, ord *types.Order) (*Party, error) {
	if ord == nil {
		return nil, errors.New("nil order")
	}
	return &Party{
		ID: ord.PartyID,
	}, nil
}

// END: Order Resolver

// BEGIN: Trade Resolver

type myTradeResolver VegaResolverRoot

func (r *myTradeResolver) Market(ctx context.Context, obj *types.Trade) (*Market, error) {
	if obj == nil {
		return nil, errors.New("invalid trade")
	}

	req := protoapi.MarketByIDRequest{MarketID: obj.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return MarketFromProto(res.Market)
}
func (r *myTradeResolver) Aggressor(ctx context.Context, obj *types.Trade) (Side, error) {
	return Side(obj.Aggressor.String()), nil
}
func (r *myTradeResolver) Price(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}
func (r *myTradeResolver) Size(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *myTradeResolver) Timestamp(ctx context.Context, obj *types.Trade) (string, error) {
	return strconv.FormatInt(obj.Timestamp, 10), nil
}
func (r *myTradeResolver) Datetime(ctx context.Context, obj *types.Trade) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}
func (r *myTradeResolver) CreatedAt(ctx context.Context, obj *types.Trade) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}

// END: Trade Resolver

// BEGIN: Candle Resolver

type myCandleResolver VegaResolverRoot

func (r *myCandleResolver) High(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.High, 10), nil
}
func (r *myCandleResolver) Low(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Low, 10), nil
}
func (r *myCandleResolver) Open(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Open, 10), nil
}
func (r *myCandleResolver) Close(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Close, 10), nil
}
func (r *myCandleResolver) Volume(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}
func (r *myCandleResolver) Datetime(ctx context.Context, obj *types.Candle) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}
func (r *myCandleResolver) Timestamp(ctx context.Context, obj *types.Candle) (string, error) {
	return strconv.FormatInt(obj.Timestamp, 10), nil
}
func (r *myCandleResolver) Interval(ctx context.Context, obj *types.Candle) (Interval, error) {
	interval := Interval(obj.Interval.String())
	if interval.IsValid() {
		return interval, nil
	}
	r.log.Warn("Interval conversion from proto to gql type failed, falling back to default: I15M",
		logging.String("interval", interval.String()))
	return IntervalI15m, nil
}

// END: Candle Resolver

// BEGIN: Price Level Resolver

type myPriceLevelResolver VegaResolverRoot

func (r *myPriceLevelResolver) Price(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}

func (r *myPriceLevelResolver) Volume(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Volume, 10), nil
}

func (r *myPriceLevelResolver) NumberOfOrders(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.Price, 10), nil
}

func (r *myPriceLevelResolver) CumulativeVolume(ctx context.Context, obj *types.PriceLevel) (string, error) {
	return strconv.FormatUint(obj.CumulativeVolume, 10), nil
}

// END: Price Level Resolver

// BEGIN: Position Resolver

type myPositionResolver VegaResolverRoot

func (r *myPositionResolver) RealisedVolume(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return strconv.FormatInt(obj.RealisedVolume, 10), nil
}

func (r *myPositionResolver) RealisedProfitValue(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return r.absInt64Str(obj.RealisedPNL), nil
}

func (r *myPositionResolver) RealisedProfitDirection(ctx context.Context, obj *types.MarketPosition) (ValueDirection, error) {
	return r.direction(obj.RealisedPNL), nil
}

func (r *myPositionResolver) UnrealisedVolume(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return strconv.FormatInt(obj.UnrealisedVolume, 10), nil
}

func (r *myPositionResolver) UnrealisedProfitValue(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return r.absInt64Str(obj.UnrealisedPNL), nil
}

func (r *myPositionResolver) UnrealisedProfitDirection(ctx context.Context, obj *types.MarketPosition) (ValueDirection, error) {
	return r.direction(obj.UnrealisedPNL), nil
}

func (r *myPositionResolver) AverageEntryPrice(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return strconv.FormatUint(obj.AverageEntryPrice, 10), nil
}

func (r *myPositionResolver) MinimumMargin(ctx context.Context, obj *types.MarketPosition) (string, error) {
	return strconv.FormatInt(obj.MinimumMargin, 10), nil
}

func (r *myPositionResolver) Market(ctx context.Context, obj *types.MarketPosition) (*Market, error) {
	if obj == nil {
		return nil, errors.New("invalid position")
	}

	req := protoapi.MarketByIDRequest{MarketID: obj.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return MarketFromProto(res.Market)
}

func (r *myPositionResolver) absInt64Str(val int64) string {
	if val < 0 {
		return strconv.FormatInt(val*-1, 10)
	}
	return strconv.FormatInt(val, 10)
}

func (r *myPositionResolver) direction(val int64) ValueDirection {
	if val < 0 {
		return ValueDirectionNegative
	}
	return ValueDirectionPositive
}

// END: Position Resolver

// BEGIN: Mutation Resolver

type myMutationResolver VegaResolverRoot

func (r *myMutationResolver) OrderSubmit(ctx context.Context, market string, party string,
	price *string, size string, side Side, timeInForce OrderTimeInForce, expiration *string,
	ty OrderType) (*types.PendingOrder, error) {

	order := &types.OrderSubmission{}

	tkn := gateway.TokenFromContext(ctx)

	var (
		p   uint64 = 0
		err error
	)

	// We need to convert strings to uint64 (JS doesn't yet support uint64)
	if price != nil {
		p, err = safeStringUint64(*price)
		if err != nil {
			return nil, err
		}
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
	if order.TimeInForce, err = parseOrderTimeInForce(timeInForce); err != nil {
		return nil, err
	}
	if order.Side, err = parseSide(&side); err != nil {
		return nil, err
	}
	if order.Type, err = parseOrderType(ty); err != nil {
		return nil, err
	}

	// GTT must have an expiration value
	if order.TimeInForce == types.Order_GTT && expiration != nil {
		expiresAt, err := vegatime.Parse(*expiration)
		if err != nil {
			return nil, fmt.Errorf("cannot parse expiration time: %s - invalid format sent to create order (example: 2018-01-02T15:04:05Z)", *expiration)
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
		r.log.Error("Failed to create order using rpc client in graphQL resolver", logging.Error(err))
		return nil, err
	}

	return pendingOrder, nil

}

func (r *myMutationResolver) OrderCancel(ctx context.Context, id string, party string, market string) (*types.PendingOrder, error) {
	order := &types.OrderCancellation{}

	tkn := gateway.TokenFromContext(ctx)

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	if len(market) <= 0 {
		return nil, errors.New("market missing or empty")
	}
	order.MarketID = market
	if len(id) == 0 {
		return nil, errors.New("id missing or empty")
	}
	order.OrderID = id
	if len(party) == 0 {
		return nil, errors.New("party missing or empty")
	}

	order.PartyID = party

	// Pass the cancellation over for consensus (service layer will use RPC client internally and handle errors etc)

	req := protoapi.CancelOrderRequest{
		Cancellation: order,
		Token:        tkn,
	}
	pendingOrder, err := r.tradingClient.CancelOrder(ctx, &req)
	if err != nil {
		return nil, err
	}

	return pendingOrder, nil

}

func (r *myMutationResolver) OrderAmend(ctx context.Context, id string, party string, price, size int, expiration *string) (*types.PendingOrder, error) {
	order := &types.OrderAmendment{}

	tkn := gateway.TokenFromContext(ctx)

	// Cancellation currently only requires ID and Market to be set, all other fields will be added
	if len(id) == 0 {
		return nil, errors.New("id missing or empty")
	}
	order.OrderID = id
	if len(party) == 0 {
		return nil, errors.New("party missing or empty")
	}
	order.PartyID = party
	if price < 0 {
		return nil, errors.New("cannot have price less than 0")
	}
	order.Price = uint64(price)
	if size < 0 {
		return nil, errors.New("cannot have size less thean 0")
	}
	order.Size = uint64(size)
	if expiration != nil {
		expiresAt, err := vegatime.Parse(*expiration)
		if err != nil {
			return nil, fmt.Errorf("cannot parse expiration time: %s - invalid format sent to create order (example: 2018-01-02T15:04:05Z)", *expiration)
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
		return nil, err
	}

	return pendingOrder, nil
}

func (r *myMutationResolver) Signin(ctx context.Context, id string, password string) (string, error) {
	req := protoapi.SignInRequest{
		Id:       id,
		Password: password,
	}

	res, err := r.tradingClient.SignIn(ctx, &req)
	if err != nil {
		return "", err
	}

	return res.Token, nil
}

// END: Mutation Resolver

// BEGIN: Subscription Resolver

type mySubscriptionResolver VegaResolverRoot

func (r *mySubscriptionResolver) Accounts(ctx context.Context, marketID *string, partyID *string, asset *string, typeArg *AccountType) (<-chan *proto.Account, error) {
	var (
		mkt, pty string
		ty       types.AccountType
	)

	if marketID != nil {
		mkt = *marketID
	}
	if partyID != nil {
		pty = *partyID
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
		return nil, err
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
				r.log.Error("accounts: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("accounts: stream closed", logging.Error(err))
				break
			}
			c <- a
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Orders(ctx context.Context, market *string, party *string) (<-chan []*types.Order, error) {
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

	c := make(chan []*types.Order)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
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
			c <- o.Orders
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Trades(ctx context.Context, market *string, party *string) (<-chan []*types.Trade, error) {
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

	c := make(chan []*types.Trade)
	go func() {
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			t, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("trades: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("trades: stream closed", logging.Error(err))
				break
			}
			c <- t.Trades
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Positions(ctx context.Context, party string) (<-chan *types.MarketPosition, error) {
	req := &api.PositionsSubscribeRequest{
		PartyID: party,
	}
	stream, err := r.tradingDataClient.PositionsSubscribe(ctx, req)
	if err != nil {
		return nil, err
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
				r.log.Error("positions: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("positions: stream closed", logging.Error(err))
				break
			}
			c <- t
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) MarketDepth(ctx context.Context, market string) (<-chan *types.MarketDepth, error) {
	req := &api.MarketDepthSubscribeRequest{
		MarketID: market,
	}
	stream, err := r.tradingDataClient.MarketDepthSubscribe(ctx, req)
	if err != nil {
		return nil, err
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
				r.log.Error("marketDepth: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("marketDepth: stream closed", logging.Error(err))
				break
			}
			c <- md
		}
	}()

	return c, nil
}

func (r *mySubscriptionResolver) Candles(ctx context.Context, market string, interval Interval) (<-chan *types.Candle, error) {

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
		defer func() {
			stream.CloseSend()
			close(c)
		}()
		for {
			cdl, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("candles: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("candles: stream closed", logging.Error(err))
				break
			}
			c <- cdl
		}
	}()

	return c, nil

}

type myPendingOrderResolver VegaResolverRoot

func (r *myPendingOrderResolver) Type(ctx context.Context, obj *proto.PendingOrder) (*OrderType, error) {
	if obj != nil {
		ot := OrderType(obj.Type.String())
		return &ot, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *myPendingOrderResolver) Price(ctx context.Context, obj *proto.PendingOrder) (*string, error) {
	if obj != nil {
		str := fmt.Sprintf("%v", obj.Price)
		return &str, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *myPendingOrderResolver) TimeInForce(ctx context.Context, obj *proto.PendingOrder) (*OrderTimeInForce, error) {
	if obj != nil {
		ot := OrderTimeInForce(obj.TimeInForce.String())
		return &ot, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *myPendingOrderResolver) Side(ctx context.Context, obj *proto.PendingOrder) (*Side, error) {
	if obj != nil {
		s := Side(obj.Side.String())
		return &s, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *myPendingOrderResolver) Market(ctx context.Context, pord *proto.PendingOrder) (*Market, error) {
	if pord == nil {
		return nil, errors.New("invalid pending order")
	}

	req := protoapi.MarketByIDRequest{MarketID: pord.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, err
	}
	return MarketFromProto(res.Market)
}

func (r *myPendingOrderResolver) Party(ctx context.Context, pord *proto.PendingOrder) (*Party, error) {
	if pord == nil {
		return nil, nil
	}
	return &Party{ID: pord.PartyID}, nil
}

func (r *myPendingOrderResolver) Size(ctx context.Context, obj *proto.PendingOrder) (*string, error) {
	if obj != nil {
		str := fmt.Sprintf("%v", obj.Size)
		return &str, nil
	}
	return nil, ErrNilPendingOrder
}
func (r *myPendingOrderResolver) Status(ctx context.Context, obj *proto.PendingOrder) (*OrderStatus, error) {
	if obj != nil {
		os := OrderStatus(obj.Status.String())
		return &os, nil
	}
	return nil, ErrNilPendingOrder
}

// END: Subscription Resolver

// START: Account Resolver

type myAccountResolver VegaResolverRoot

func (r *myAccountResolver) Balance(ctx context.Context, acc *proto.Account) (string, error) {
	bal := fmt.Sprintf("%d", acc.Balance)
	return bal, nil
}

func (r *myAccountResolver) Market(ctx context.Context, acc *proto.Account) (*Market, error) {
	if acc == nil {
		return nil, errors.New("invalid account")
	}

	// Only margin accounts have a market relation
	if acc.Type == types.AccountType_MARGIN {
		req := protoapi.MarketByIDRequest{MarketID: acc.MarketID}
		res, err := r.tradingDataClient.MarketByID(ctx, &req)
		if err != nil {
			r.log.Error("tradingData client", logging.Error(err))
			return nil, err
		}
		return MarketFromProto(res.Market)
	}

	return nil, nil
}

func (r *myAccountResolver) Type(ctx context.Context, obj *proto.Account) (AccountType, error) {
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

type myStatisticsResolver VegaResolverRoot

func (r *myStatisticsResolver) BlockHeight(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.BlockHeight), nil
}

func (r *myStatisticsResolver) BacklogLength(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.BacklogLength), nil
}

func (r *myStatisticsResolver) TotalPeers(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalPeers), nil
}

func (r *myStatisticsResolver) Status(ctx context.Context, obj *proto.Statistics) (string, error) {
	return obj.Status.String(), nil
}

func (r *myStatisticsResolver) TxPerBlock(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TxPerBlock), nil
}

func (r *myStatisticsResolver) AverageTxBytes(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.AverageTxBytes), nil
}

func (r *myStatisticsResolver) AverageOrdersPerBlock(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.AverageOrdersPerBlock), nil
}

func (r *myStatisticsResolver) TradesPerSecond(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TradesPerSecond), nil
}

func (r *myStatisticsResolver) OrdersPerSecond(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.OrdersPerSecond), nil
}

func (r *myStatisticsResolver) TotalMarkets(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalMarkets), nil
}

func (r *myStatisticsResolver) TotalParties(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalParties), nil
}

func (r *myStatisticsResolver) TotalAmendOrder(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalAmendOrder), nil
}

func (r *myStatisticsResolver) TotalCancelOrder(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalCancelOrder), nil
}

func (r *myStatisticsResolver) TotalCreateOrder(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalCreateOrder), nil
}

func (r *myStatisticsResolver) TotalOrders(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalOrders), nil
}

func (r *myStatisticsResolver) TotalTrades(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.TotalTrades), nil
}

func (r *myStatisticsResolver) BlockDuration(ctx context.Context, obj *proto.Statistics) (int, error) {
	return int(obj.BlockDuration), nil
}
