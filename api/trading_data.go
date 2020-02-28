package api

import (
	"context"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/ptypes/empty"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
)

var defaultPagination = protoapi.Pagination{
	Skip:       0,
	Limit:      50,
	Descending: true,
}

// VegaTime ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/vega_time_mock.go -package mocks code.vegaprotocol.io/vega/api VegaTime
type VegaTime interface {
	GetTimeNow() (time.Time, error)
}

// OrderService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_service_mock.go -package mocks code.vegaprotocol.io/vega/api OrderService
type OrderService interface {
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByMarketAndID(ctx context.Context, market string, id string) (order *types.Order, err error)
	GetByReference(ctx context.Context, ref string) (order *types.Order, err error)
	ObserveOrders(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Order, ref uint64)
	GetOrderSubscribersCount() int32
}

// TradeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_service_mock.go -package mocks code.vegaprotocol.io/vega/api TradeService
type TradeService interface {
	GetByOrderID(ctx context.Context, orderID string) ([]*types.Trade, error)
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (trades []*types.Trade, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, marketID *string) (trades []*types.Trade, err error)
	GetPositionsByParty(ctx context.Context, party, marketID string) (positions []*types.Position, err error)
	ObserveTrades(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Trade, ref uint64)
	ObservePositions(ctx context.Context, retries int, party string) (positions <-chan *types.Position, ref uint64)
	GetTradeSubscribersCount() int32
	GetPositionsSubscribersCount() int32
}

// CandleService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_service_mock.go -package mocks code.vegaprotocol.io/vega/api CandleService
type CandleService interface {
	GetCandles(ctx context.Context, market string, since time.Time, interval types.Interval) (candles []*types.Candle, err error)
	ObserveCandles(ctx context.Context, retries int, market *string, interval *types.Interval) (candleCh <-chan *types.Candle, ref uint64)
	GetCandleSubscribersCount() int32
}

// MarketService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_service_mock.go -package mocks code.vegaprotocol.io/vega/api MarketService
type MarketService interface {
	GetByID(ctx context.Context, name string) (*types.Market, error)
	GetAll(ctx context.Context) ([]*types.Market, error)
	GetDepth(ctx context.Context, market string, limit uint64) (marketDepth *types.MarketDepth, err error)
	ObserveDepth(ctx context.Context, retries int, market string) (depth <-chan *types.MarketDepth, ref uint64)
	GetMarketDepthSubscribersCount() int32
	ObserveMarketsData(ctx context.Context, retries int, marketID string) (<-chan []types.MarketData, uint64)
	GetMarketDataSubscribersCount() int32
	GetMarketDataByID(marketID string) (types.MarketData, error)
	GetMarketsData() []types.MarketData
}

// PartyService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/party_service_mock.go -package mocks code.vegaprotocol.io/vega/api PartyService
type PartyService interface {
	GetByID(ctx context.Context, id string) (*types.Party, error)
	GetAll(ctx context.Context) ([]*types.Party, error)
}

// BlockchainClient ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_client_mock.go -package mocks code.vegaprotocol.io/vega/api BlockchainClient
type BlockchainClient interface {
	SubmitTransaction(ctx context.Context, tx *types.SignedBundle) (bool, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) (success bool, err error)
	CreateOrder(ctx context.Context, order *types.Order) (*types.PendingOrder, error)
	GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error)
	GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error)
	GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error)
	GetUnconfirmedTxCount(ctx context.Context) (count int, err error)
	Health() (*tmctypes.ResultHealth, error)
	NotifyTraderAccount(ctx context.Context, notify *types.NotifyTraderAccount) (success bool, err error)
	Withdraw(context.Context, *types.Withdraw) (success bool, err error)
}

// AccountsService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/accounts_service_mock.go -package mocks code.vegaprotocol.io/vega/api AccountsService
type AccountsService interface {
	GetPartyAccounts(partyID, marketID, asset string, ty types.AccountType) ([]*types.Account, error)
	GetMarketAccounts(marketID, asset string) ([]*types.Account, error)
	ObserveAccounts(ctx context.Context, retries int, marketID, partyID, asset string, ty types.AccountType) (candleCh <-chan []*types.Account, ref uint64)
	GetAccountSubscribersCount() int32
}

// TransferResponseService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/transfer_response_service_mock.go -package mocks code.vegaprotocol.io/vega/api TransferResponseService
type TransferResponseService interface {
	ObserveTransferResponses(ctx context.Context, retries int) (<-chan []*types.TransferResponse, uint64)
}

// RiskService ...
type RiskService interface {
	ObserveMarginLevels(
		ctx context.Context, retries int, partyID, marketID string,
	) (<-chan []types.MarginLevels, uint64)
	GetMarginLevelsSubscribersCount() int32
	GetMarginLevelsByID(partyID, marketID string) ([]types.MarginLevels, error)
}

type tradingDataService struct {
	log                     *logging.Logger
	Config                  Config
	Client                  BlockchainClient
	Stats                   *stats.Stats
	TimeService             VegaTime
	OrderService            OrderService
	TradeService            TradeService
	CandleService           CandleService
	MarketService           MarketService
	PartyService            PartyService
	AccountsService         AccountsService
	RiskService             RiskService
	TransferResponseService TransferResponseService
	statusChecker           *monitoring.Status
	ctx                     context.Context
}

// OrdersByMarket provides a list of orders for a given market.
// Pagination: Optional. If not provided, defaults are used.
// Returns a list of orders sorted by timestamp descending (most recent first).
func (h *tradingDataService) OrdersByMarket(ctx context.Context,
	request *protoapi.OrdersByMarketRequest) (*protoapi.OrdersByMarketResponse, error) {

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	o, err := h.OrderService.GetByMarket(ctx, request.MarketID, p.Skip, p.Limit, p.Descending, &request.Open)
	if err != nil {
		return nil, apiError(codes.Internal, ErrOrderServiceGetByMarket, err)
	}

	var response = &protoapi.OrdersByMarketResponse{}
	if len(o) > 0 {
		response.Orders = o
	}

	return response, nil
}

// OrdersByParty provides a list of orders for a given party.
// Pagination: Optional. If not provided, defaults are used.
// Returns a list of orders sorted by timestamp descending (most recent first).
func (h *tradingDataService) OrdersByParty(ctx context.Context,
	request *protoapi.OrdersByPartyRequest) (*protoapi.OrdersByPartyResponse, error) {

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	o, err := h.OrderService.GetByParty(ctx, request.PartyID, p.Skip, p.Limit, p.Descending, &request.Open)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByParty, err)
	}

	var response = &protoapi.OrdersByPartyResponse{}
	if len(o) > 0 {
		response.Orders = o
	}

	return response, nil
}

// Markets provides a list of all current markets that exist on the VEGA platform.
func (h *tradingDataService) Markets(ctx context.Context, request *empty.Empty) (*protoapi.MarketsResponse, error) {
	markets, err := h.MarketService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarkets, err)
	}
	return &protoapi.MarketsResponse{
		Markets: markets,
	}, nil
}

// OrdersByMarketAndID provides the given order, searching by Market and (Order)Id.
func (h *tradingDataService) OrderByMarketAndID(ctx context.Context,
	request *protoapi.OrderByMarketAndIdRequest) (*protoapi.OrderByMarketAndIdResponse, error) {

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	order, err := h.OrderService.GetByMarketAndID(ctx, request.MarketID, request.OrderID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrOrderServiceGetByMarketAndID, err)
	}

	return &protoapi.OrderByMarketAndIdResponse{
		Order: order,
	}, nil
}

// OrderByReference provides the (possibly not yet accepted/rejected) order.
func (h *tradingDataService) OrderByReference(ctx context.Context, req *protoapi.OrderByReferenceRequest) (*protoapi.OrderByReferenceResponse, error) {

	err := req.Validate()
	if err != nil {
		return nil, err
	}

	order, err := h.OrderService.GetByReference(ctx, req.Reference)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByReference, err)
	}
	return &protoapi.OrderByReferenceResponse{
		Order: order,
	}, nil
}

// Candles returns trade OHLC/volume data for the given time period and interval.
// It will fill in any intervals without trades with zero based candles.
// SinceTimestamp must be in RFC3339 string format.
func (h *tradingDataService) Candles(ctx context.Context,
	request *protoapi.CandlesRequest) (*protoapi.CandlesResponse, error) {

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	c, err := h.CandleService.GetCandles(ctx, request.MarketID, vegatime.UnixNano(request.SinceTimestamp), request.Interval)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandles, err)
	}

	return &protoapi.CandlesResponse{
		Candles: c,
	}, nil
}

// MarketDepth provides the order book for a given market, and also returns the most recent trade
// for the given market.
func (h *tradingDataService) MarketDepth(ctx context.Context, req *protoapi.MarketDepthRequest) (*protoapi.MarketDepthResponse, error) {

	err := req.Validate()
	if err != nil {
		return nil, err
	}

	// Query market depth statistics
	depth, err := h.MarketService.GetDepth(ctx, req.MarketID, req.MaxDepth)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetDepth, err)
	}
	t, err := h.TradeService.GetByMarket(ctx, req.MarketID, 0, 1, true)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	// Build market depth response, including last trade (if available)
	resp := &protoapi.MarketDepthResponse{
		Buy:      depth.Buy,
		MarketID: depth.MarketID,
		Sell:     depth.Sell,
	}
	if len(t) > 0 && t[0] != nil {
		resp.LastTrade = t[0]
	}
	return resp, nil
}

// TradesByMarket provides a list of trades for a given market.
// Pagination: Optional. If not provided, defaults are used.
func (h *tradingDataService) TradesByMarket(ctx context.Context, request *protoapi.TradesByMarketRequest) (*protoapi.TradesByMarketResponse, error) {

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	t, err := h.TradeService.GetByMarket(ctx, request.MarketID, p.Skip, p.Limit, p.Descending)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}
	return &protoapi.TradesByMarketResponse{
		Trades: t,
	}, nil
}

// PositionsByParty provides a list of positions for a given party.
func (h *tradingDataService) PositionsByParty(ctx context.Context, request *protoapi.PositionsByPartyRequest) (*protoapi.PositionsByPartyResponse, error) {

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	// Check here for a valid marketID so we don't fail later
	if request.MarketID != "" {
		_, err := h.MarketService.GetByID(ctx, request.MarketID)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, ErrInvalidMarketID, err)
		}
	}

	positions, err := h.TradeService.GetPositionsByParty(ctx, request.PartyID, request.MarketID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetPositionsByParty, err)
	}
	var response = &protoapi.PositionsByPartyResponse{}
	response.Positions = positions
	return response, nil
}

// MarginLevels returns the current margin levels for a given party and market.
func (h *tradingDataService) MarginLevels(_ context.Context, req *protoapi.MarginLevelsRequest) (*protoapi.MarginLevelsResponse, error) {

	err := req.Validate()
	if err != nil {
		return nil, err
	}

	mls, err := h.RiskService.GetMarginLevelsByID(req.PartyID, req.MarketID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrRiskServiceGetMarginLevelsByID, err)
	}
	levels := make([]*types.MarginLevels, 0, len(mls))
	for _, v := range mls {
		v := v
		levels = append(levels, &v)
	}
	return &protoapi.MarginLevelsResponse{
		MarginLevels: levels,
	}, nil
}

// MarketDataByID provides market data for the given ID.
func (h *tradingDataService) MarketDataByID(_ context.Context, req *protoapi.MarketDataByIDRequest) (*protoapi.MarketDataByIDResponse, error) {

	err := req.Validate()
	if err != nil {
		return nil, err
	}

	md, err := h.MarketService.GetMarketDataByID(req.MarketID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarketData, err)
	}
	return &protoapi.MarketDataByIDResponse{
		MarketData: &md,
	}, nil
}

// MarketsData provides all market data for all markets on this network.
func (h *tradingDataService) MarketsData(_ context.Context, _ *empty.Empty) (*protoapi.MarketsDataResponse, error) {
	mds := h.MarketService.GetMarketsData()
	mdptrs := make([]*types.MarketData, 0, len(mds))
	for _, v := range mds {
		v := v
		mdptrs = append(mdptrs, &v)
	}
	return &protoapi.MarketsDataResponse{
		MarketsData: mdptrs,
	}, nil
}

// Statistics provides various blockchain and Vega statistics, including:
// Blockchain height, backlog length, current time, orders and trades per block, tendermint version
// Vega counts for parties, markets, order actions (amend, cancel, submit), Vega version
func (h *tradingDataService) Statistics(ctx context.Context, request *empty.Empty) (*types.Statistics, error) {
	// Call tendermint and related services to get information for statistics
	// We load read-only internal statistics through each package level statistics structs
	epochTime, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, apiError(codes.Internal, ErrTimeServiceGetTimeNow, err)
	}

	// Call tendermint via rpc client
	backlogLength, numPeers, gt, err := h.getTendermintStats(ctx)
	if err != nil {
		return nil, err // getTendermintStats already returns an API error
	}

	// If the chain is replaying then genesis time can be nil
	genesisTime := ""
	if gt != nil {
		genesisTime = vegatime.Format(*gt)
	}

	// Load current markets details
	m, err := h.MarketService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Unavailable, ErrMarketServiceGetMarkets, err)
	}

	// Load current parties details
	p, err := h.PartyService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Unavailable, ErrPartyServiceGetAll, err)
	}

	// Extract names for ease of reading in stats
	partyNames := make([]string, 0, len(p))
	for _, v := range p {
		if v != nil {
			pp := *v
			partyNames = append(partyNames, pp.Id)
		}
	}

	return &types.Statistics{
		BlockHeight:              h.Stats.Blockchain.Height(),
		BacklogLength:            uint64(backlogLength),
		TotalPeers:               uint64(numPeers),
		GenesisTime:              genesisTime,
		CurrentTime:              vegatime.Format(vegatime.Now()),
		VegaTime:                 vegatime.Format(epochTime),
		Uptime:                   vegatime.Format(h.Stats.GetUptime()),
		TxPerBlock:               uint64(h.Stats.Blockchain.TotalTxLastBatch()),
		AverageTxBytes:           uint64(h.Stats.Blockchain.AverageTxSizeBytes()),
		AverageOrdersPerBlock:    uint64(h.Stats.Blockchain.AverageOrdersPerBatch()),
		TradesPerSecond:          h.Stats.Blockchain.TradesPerSecond(),
		OrdersPerSecond:          h.Stats.Blockchain.OrdersPerSecond(),
		Status:                   h.statusChecker.ChainStatus(),
		TotalMarkets:             uint64(len(m)),
		TotalParties:             uint64(len(p)),
		Parties:                  partyNames,
		AppVersionHash:           h.Stats.GetVersionHash(),
		AppVersion:               h.Stats.GetVersion(),
		ChainVersion:             h.Stats.GetChainVersion(),
		TotalAmendOrder:          h.Stats.Blockchain.TotalAmendOrder(),
		TotalCancelOrder:         h.Stats.Blockchain.TotalCancelOrder(),
		TotalCreateOrder:         h.Stats.Blockchain.TotalCreateOrder(),
		TotalOrders:              h.Stats.Blockchain.TotalOrders(),
		TotalTrades:              h.Stats.Blockchain.TotalTrades(),
		BlockDuration:            h.Stats.Blockchain.BlockDuration(),
		OrderSubscriptions:       uint32(h.OrderService.GetOrderSubscribersCount()),
		TradeSubscriptions:       uint32(h.TradeService.GetTradeSubscribersCount()),
		PositionsSubscriptions:   uint32(h.TradeService.GetPositionsSubscribersCount()),
		MarketDepthSubscriptions: uint32(h.MarketService.GetMarketDepthSubscribersCount()),
		CandleSubscriptions:      uint32(h.CandleService.GetCandleSubscribersCount()),
		AccountSubscriptions:     uint32(h.AccountsService.GetAccountSubscribersCount()),
		MarketDataSubscriptions:  uint32(h.MarketService.GetMarketDataSubscribersCount()),
	}, nil
}

// GetVegaTime returns the latest blockchain header timestamp, in UnixNano format.
// Example: "1568025900111222333" corresponds to 2019-09-09T10:45:00.111222333Z.
func (h *tradingDataService) GetVegaTime(ctx context.Context, request *empty.Empty) (*protoapi.VegaTimeResponse, error) {
	ts, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, apiError(codes.Internal, ErrTimeServiceGetTimeNow, err)
	}
	return &protoapi.VegaTimeResponse{
		Timestamp: ts.UnixNano(),
	}, nil

}

// TransferResponsesSubscribe opens a subscription to transfer response data provided by the transfer response service.
func (h *tradingDataService) TransferResponsesSubscribe(
	req *empty.Empty, srv protoapi.TradingData_TransferResponsesSubscribeServer) error {
	// Wrap context from the request into cancellable. We can close internal chan in error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	transferResponsesChan, ref := h.TransferResponseService.ObserveTransferResponses(ctx, h.Config.StreamRetries)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("TransferResponses subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case transferResponses := <-transferResponsesChan:
			if transferResponses == nil {
				err = ErrChannelClosed
				h.log.Error("TransferResponses subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			for _, tr := range transferResponses {
				tr := tr
				err = srv.Send(tr)
				if err != nil {
					h.log.Error("TransferResponses subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("TransferResponses subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if transferResponsesChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("TransferResponses subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// MarketsDataSubscribe opens a subscription to market data provided by the markets service.
func (h *tradingDataService) MarketsDataSubscribe(req *protoapi.MarketsDataSubscribeRequest,
	srv protoapi.TradingData_MarketsDataSubscribeServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	marketsDataChan, ref := h.MarketService.ObserveMarketsData(ctx, h.Config.StreamRetries, req.MarketID)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("Markets data subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case mds := <-marketsDataChan:
			if mds == nil {
				err = ErrChannelClosed
				h.log.Error("Markets data subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			for _, md := range mds {
				err = srv.Send(&md)
				if err != nil {
					h.log.Error("Markets data subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Markets data subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if marketsDataChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Markets data subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// AccountsSubscribe opens a subscription to the Margin Levels provided by the risk service.
func (h *tradingDataService) MarginLevelsSubscribe(req *protoapi.MarginLevelsSubscribeRequest, srv protoapi.TradingData_MarginLevelsSubscribeServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	err := req.Validate()
	if err != nil {
		return err
	}

	marginLevelsChan, ref := h.RiskService.ObserveMarginLevels(ctx, h.Config.StreamRetries, req.PartyID, req.MarketID)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("Margin levels subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case mls := <-marginLevelsChan:
			if mls == nil {
				err = ErrChannelClosed
				h.log.Error("Margin levels subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			for _, ml := range mls {
				ml := ml
				err = srv.Send(&ml)
				if err != nil {
					h.log.Error("Margin levels data subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Margin levels data subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if marginLevelsChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Margin levels data subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// AccountsSubscribe opens a subscription to the Accounts service.
func (h *tradingDataService) AccountsSubscribe(req *protoapi.AccountsSubscribeRequest,
	srv protoapi.TradingData_AccountsSubscribeServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	accountsChan, ref := h.AccountsService.ObserveAccounts(
		ctx, h.Config.StreamRetries, req.MarketID, req.PartyID, req.Asset, req.Type)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("Accounts subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case accounts := <-accountsChan:
			if accounts == nil {
				err = ErrChannelClosed
				h.log.Error("Accounts subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			for _, account := range accounts {
				account := account
				err = srv.Send(account)
				if err != nil {
					h.log.Error("Accounts subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			err = ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Accounts subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if accountsChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Accounts subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// OrdersSubscribe opens a subscription to the Orders service.
// MarketID: Optional.
// PartyID: Optional.
func (h *tradingDataService) OrdersSubscribe(
	req *protoapi.OrdersSubscribeRequest, srv protoapi.TradingData_OrdersSubscribeServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var (
		err               error
		marketID, partyID *string
	)

	if len(req.MarketID) > 0 {
		marketID = &req.MarketID
	}
	if len(req.PartyID) > 0 {
		partyID = &req.PartyID
	}

	ordersChan, ref := h.OrderService.ObserveOrders(ctx, h.Config.StreamRetries, marketID, partyID)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case orders := <-ordersChan:
			if orders == nil {
				err = ErrChannelClosed
				h.log.Error("Orders subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			out := make([]*types.Order, 0, len(orders))
			for _, v := range orders {
				v := v
				out = append(out, &v)
			}
			err = srv.Send(&protoapi.OrdersStream{Orders: out})
			if err != nil {
				h.log.Error("Orders subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Orders subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if ordersChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Orders subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// TradesSubscribe opens a subscription to the Trades service.
func (h *tradingDataService) TradesSubscribe(req *protoapi.TradesSubscribeRequest,
	srv protoapi.TradingData_TradesSubscribeServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var (
		err               error
		marketID, partyID *string
	)
	if len(req.MarketID) > 0 {
		marketID = &req.MarketID
	}
	if len(req.PartyID) > 0 {
		partyID = &req.PartyID
	}

	tradesChan, ref := h.TradeService.ObserveTrades(ctx, h.Config.StreamRetries, marketID, partyID)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("Trades subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case trades := <-tradesChan:
			if len(trades) <= 0 {
				err = ErrChannelClosed
				h.log.Error("Trades subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}

			out := make([]*types.Trade, 0, len(trades))
			for _, v := range trades {
				v := v
				out = append(out, &v)
			}
			err = srv.Send(&protoapi.TradesStream{Trades: out})
			if err != nil {
				h.log.Error("Trades subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Trades subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}
		if tradesChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Trades subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// CandlesSubscribe opens a subscription to the Candles service.
func (h *tradingDataService) CandlesSubscribe(req *protoapi.CandlesSubscribeRequest,
	srv protoapi.TradingData_CandlesSubscribeServer) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var (
		err      error
		marketID *string
	)
	if len(req.MarketID) > 0 {
		marketID = &req.MarketID
	} else {
		return apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}

	candlesChan, ref := h.CandleService.ObserveCandles(ctx, h.Config.StreamRetries, marketID, &req.Interval)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("Candles subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case candle := <-candlesChan:
			if candle == nil {
				err = ErrChannelClosed
				h.log.Error("Candles subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			err = srv.Send(candle)
			if err != nil {
				h.log.Error("Candles subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Candles subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if candlesChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Candles subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// MarketDepthSubscribe opens a subscription to the MarketDepth service.
func (h *tradingDataService) MarketDepthSubscribe(
	req *protoapi.MarketDepthSubscribeRequest,
	srv protoapi.TradingData_MarketDepthSubscribeServer,
) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	_, err := validateMarket(ctx, req.MarketID, h.MarketService)
	if err != nil {
		return err // validateMarket already returns an API error, no additional wrapping needed
	}

	depthChan, ref := h.MarketService.ObserveDepth(
		ctx, h.Config.StreamRetries, req.MarketID)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("Depth subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case depth := <-depthChan:
			if depth == nil {
				err = ErrChannelClosed
				h.log.Error("Depth subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			err = srv.Send(depth)
			if err != nil {
				if h.log.GetLevel() == logging.DebugLevel {
					h.log.Error("Depth subscriber - rpc stream error",
						logging.Error(err),
						logging.Uint64("ref", ref),
					)
				}
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err = ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Depth subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if depthChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Depth subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// PositionsSubscribe opens a subscription to the Positions service.
func (h *tradingDataService) PositionsSubscribe(
	req *protoapi.PositionsSubscribeRequest,
	srv protoapi.TradingData_PositionsSubscribeServer,
) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	positionsChan, ref := h.TradeService.ObservePositions(ctx, h.Config.StreamRetries, req.PartyID)

	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("Positions subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	for {
		select {
		case position := <-positionsChan:
			if position == nil {
				err := ErrChannelClosed
				h.log.Error("Positions subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, err)
			}
			err := srv.Send(position)
			if err != nil {
				h.log.Error("Positions subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err := ctx.Err()
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Positions subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		case <-h.ctx.Done():
			return apiError(codes.Internal, ErrServerShutdown)
		}

		if positionsChan == nil {
			if h.log.GetLevel() == logging.DebugLevel {
				h.log.Debug("Positions subscriber - rpc stream closed", logging.Uint64("ref", ref))
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

// MarketByID provides the given market.
func (h *tradingDataService) MarketByID(ctx context.Context, req *protoapi.MarketByIDRequest) (*protoapi.MarketByIDResponse, error) {
	mkt, err := validateMarket(ctx, req.MarketID, h.MarketService)
	if err != nil {
		return nil, err // validateMarket already returns an API error, no need to additionally wrap
	}

	return &protoapi.MarketByIDResponse{
		Market: mkt,
	}, nil
}

// Parties provides a list of all parties.
func (h *tradingDataService) Parties(ctx context.Context, req *empty.Empty) (*protoapi.PartiesResponse, error) {
	parties, err := h.PartyService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPartyServiceGetAll, err)
	}
	return &protoapi.PartiesResponse{
		Parties: parties,
	}, nil
}

// PartyByID provides the given party.
func (h *tradingDataService) PartyByID(ctx context.Context, req *protoapi.PartyByIDRequest) (*protoapi.PartyByIDResponse, error) {
	pty, err := validateParty(ctx, req.PartyID, h.PartyService)
	if err != nil {
		return nil, err // validateParty already returns an API error, no need to additionally wrap
	}
	return &protoapi.PartyByIDResponse{
		Party: pty,
	}, nil
}

// TradesByParty provides a list of trades for the given party.
// Pagination: Optional. If not provided, defaults are used.
func (h *tradingDataService) TradesByParty(ctx context.Context,
	req *protoapi.TradesByPartyRequest) (*protoapi.TradesByPartyResponse, error) {

	p := defaultPagination
	if req.Pagination != nil {
		p = *req.Pagination
	}
	trades, err := h.TradeService.GetByParty(ctx, req.PartyID, p.Skip, p.Limit, p.Descending, &req.MarketID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByParty, err)
	}

	return &protoapi.TradesByPartyResponse{Trades: trades}, nil
}

// TradesByOrder provides a list of the trades that correspond to a given order.
func (h *tradingDataService) TradesByOrder(ctx context.Context,
	req *protoapi.TradesByOrderRequest) (*protoapi.TradesByOrderResponse, error) {
	trades, err := h.TradeService.GetByOrderID(ctx, req.OrderID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByOrderID, err)
	}
	return &protoapi.TradesByOrderResponse{
		Trades: trades,
	}, nil
}

// LastTrade provides the last trade for the given market.
func (h *tradingDataService) LastTrade(ctx context.Context,
	req *protoapi.LastTradeRequest) (*protoapi.LastTradeResponse, error) {
	if len(req.MarketID) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}
	t, err := h.TradeService.GetByMarket(ctx, req.MarketID, 0, 1, true)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}
	if len(t) > 0 && t[0] != nil {
		return &protoapi.LastTradeResponse{Trade: t[0]}, nil
	}
	// No trades found on the market yet (and no errors)
	// this can happen at the beginning of a new market
	return &protoapi.LastTradeResponse{}, nil
}

func (h *tradingDataService) MarketAccounts(_ context.Context,
	req *protoapi.MarketAccountsRequest) (*protoapi.MarketAccountsResponse, error) {
	accs, err := h.AccountsService.GetMarketAccounts(req.MarketID, req.Asset)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetMarketAccounts, err)
	}
	return &protoapi.MarketAccountsResponse{
		Accounts: accs,
	}, nil
}

func (h *tradingDataService) PartyAccounts(_ context.Context,
	req *protoapi.PartyAccountsRequest) (*protoapi.PartyAccountsResponse, error) {
	accs, err := h.AccountsService.GetPartyAccounts(req.PartyID, req.MarketID, req.Asset, req.Type)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetPartyAccounts, err)
	}
	return &protoapi.PartyAccountsResponse{
		Accounts: accs,
	}, nil
}

func validateMarket(ctx context.Context, marketID string, marketService MarketService) (*types.Market, error) {
	var mkt *types.Market
	var err error
	if len(marketID) == 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}
	mkt, err = marketService.GetByID(ctx, marketID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetByID, err)
	}
	return mkt, nil
}

func validateParty(ctx context.Context, partyID string, partyService PartyService) (*types.Party, error) {
	var pty *types.Party
	var err error
	if len(partyID) == 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingPartyID)
	}
	pty, err = partyService.GetByID(ctx, partyID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPartyServiceGetByID, err)
	}
	return pty, err
}

func (h *tradingDataService) getTendermintStats(ctx context.Context) (backlogLength int,
	numPeers int, genesis *time.Time, err error) {

	if h.Stats == nil || h.Stats.Blockchain == nil {
		return 0, 0, nil, apiError(codes.Internal, ErrChainNotConnected)
	}

	refused := "connection refused"

	// Unconfirmed TX count == current transaction backlog length
	backlogLength, err = h.Client.GetUnconfirmedTxCount(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return 0, 0, nil, nil
		}
		return 0, 0, nil, apiError(codes.Internal, ErrBlockchainBacklogLength, err)
	}

	// Net info provides peer stats etc (block chain network info) == number of peers
	netInfo, err := h.Client.GetNetworkInfo(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return backlogLength, 0, nil, nil
		}
		return backlogLength, 0, nil, apiError(codes.Internal, ErrBlockchainNetworkInfo, err)
	}

	// Genesis retrieves the current genesis date/time for the blockchain
	genesisTime, err := h.Client.GetGenesisTime(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return backlogLength, 0, nil, nil
		}
		return backlogLength, 0, nil, apiError(codes.Internal, ErrBlockchainGenesisTime, err)
	}

	return backlogLength, netInfo.NPeers, &genesisTime, nil
}
