package gql

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"code.vegaprotocol.io/vega/gateway"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/vegatime"
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
	// prepare calls (unary-like calls)
	PrepareSubmitOrder(ctx context.Context, in *protoapi.SubmitOrderRequest, opts ...grpc.CallOption) (*protoapi.PrepareSubmitOrderResponse, error)
	PrepareAmendOrder(ctx context.Context, in *protoapi.AmendOrderRequest, opts ...grpc.CallOption) (*protoapi.PrepareAmendOrderResponse, error)
	PrepareCancelOrder(ctx context.Context, in *protoapi.CancelOrderRequest, opts ...grpc.CallOption) (*protoapi.PrepareCancelOrderResponse, error)
	// unary calls - writes
	SubmitTransaction(ctx context.Context, in *protoapi.SubmitTransactionRequest, opts ...grpc.CallOption) (*protoapi.SubmitTransactionResponse, error)
	// old requests
	SubmitOrder(ctx context.Context, in *protoapi.SubmitOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, in *protoapi.CancelOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, in *protoapi.AmendOrderRequest, opts ...grpc.CallOption) (*types.PendingOrder, error)
	SignIn(ctx context.Context, in *protoapi.SignInRequest, opts ...grpc.CallOption) (*protoapi.SignInResponse, error)
	// unary calls - reads
	CheckToken(context.Context, *protoapi.CheckTokenRequest, ...grpc.CallOption) (*protoapi.CheckTokenResponse, error)
}

// TradingDataClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trading_data_client_mock.go -package mocks code.vegaprotocol.io/vega/gateway/graphql TradingDataClient
type TradingDataClient interface {
	// orders
	OrdersByMarket(ctx context.Context, in *protoapi.OrdersByMarketRequest, opts ...grpc.CallOption) (*protoapi.OrdersByMarketResponse, error)
	OrderByReference(ctx context.Context, in *protoapi.OrderByReferenceRequest, opts ...grpc.CallOption) (*protoapi.OrderByReferenceResponse, error)
	OrdersByParty(ctx context.Context, in *protoapi.OrdersByPartyRequest, opts ...grpc.CallOption) (*protoapi.OrdersByPartyResponse, error)
	OrderByMarketAndID(ctx context.Context, in *protoapi.OrderByMarketAndIdRequest, opts ...grpc.CallOption) (*protoapi.OrderByMarketAndIdResponse, error)
	// markets
	MarketByID(ctx context.Context, in *protoapi.MarketByIDRequest, opts ...grpc.CallOption) (*protoapi.MarketByIDResponse, error)
	Markets(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*protoapi.MarketsResponse, error)
	MarketDepth(ctx context.Context, in *protoapi.MarketDepthRequest, opts ...grpc.CallOption) (*protoapi.MarketDepthResponse, error)
	LastTrade(ctx context.Context, in *protoapi.LastTradeRequest, opts ...grpc.CallOption) (*protoapi.LastTradeResponse, error)
	MarketDataByID(ctx context.Context, in *protoapi.MarketDataByIDRequest, opts ...grpc.CallOption) (*protoapi.MarketDataByIDResponse, error)
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
	MarketsDataSubscribe(ctx context.Context, in *protoapi.MarketsDataSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_MarketsDataSubscribeClient, error)
	MarginLevelsSubscribe(ctx context.Context, in *protoapi.MarginLevelsSubscribeRequest, opts ...grpc.CallOption) (protoapi.TradingData_MarginLevelsSubscribeClient, error)
	// accounts
	PartyAccounts(ctx context.Context, req *protoapi.PartyAccountsRequest, opts ...grpc.CallOption) (*protoapi.PartyAccountsResponse, error)
	MarketAccounts(ctx context.Context, req *protoapi.MarketAccountsRequest, opts ...grpc.CallOption) (*protoapi.MarketAccountsResponse, error)
	// margins
	MarginLevels(ctx context.Context, in *protoapi.MarginLevelsRequest, opts ...grpc.CallOption) (*protoapi.MarginLevelsResponse, error)
}

// VegaResolverRoot is the root resolver for all graphql types
type VegaResolverRoot struct {
	gateway.Config

	log               *logging.Logger
	tradingClient     TradingClient
	tradingDataClient TradingDataClient
}

// NewResolverRoot instantiate a graphql root resolver
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

// MarketData returns the market data resolver
func (r *VegaResolverRoot) MarketData() MarketDataResolver {
	return (*myMarketDataResolver)(r)
}

// MarginLevels returns the market levels resolver
func (r *VegaResolverRoot) MarginLevels() MarginLevelsResolver {
	return (*myMarginLevelsResolver)(r)
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
		return nil, customErrorFromStatus(err)
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
		return nil, customErrorFromStatus(err)
	}
	market, err := MarketFromProto(res.Market)
	if err != nil {
		r.log.Error("unable to convert market from proto", logging.Error(err))
		return nil, err
	}
	return market, nil
}

func (r *myQueryResolver) Parties(ctx context.Context, name *string) ([]*types.Party, error) {
	if name == nil {
		return nil, errors.New("all parties not implemented")
	}
	party, err := r.Party(ctx, *name)
	if err != nil {
		return nil, err
	}
	return []*types.Party{party}, nil
}

func (r *myQueryResolver) Party(ctx context.Context, name string) (*types.Party, error) {
	if len(name) == 0 {
		return nil, errors.New("invalid party")
	}
	req := protoapi.PartyByIDRequest{PartyID: name}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
}

func (r *myQueryResolver) Statistics(ctx context.Context) (*types.Statistics, error) {
	res, err := r.tradingDataClient.Statistics(ctx, &empty.Empty{})
	if err != nil {
		r.log.Error("tradingCore client", logging.Error(err))
		return nil, customErrorFromStatus(err)
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
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return &CheckTokenResponse{Ok: response.Ok}, nil
}

// END: Root Resolver

// BEGIN: Market Resolver

type myMarketResolver VegaResolverRoot

func (r *myMarketResolver) Data(ctx context.Context, market *Market) (*types.MarketData, error) {
	req := protoapi.MarketDataByIDRequest{
		MarketID: market.ID,
	}
	res, err := r.tradingDataClient.MarketDataByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.MarketData, nil
}

func (r *myMarketResolver) Orders(ctx context.Context, market *Market,
	open *bool, skip *int, first *int, last *int) ([]*types.Order, error) {
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
		return nil, customErrorFromStatus(err)
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
		return nil, customErrorFromStatus(err)
	}

	return res.Trades, nil
}

func (r *myMarketResolver) Depth(ctx context.Context, market *Market, maxDepth *int) (*types.MarketDepth, error) {

	if market == nil {
		return nil, errors.New("market missing or empty")
	}

	req := protoapi.MarketDepthRequest{MarketID: market.ID}
	if maxDepth != nil {
		if *maxDepth <= 0 {
			return nil, errors.New("invalid maxDepth, must be a positive number")
		}
		req.MaxDepth = uint64(*maxDepth)
	}

	// Look for market depth for the given market (will validate market internally)
	// Note: Market depth is also known as OrderBook depth within the matching-engine
	res, err := r.tradingDataClient.MarketDepth(ctx, &req)
	if err != nil {
		r.log.Error("trading data client", logging.Error(err))
		return nil, customErrorFromStatus(err)
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
		return nil, customErrorFromStatus(err)
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
		return nil, customErrorFromStatus(err)
	}
	return res.Order, nil
}

// Accounts ...
// if partyID specified get margin account for the given market
// if nil return the insurance pool for the market
func (r *myMarketResolver) Accounts(ctx context.Context, market *Market, partyID *string) ([]*types.Account, error) {
	// get margin account for a party
	if partyID != nil {
		req := protoapi.PartyAccountsRequest{
			PartyID:  *partyID,
			MarketID: market.ID,
			Type:     types.AccountType_MARGIN,
			Asset:    "",
		}
		res, err := r.tradingDataClient.PartyAccounts(ctx, &req)
		if err != nil {
			r.log.Error("unable to get PartyAccounts",
				logging.Error(err),
				logging.String("market-id", market.ID),
				logging.String("party-id", *partyID))
			return []*types.Account{}, customErrorFromStatus(err)
		}
		return res.Accounts, nil
	}
	// get accounts for the market
	req := protoapi.MarketAccountsRequest{
		MarketID: market.ID,
		Asset:    "", // all assets
	}
	res, err := r.tradingDataClient.MarketAccounts(ctx, &req)
	if err != nil {
		r.log.Error("unable to get MarketAccounts",
			logging.Error(err),
			logging.String("market-id", market.ID))
		return []*types.Account{}, customErrorFromStatus(err)
	}
	return res.Accounts, nil
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

func (r *myPartyResolver) Margins(ctx context.Context,
	party *types.Party, marketID *string) ([]*types.MarginLevels, error) {

	var marketId string
	if marketID != nil {
		marketId = *marketID
	}
	req := protoapi.MarginLevelsRequest{
		PartyID:  party.Id,
		MarketID: marketId,
	}
	res, err := r.tradingDataClient.MarginLevels(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	out := make([]*types.MarginLevels, 0, len(res.MarginLevels))
	out = append(out, res.MarginLevels...)
	return out, nil
}

func (r *myPartyResolver) Orders(ctx context.Context, party *types.Party,
	open *bool, skip *int, first *int, last *int) ([]*types.Order, error) {

	p := makePagination(skip, first, last)
	openOnly := open != nil && *open
	req := protoapi.OrdersByPartyRequest{
		PartyID:    party.Id,
		Open:       openOnly,
		Pagination: p,
	}
	res, err := r.tradingDataClient.OrdersByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	if len(res.Orders) > 0 {
		return res.Orders, nil
	} else {
		// mandatory return field in schema
		return []*types.Order{}, nil
	}
}

func (r *myPartyResolver) Trades(ctx context.Context, party *types.Party,
	market *string, skip *int, first *int, last *int) ([]*types.Trade, error) {

	var mkt string
	if market != nil {
		mkt = *market
	}

	p := makePagination(skip, first, last)
	req := protoapi.TradesByPartyRequest{
		PartyID:    party.Id,
		MarketID:   mkt,
		Pagination: p,
	}

	res, err := r.tradingDataClient.TradesByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	if len(res.Trades) > 0 {
		return res.Trades, nil
	} else {
		// mandatory return field in schema
		return []*types.Trade{}, nil
	}
}

func (r *myPartyResolver) Positions(ctx context.Context, party *types.Party) ([]*types.Position, error) {
	if party == nil {
		return nil, errors.New("nil party")
	}
	req := protoapi.PositionsByPartyRequest{PartyID: party.Id}
	res, err := r.tradingDataClient.PositionsByParty(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	if len(res.Positions) > 0 {
		return res.Positions, nil
	} else {
		// mandatory return field in schema
		return []*types.Position{}, nil
	}
}

func AccountTypeToProto(acc AccountType) (types.AccountType, error) {
	switch acc {
	case AccountTypeGeneral:
		return types.AccountType_GENERAL, nil
	case AccountTypeMargin:
		return types.AccountType_MARGIN, nil
	case AccountTypeInsurance:
		return types.AccountType_INSURANCE, nil
	case AccountTypeSettlement:
		return types.AccountType_SETTLEMENT, nil
	default:
		return types.AccountType_ALL, fmt.Errorf("invalid account type %v, return default (ALL)", acc)
	}
}

func (r *myPartyResolver) Accounts(ctx context.Context, party *types.Party,
	marketID *string, asset *string, accType *AccountType) ([]*types.Account, error) {
	if party == nil {
		return nil, errors.New("a party must be specified when querying accounts")
	}
	var (
		mktid = ""
		asst  = ""
		accTy = types.AccountType_ALL
		err   error
	)

	if marketID != nil {
		mktid = *marketID
	}
	if asset != nil {
		asst = *asset
	}
	if accType != nil {
		accTy, err = AccountTypeToProto(*accType)
		if err != nil || (accTy != types.AccountType_GENERAL && accTy != types.AccountType_MARGIN) {
			return nil, fmt.Errorf("invalid account type for party %v", accType)
		}
	}
	req := protoapi.PartyAccountsRequest{
		PartyID:  party.Id,
		MarketID: mktid,
		Asset:    asst,
		Type:     accTy,
	}
	res, err := r.tradingDataClient.PartyAccounts(ctx, &req)
	if err != nil {
		r.log.Error("unable to get Party account",
			logging.Error(err),
			logging.String("party-id", party.Id),
			logging.String("market-id", mktid),
			logging.String("asset", asst),
			logging.String("type", accTy.String()))
		return nil, customErrorFromStatus(err)
	}

	if len(res.Accounts) > 0 {
		return res.Accounts, nil
	} else {
		// mandatory return field in schema
		return []*types.Account{}, nil
	}
}

// END: Party Resolver

// BEGIN: MarginLevels Resolver

type myMarginLevelsResolver VegaResolverRoot

func (r *myMarginLevelsResolver) Market(ctx context.Context, m *types.MarginLevels) (*Market, error) {
	req := protoapi.MarketByIDRequest{MarketID: m.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	market, err := MarketFromProto(res.Market)
	if err != nil {
		r.log.Error("unable to convert market from proto", logging.Error(err))
		return nil, err
	}
	return market, nil
}

func (r *myMarginLevelsResolver) Party(ctx context.Context, m *types.MarginLevels) (*types.Party, error) {
	if m == nil {
		return nil, errors.New("nil order")
	}
	if len(m.PartyID) == 0 {
		return nil, errors.New("invalid party")
	}
	req := protoapi.PartyByIDRequest{PartyID: m.PartyID}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
}

func (r *myMarginLevelsResolver) Asset(_ context.Context, m *types.MarginLevels) (string, error) {
	return m.Asset, nil
}

func (r *myMarginLevelsResolver) CollateralReleaseLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return strconv.FormatInt(m.CollateralReleaseLevel, 10), nil
}

func (r *myMarginLevelsResolver) InitialLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return strconv.FormatInt(m.InitialMargin, 10), nil
}

func (r *myMarginLevelsResolver) SearchLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return strconv.FormatInt(m.SearchLevel, 10), nil
}

func (r *myMarginLevelsResolver) MaintenanceLevel(_ context.Context, m *types.MarginLevels) (string, error) {
	return strconv.FormatInt(m.MaintenanceMargin, 10), nil
}

func (r *myMarginLevelsResolver) Timestamp(_ context.Context, m *types.MarginLevels) (string, error) {
	return vegatime.Format(vegatime.UnixNano(m.Timestamp)), nil
}

// END: MarginLevels Resolver

// BEGIN: MarketData resolver

type myMarketDataResolver VegaResolverRoot

func (r *myMarketDataResolver) BestBidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestBidPrice, 10), nil
}

func (r *myMarketDataResolver) BestBidVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestBidVolume, 10), nil
}

func (r *myMarketDataResolver) BestOfferPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestOfferPrice, 10), nil
}

func (r *myMarketDataResolver) BestOfferVolume(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.BestOfferVolume, 10), nil
}

func (r *myMarketDataResolver) MidPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.MidPrice, 10), nil
}

func (r *myMarketDataResolver) MarkPrice(_ context.Context, m *types.MarketData) (string, error) {
	return strconv.FormatUint(m.MarkPrice, 10), nil
}

func (r *myMarketDataResolver) Timestamp(_ context.Context, m *types.MarketData) (string, error) {
	return vegatime.Format(vegatime.UnixNano(m.Timestamp)), nil
}

func (r *myMarketDataResolver) Market(ctx context.Context, m *types.MarketData) (*Market, error) {
	req := protoapi.MarketByIDRequest{MarketID: m.Market}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	market, err := MarketFromProto(res.Market)
	if err != nil {
		r.log.Error("unable to convert market from proto", logging.Error(err))
		return nil, err
	}
	return market, nil
}

// END: MarketData resolver

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
		return nil, customErrorFromStatus(err)
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
		return nil, customErrorFromStatus(err)
	}
	return MarketFromProto(res.Market)
}

// END: Market Depth Resolver

// BEGIN: Order Resolver

type myOrderResolver VegaResolverRoot

func RejectionReasonFromProtoOrderError(o types.OrderError) (RejectionReason, error) {
	switch o {
	case types.OrderError_INVALID_MARKET_ID:
		return RejectionReasonInvalidMarketID, nil
	case types.OrderError_INVALID_ORDER_ID:
		return RejectionReasonInvalidOrderID, nil
	case types.OrderError_ORDER_OUT_OF_SEQUENCE:
		return RejectionReasonOrderOutOfSequence, nil
	case types.OrderError_INVALID_REMAINING_SIZE:
		return RejectionReasonInvalidRemainingSize, nil
	case types.OrderError_TIME_FAILURE:
		return RejectionReasonTimeFailure, nil
	case types.OrderError_ORDER_REMOVAL_FAILURE:
		return RejectionReasonOrderRemovalFailure, nil
	case types.OrderError_INVALID_EXPIRATION_DATETIME:
		return RejectionReasonInvalidExpirationTime, nil
	case types.OrderError_INVALID_ORDER_REFERENCE:
		return RejectionReasonInvalidOrderReference, nil
	case types.OrderError_EDIT_NOT_ALLOWED:
		return RejectionReasonEditNotAllowed, nil
	case types.OrderError_ORDER_AMEND_FAILURE:
		return RejectionReasonOrderAmendFailure, nil
	case types.OrderError_ORDER_NOT_FOUND:
		return RejectionReasonOrderNotFound, nil
	case types.OrderError_INVALID_PARTY_ID:
		return RejectionReasonInvalidPartyID, nil
	case types.OrderError_MARKET_CLOSED:
		return RejectionReasonMarketClosed, nil
	case types.OrderError_MARGIN_CHECK_FAILED:
		return RejectionReasonMarginCheckFailed, nil
	case types.OrderError_INTERNAL_ERROR:
		return RejectionReasonInternalError, nil
	default:
		return "", fmt.Errorf("invalid RejectionReason: %v", o)
	}
}

func (r *myOrderResolver) RejectionReason(_ context.Context, o *types.Order) (*RejectionReason, error) {
	if o.Reason == types.OrderError_NONE {
		return nil, nil
	}
	reason, err := RejectionReasonFromProtoOrderError(o.Reason)
	if err != nil {
		return nil, err
	}
	return &reason, nil
}

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
		return nil, customErrorFromStatus(err)
	}
	return MarketFromProto(res.Market)
}
func (r *myOrderResolver) Size(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Size, 10), nil
}
func (r *myOrderResolver) Remaining(ctx context.Context, obj *types.Order) (string, error) {
	return strconv.FormatUint(obj.Remaining, 10), nil
}
func (r *myOrderResolver) Status(ctx context.Context, obj *types.Order) (OrderStatus, error) {
	return OrderStatus(obj.Status.String()), nil
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
		return nil, customErrorFromStatus(err)
	}
	return res.Trades, nil
}
func (r *myOrderResolver) Party(ctx context.Context, order *types.Order) (*types.Party, error) {
	if order == nil {
		return nil, errors.New("nil order")
	}
	if len(order.PartyID) == 0 {
		return nil, errors.New("invalid party")
	}
	req := protoapi.PartyByIDRequest{PartyID: order.PartyID}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
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
		return nil, customErrorFromStatus(err)
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
func (r *myTradeResolver) CreatedAt(ctx context.Context, obj *types.Trade) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}
func (r *myTradeResolver) Buyer(ctx context.Context, obj *types.Trade) (*types.Party, error) {
	if obj == nil {
		return nil, errors.New("invalid trade")
	}
	if len(obj.Buyer) == 0 {
		return nil, errors.New("invalid buyer")
	}
	req := protoapi.PartyByIDRequest{PartyID: obj.Buyer}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
}
func (r *myTradeResolver) Seller(ctx context.Context, obj *types.Trade) (*types.Party, error) {
	if obj == nil {
		return nil, errors.New("invalid trade")
	}
	if len(obj.Seller) == 0 {
		return nil, errors.New("invalid seller")
	}
	req := protoapi.PartyByIDRequest{PartyID: obj.Seller}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
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

func (r *myPositionResolver) Market(ctx context.Context, obj *types.Position) (*Market, error) {
	if obj == nil {
		return nil, errors.New("invalid position")
	}
	if len(obj.MarketID) <= 0 {
		return nil, nil
	}
	req := protoapi.MarketByIDRequest{MarketID: obj.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return MarketFromProto(res.Market)
}

func (r *myPositionResolver) OpenVolume(ctx context.Context, obj *types.Position) (string, error) {
	return strconv.FormatInt(obj.OpenVolume, 10), nil
}

func (r *myPositionResolver) RealisedPnl(ctx context.Context, obj *types.Position) (string, error) {
	return strconv.FormatInt(obj.RealisedPNL, 10), nil
}

func (r *myPositionResolver) UnrealisedPnl(ctx context.Context, obj *types.Position) (string, error) {
	return strconv.FormatInt(obj.UnrealisedPNL, 10), nil
}

func (r *myPositionResolver) AverageEntryPrice(ctx context.Context, obj *types.Position) (string, error) {
	return strconv.FormatUint(obj.AverageEntryPrice, 10), nil
}

func (r *myPositionResolver) Margins(ctx context.Context, obj *types.Position) ([]*types.MarginLevels, error) {
	if obj == nil {
		return nil, errors.New("invalid position")
	}
	if len(obj.PartyID) <= 0 {
		return nil, errors.New("missing party id")
	}
	req := protoapi.MarginLevelsRequest{
		PartyID:  obj.PartyID,
		MarketID: obj.MarketID,
	}
	res, err := r.tradingDataClient.MarginLevels(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	out := make([]*types.MarginLevels, 0, len(res.MarginLevels))
	out = append(out, res.MarginLevels...)
	return out, nil
}

// END: Position Resolver

// BEGIN: Mutation Resolver

type myMutationResolver VegaResolverRoot

func (r *myMutationResolver) SubmitTransaction(ctx context.Context, data, sig string, address, pubkey *string) (*TransactionSubmitted, error) {
	if address == nil && pubkey == nil {
		return nil, errors.New("auth missing: either address or pubkey needs to be set")
	}
	decodedData, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	decodedSig, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return nil, err
	}
	req := &protoapi.SubmitTransactionRequest{
		Tx: &types.SignedBundle{
			Data: decodedData,
			Sig:  decodedSig,
		},
	}
	if pubkey != nil {
		pk, err := hex.DecodeString(*pubkey)
		if err != nil {
			return nil, err
		}
		req.Tx.Auth = &types.SignedBundle_PubKey{
			PubKey: pk,
		}
	} else {
		addr, err := hex.DecodeString(*address)
		if err != nil {
			return nil, err
		}
		// address is guaranteed to be set here...
		req.Tx.Auth = &types.SignedBundle_Address{
			Address: addr,
		}
	}
	res, err := r.tradingClient.SubmitTransaction(ctx, req)
	if err != nil {
		r.log.Error("Failed to submit transaction", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}

	return &TransactionSubmitted{
		Success: res.Success,
	}, nil
}

func (r *myMutationResolver) PrepareOrderSubmit(ctx context.Context, market, party string, price *string, size string, side Side, timeInForce OrderTimeInForce, expiration *string, ty OrderType) (*PreparedSubmitOrder, error) {

	order := &types.OrderSubmission{}

	tkn := gateway.TokenFromContext(ctx)

	var (
		p   uint64
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
		var expiresAt time.Time
		expiresAt, err = vegatime.Parse(*expiration)
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
	resp, err := r.tradingClient.PrepareSubmitOrder(ctx, &req)
	if err != nil {
		r.log.Error("Failed to create order using rpc client in graphQL resolver", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return &PreparedSubmitOrder{
		Blob:         base64.StdEncoding.EncodeToString(resp.Blob),
		PendingOrder: resp.PendingOrder,
	}, nil
}

func (r *myMutationResolver) OrderSubmit(ctx context.Context, market string, party string,
	price *string, size string, side Side, timeInForce OrderTimeInForce, expiration *string,
	ty OrderType) (*types.PendingOrder, error) {

	order := &types.OrderSubmission{}

	tkn := gateway.TokenFromContext(ctx)

	var (
		p   uint64
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
		var expiresAt time.Time
		expiresAt, err = vegatime.Parse(*expiration)
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
		return nil, customErrorFromStatus(err)
	}

	return pendingOrder, nil

}

func (r *myMutationResolver) PrepareOrderCancel(ctx context.Context, id string, party string, market string) (*PreparedCancelOrder, error) {
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
	pendingOrder, err := r.tradingClient.PrepareCancelOrder(ctx, &req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}
	return &PreparedCancelOrder{
		Blob:         base64.StdEncoding.EncodeToString(pendingOrder.Blob),
		PendingOrder: pendingOrder.PendingOrder,
	}, nil

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
		return nil, customErrorFromStatus(err)
	}

	return pendingOrder, nil

}

func (r *myMutationResolver) PrepareOrderAmend(ctx context.Context, id string, party string, price, size string, expiration *string) (*PreparedAmendOrder, error) {
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

	var err error
	order.Price, err = strconv.ParseUint(price, 10, 64)
	if err != nil {
		r.log.Error("unable to convert price from string in order amend",
			logging.Error(err))
		return nil, errors.New("invalid price, could not convert to unsigned int")
	}

	order.Size, err = strconv.ParseUint(size, 10, 64)
	if err != nil {
		r.log.Error("unable to convert size from string in order amend",
			logging.Error(err))
		return nil, errors.New("invalid size, could not convert to unsigned int")
	}

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
	pendingOrder, err := r.tradingClient.PrepareAmendOrder(ctx, &req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}
	return &PreparedAmendOrder{
		Blob:         base64.StdEncoding.EncodeToString(pendingOrder.Blob),
		PendingOrder: pendingOrder.PendingOrder,
	}, nil
}
func (r *myMutationResolver) OrderAmend(ctx context.Context, id string, party string, price, size string, expiration *string) (*types.PendingOrder, error) {
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

	var err error
	order.Price, err = strconv.ParseUint(price, 10, 64)
	if err != nil {
		r.log.Error("unable to convert price from string in order amend",
			logging.Error(err))
		return nil, errors.New("invalid price, could not convert to unsigned int")
	}

	order.Size, err = strconv.ParseUint(size, 10, 64)
	if err != nil {
		r.log.Error("unable to convert size from string in order amend",
			logging.Error(err))
		return nil, errors.New("invalid size, could not convert to unsigned int")
	}

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
		return nil, customErrorFromStatus(err)
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

func (r *mySubscriptionResolver) Margins(ctx context.Context, partyID string, marketID *string) (<-chan *types.MarginLevels, error) {
	var mktid string
	if marketID != nil {
		mktid = *marketID
	}
	req := &protoapi.MarginLevelsSubscribeRequest{
		MarketID: mktid,
		PartyID:  partyID,
	}
	stream, err := r.tradingDataClient.MarginLevelsSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	ch := make(chan *types.MarginLevels)
	go func() {
		defer func() {
			stream.CloseSend()
			close(ch)
		}()
		for {
			m, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("margin levels: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("margin levls: stream closed", logging.Error(err))
				break
			}
			ch <- m
		}
	}()

	return ch, nil
}

func (r *mySubscriptionResolver) MarketData(ctx context.Context, marketID *string) (<-chan *types.MarketData, error) {
	var mktid string
	if marketID != nil {
		mktid = *marketID
	}
	req := &protoapi.MarketsDataSubscribeRequest{
		MarketID: mktid,
	}
	stream, err := r.tradingDataClient.MarketsDataSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	ch := make(chan *types.MarketData)
	go func() {
		defer func() {
			stream.CloseSend()
			close(ch)
		}()
		for {
			m, err := stream.Recv()
			if err == io.EOF {
				r.log.Error("marketdata: stream closed by server", logging.Error(err))
				break
			}
			if err != nil {
				r.log.Error("marketdata: stream closed", logging.Error(err))
				break
			}
			ch <- m
		}
	}()

	return ch, nil
}

func (r *mySubscriptionResolver) Accounts(ctx context.Context, marketID *string, partyID *string, asset *string, typeArg *AccountType) (<-chan *types.Account, error) {
	var (
		mkt, pty string
		ty       types.AccountType
	)

	if marketID == nil && partyID == nil && asset == nil && typeArg == nil {
		// Updates on every balance update, on every account, for everyone and shouldn't be allowed for GraphQL.
		return nil, errors.New("at least one query filter must be applied for this subscription")
	}
	if marketID != nil {
		mkt = *marketID
	}
	if partyID != nil {
		pty = *partyID
	}
	if typeArg != nil {
		ty = typeArg.IntoProto()
	}

	req := &protoapi.AccountsSubscribeRequest{
		MarketID: mkt,
		PartyID:  pty,
		Type:     ty,
	}
	stream, err := r.tradingDataClient.AccountsSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
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

	req := &protoapi.OrdersSubscribeRequest{
		MarketID: mkt,
		PartyID:  pty,
	}
	stream, err := r.tradingDataClient.OrdersSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
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

	req := &protoapi.TradesSubscribeRequest{
		MarketID: mkt,
		PartyID:  pty,
	}
	stream, err := r.tradingDataClient.TradesSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
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

func (r *mySubscriptionResolver) Positions(ctx context.Context, party string) (<-chan *types.Position, error) {
	req := &protoapi.PositionsSubscribeRequest{
		PartyID: party,
	}
	stream, err := r.tradingDataClient.PositionsSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
	}

	c := make(chan *types.Position)
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
	req := &protoapi.MarketDepthSubscribeRequest{
		MarketID: market,
	}
	stream, err := r.tradingDataClient.MarketDepthSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
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

	req := &protoapi.CandlesSubscribeRequest{
		MarketID: market,
		Interval: pinterval,
	}
	stream, err := r.tradingDataClient.CandlesSubscribe(ctx, req)
	if err != nil {
		return nil, customErrorFromStatus(err)
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

func (r *myPendingOrderResolver) Type(ctx context.Context, obj *types.PendingOrder) (*OrderType, error) {
	if obj != nil {
		ot := OrderType(obj.Type.String())
		return &ot, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *myPendingOrderResolver) Price(ctx context.Context, obj *types.PendingOrder) (*string, error) {
	if obj != nil {
		str := fmt.Sprintf("%v", obj.Price)
		return &str, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *myPendingOrderResolver) TimeInForce(ctx context.Context, obj *types.PendingOrder) (*OrderTimeInForce, error) {
	if obj != nil {
		ot := OrderTimeInForce(obj.TimeInForce.String())
		return &ot, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *myPendingOrderResolver) Side(ctx context.Context, obj *types.PendingOrder) (*Side, error) {
	if obj != nil {
		s := Side(obj.Side.String())
		return &s, nil
	}
	return nil, ErrNilPendingOrder
}

func (r *myPendingOrderResolver) Market(ctx context.Context, pord *types.PendingOrder) (*Market, error) {
	if pord == nil {
		return nil, errors.New("invalid pending order")
	}

	req := protoapi.MarketByIDRequest{MarketID: pord.MarketID}
	res, err := r.tradingDataClient.MarketByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return MarketFromProto(res.Market)
}

func (r *myPendingOrderResolver) Party(ctx context.Context, pendingOrder *types.PendingOrder) (*types.Party, error) {
	if pendingOrder == nil {
		return nil, nil
	}
	if len(pendingOrder.PartyID) == 0 {
		return nil, errors.New("invalid party")
	}
	req := protoapi.PartyByIDRequest{PartyID: pendingOrder.PartyID}
	res, err := r.tradingDataClient.PartyByID(ctx, &req)
	if err != nil {
		r.log.Error("tradingData client", logging.Error(err))
		return nil, customErrorFromStatus(err)
	}
	return res.Party, nil
}

func (r *myPendingOrderResolver) Size(ctx context.Context, obj *types.PendingOrder) (*string, error) {
	if obj != nil {
		str := fmt.Sprintf("%v", obj.Size)
		return &str, nil
	}
	return nil, ErrNilPendingOrder
}
func (r *myPendingOrderResolver) Status(ctx context.Context, obj *types.PendingOrder) (*OrderStatus, error) {
	if obj != nil {
		os := OrderStatus(obj.Status.String())
		return &os, nil
	}
	return nil, ErrNilPendingOrder
}

// END: Subscription Resolver

// START: Account Resolver

type myAccountResolver VegaResolverRoot

func (r *myAccountResolver) Balance(ctx context.Context, acc *types.Account) (string, error) {
	bal := fmt.Sprintf("%d", acc.Balance)
	return bal, nil
}

func (r *myAccountResolver) Market(ctx context.Context, acc *types.Account) (*Market, error) {
	if acc == nil {
		return nil, errors.New("invalid account")
	}

	// Only margin accounts have a market relation
	if acc.Type == types.AccountType_MARGIN {
		req := protoapi.MarketByIDRequest{MarketID: acc.MarketID}
		res, err := r.tradingDataClient.MarketByID(ctx, &req)
		if err != nil {
			r.log.Error("tradingData client", logging.Error(err))
			return nil, customErrorFromStatus(err)
		}
		return MarketFromProto(res.Market)
	}

	return nil, nil
}

func (r *myAccountResolver) Type(ctx context.Context, obj *types.Account) (AccountType, error) {
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

func (r *myStatisticsResolver) BlockHeight(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.BlockHeight), nil
}

func (r *myStatisticsResolver) BacklogLength(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.BacklogLength), nil
}

func (r *myStatisticsResolver) TotalPeers(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalPeers), nil
}

func (r *myStatisticsResolver) Status(ctx context.Context, obj *types.Statistics) (string, error) {
	return obj.Status.String(), nil
}

func (r *myStatisticsResolver) TxPerBlock(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TxPerBlock), nil
}

func (r *myStatisticsResolver) AverageTxBytes(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.AverageTxBytes), nil
}

func (r *myStatisticsResolver) AverageOrdersPerBlock(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.AverageOrdersPerBlock), nil
}

func (r *myStatisticsResolver) TradesPerSecond(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TradesPerSecond), nil
}

func (r *myStatisticsResolver) OrdersPerSecond(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.OrdersPerSecond), nil
}

func (r *myStatisticsResolver) TotalMarkets(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalMarkets), nil
}

func (r *myStatisticsResolver) TotalParties(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalParties), nil
}

func (r *myStatisticsResolver) TotalAmendOrder(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalAmendOrder), nil
}

func (r *myStatisticsResolver) TotalCancelOrder(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalCancelOrder), nil
}

func (r *myStatisticsResolver) TotalCreateOrder(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalCreateOrder), nil
}

func (r *myStatisticsResolver) TotalOrders(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalOrders), nil
}

func (r *myStatisticsResolver) TotalTrades(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.TotalTrades), nil
}

func (r *myStatisticsResolver) BlockDuration(ctx context.Context, obj *types.Statistics) (int, error) {
	return int(obj.BlockDuration), nil
}
