package api

import (
	"context"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
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
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open bool) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open bool) (orders []*types.Order, err error)
	GetByMarketAndID(ctx context.Context, market string, id string) (order *types.Order, err error)
	GetByOrderID(ctx context.Context, id string, version uint64) (order *types.Order, err error)
	GetByReference(ctx context.Context, ref string) (order *types.Order, err error)
	GetAllVersionsByOrderID(ctx context.Context, id string, skip, limit uint64, descending bool) (orders []*types.Order, err error)
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
	CreateOrder(ctx context.Context, order *types.Order) error
	GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error)
	GetChainID(ctx context.Context) (chainID string, err error)
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

// GovernanceDataService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/governance_data_service_mock.go -package mocks code.vegaprotocol.io/vega/api  GovernanceDataService
type GovernanceDataService interface {
	GetProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetProposalsByParty(partyID string, inState *types.Proposal_State) []*types.GovernanceData
	GetVotesByParty(partyID string) []*types.Vote

	GetProposalByID(id string) (*types.GovernanceData, error)
	GetProposalByReference(ref string) (*types.GovernanceData, error)

	GetNewMarketProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetUpdateMarketProposals(marketID string, inState *types.Proposal_State) []*types.GovernanceData
	GetNetworkParametersProposals(inState *types.Proposal_State) []*types.GovernanceData
	GetNewAssetProposals(inState *types.Proposal_State) []*types.GovernanceData

	ObserveGovernance(ctx context.Context, retries int) <-chan []types.GovernanceData
	ObservePartyProposals(ctx context.Context, retries int, partyID string) <-chan []types.GovernanceData
	ObservePartyVotes(ctx context.Context, retries int, partyID string) <-chan []types.Vote
	ObserveProposalVotes(ctx context.Context, retries int, proposalID string) <-chan []types.Vote
}

// RiskService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_service_mock.go -package mocks code.vegaprotocol.io/vega/api  RiskService
type RiskService interface {
	ObserveMarginLevels(
		ctx context.Context, retries int, partyID, marketID string,
	) (<-chan []types.MarginLevels, uint64)
	GetMarginLevelsSubscribersCount() int32
	GetMarginLevelsByID(partyID, marketID string) ([]types.MarginLevels, error)
}

// Notary ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_service_mock.go -package mocks code.vegaprotocol.io/vega/api  NotaryService
type NotaryService interface {
	GetByID(id string) ([]types.NodeSignature, error)
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
	NotaryService           NotaryService
	TransferResponseService TransferResponseService
	governanceService       GovernanceDataService
	statusChecker           *monitoring.Status
	ctx                     context.Context
}

func (h *tradingDataService) GetNodeSignaturesAggregate(ctx context.Context,
	req *protoapi.GetNodeSignaturesAggregateRequest) (*protoapi.GetNodeSignaturesAggregateResponse, error) {
	if len(req.ID) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing ID"))
	}

	sigs, err := h.NotaryService.GetByID(req.ID)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	out := make([]*types.NodeSignature, 0, len(sigs))
	for _, v := range sigs {
		v := v
		out = append(out, &v)
	}

	return &protoapi.GetNodeSignaturesAggregateResponse{
		Signatures: out,
	}, nil
}

// OrdersByMarket provides a list of orders for a given market.
// Pagination: Optional. If not provided, defaults are used.
// Returns a list of orders sorted by timestamp descending (most recent first).
func (h *tradingDataService) OrdersByMarket(ctx context.Context,
	request *protoapi.OrdersByMarketRequest) (*protoapi.OrdersByMarketResponse, error) {
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("OrdersByMarket", startTime)

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	o, err := h.OrderService.GetByMarket(ctx, request.MarketID, p.Skip, p.Limit, p.Descending, request.Open)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("OrdersByParty", startTime)

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	p := defaultPagination
	if request.Pagination != nil {
		p = *request.Pagination
	}

	o, err := h.OrderService.GetByParty(ctx, request.PartyID, p.Skip, p.Limit, p.Descending, request.Open)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("Markets", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("OrderByMarketAndID", startTime)

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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("OrderByReference", startTime)

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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("Candles", startTime)

	err := request.Validate()
	if err != nil {
		return nil, err
	}

	if request.Interval == types.Interval_INTERVAL_UNSPECIFIED {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarketDepth", startTime)

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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("TradesByMarket", startTime)

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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("PositionsByParty", startTime)

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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarginLevels", startTime)

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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarketDataByID", startTime)

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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarketsData", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("Statistics", startTime)
	// Call tendermint and related services to get information for statistics
	// We load read-only internal statistics through each package level statistics structs
	epochTime, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, apiError(codes.Internal, ErrTimeServiceGetTimeNow, err)
	}

	// Call tendermint via rpc client
	backlogLength, numPeers, gt, chainID, err := h.getTendermintStats(ctx)
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
		ChainID:                  chainID,
	}, nil
}

// GetVegaTime returns the latest blockchain header timestamp, in UnixNano format.
// Example: "1568025900111222333" corresponds to 2019-09-09T10:45:00.111222333Z.
func (h *tradingDataService) GetVegaTime(ctx context.Context, request *empty.Empty) (*protoapi.VegaTimeResponse, error) {
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetVegaTime", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("TransferResponseSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarketsDataSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarginLevelsSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("AccountsSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("OrdersSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("TradesSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("CandlesSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarketDepthSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("PositionsSubscribe", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarketByID", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("Parties", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("PartyByID", startTime)
	pty, err := validateParty(ctx, h.log, req.PartyID, h.PartyService)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("TradesByParty", startTime)

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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("TradesByOrder", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("LastTrade", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("MarketAccounts", startTime)
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
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("PartyAccounts", startTime)
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

func validateParty(ctx context.Context, log *logging.Logger, partyID string, partyService PartyService) (*types.Party, error) {
	var pty *types.Party
	var err error
	if len(partyID) == 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingPartyID)
	}
	pty, err = partyService.GetByID(ctx, partyID)
	if err != nil {
		// we just log the error here, then return an nil error.
		// right now the only error possible is about not finding a party
		// we just not an actual error
		log.Debug("error getting party by ID",
			logging.Error(err),
			logging.String("party-id", partyID))
		err = nil
	}
	return pty, err
}

func (h *tradingDataService) getTendermintStats(ctx context.Context) (backlogLength int,
	numPeers int, genesis *time.Time, chainID string, err error) {

	if h.Stats == nil || h.Stats.Blockchain == nil {
		return 0, 0, nil, "", apiError(codes.Internal, ErrChainNotConnected)
	}

	refused := "connection refused"

	// Unconfirmed TX count == current transaction backlog length
	backlogLength, err = h.Client.GetUnconfirmedTxCount(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return 0, 0, nil, "", nil
		}
		return 0, 0, nil, "", apiError(codes.Internal, ErrBlockchainBacklogLength, err)
	}

	// Net info provides peer stats etc (block chain network info) == number of peers
	netInfo, err := h.Client.GetNetworkInfo(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return backlogLength, 0, nil, "", nil
		}
		return backlogLength, 0, nil, "", apiError(codes.Internal, ErrBlockchainNetworkInfo, err)
	}

	// Genesis retrieves the current genesis date/time for the blockchain
	genesisTime, err := h.Client.GetGenesisTime(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return backlogLength, 0, nil, "", nil
		}
		return backlogLength, 0, nil, "", apiError(codes.Internal, ErrBlockchainGenesisTime, err)
	}

	chainId, err := h.Client.GetChainID(ctx)
	if err != nil {
		return backlogLength, 0, nil, "", apiError(codes.Internal, ErrBlockchainChainID, err)
	}

	return backlogLength, netInfo.NPeers, &genesisTime, chainId, nil
}

func (h *tradingDataService) OrderByID(ctx context.Context, in *protoapi.OrderByIDRequest) (*types.Order, error) {
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("OrderByID", startTime)
	if len(in.OrderID) == 0 {
		// Invalid parameter
		return nil, ErrMissingOrderIDParameter
	}

	order, err := h.OrderService.GetByOrderID(ctx, in.OrderID, in.Version)
	if err == nil {
		return order, nil
	}

	// If we get here then no match was found
	return nil, ErrOrderNotFound
}

func (h *tradingDataService) OrderByReferenceID(ctx context.Context, in *protoapi.OrderByReferenceIDRequest) (*types.Order, error) {
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("OrderByReferenceID", startTime)
	if len(in.ReferenceID) == 0 {
		// Invalid parameter
		return nil, ErrMissingReferenceIDParameter
	}

	order, err := h.OrderService.GetByReference(ctx, in.ReferenceID)
	if err != nil {
		return nil, err
	}

	// If we get here we have matched against referenceID and all is good
	return order, nil
}

// OrderVersionsByID returns all versions of the order by its orderID
func (h *tradingDataService) OrderVersionsByID(
	ctx context.Context,
	in *protoapi.OrderVersionsByIDRequest,
) (*protoapi.OrderVersionsResponse, error) {
	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("OrderVersionsByID", startTime)

	err := in.Validate()
	if err != nil {
		return nil, err
	}
	p := defaultPagination
	if in.Pagination != nil {
		p = *in.Pagination
	}
	orders, err := h.OrderService.GetAllVersionsByOrderID(ctx,
		in.OrderID,
		p.Skip,
		p.Limit,
		p.Descending)
	if err == nil {
		return &protoapi.OrderVersionsResponse{
			Orders: orders,
		}, nil
	}
	return nil, err
}

func (h *tradingDataService) GetProposals(_ context.Context,
	in *protoapi.GetProposalsRequest,
) (*protoapi.GetProposalsResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetProposals", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetProposalsResponse{
		Data: h.governanceService.GetProposals(inState),
	}, nil
}

func (h *tradingDataService) GetProposalsByParty(_ context.Context,
	in *protoapi.GetProposalsByPartyRequest,
) (*protoapi.GetProposalsByPartyResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetProposalsByParty", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetProposalsByPartyResponse{
		Data: h.governanceService.GetProposalsByParty(in.PartyID, inState),
	}, nil
}

func (h *tradingDataService) GetVotesByParty(_ context.Context,
	in *protoapi.GetVotesByPartyRequest,
) (*protoapi.GetVotesByPartyResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetVotesByParty", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	return &protoapi.GetVotesByPartyResponse{
		Votes: h.governanceService.GetVotesByParty(in.PartyID),
	}, nil
}

func (h *tradingDataService) GetNewMarketProposals(_ context.Context,
	in *protoapi.GetNewMarketProposalsRequest,
) (*protoapi.GetNewMarketProposalsResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetNewMarketProposals", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetNewMarketProposalsResponse{
		Data: h.governanceService.GetNewMarketProposals(inState),
	}, nil
}

func (h *tradingDataService) GetUpdateMarketProposals(_ context.Context,
	in *protoapi.GetUpdateMarketProposalsRequest,
) (*protoapi.GetUpdateMarketProposalsResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetUpdateMarketProposals", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetUpdateMarketProposalsResponse{
		Data: h.governanceService.GetUpdateMarketProposals(in.MarketID, inState),
	}, nil
}

func (h *tradingDataService) GetNetworkParametersProposals(_ context.Context,
	in *protoapi.GetNetworkParametersProposalsRequest,
) (*protoapi.GetNetworkParametersProposalsResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetNetworkParametersProposals", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetNetworkParametersProposalsResponse{
		Data: h.governanceService.GetNetworkParametersProposals(inState),
	}, nil
}

func (h *tradingDataService) GetNewAssetProposals(_ context.Context,
	in *protoapi.GetNewAssetProposalsRequest,
) (*protoapi.GetNewAssetProposalsResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetNewAssetProposals", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	var inState *types.Proposal_State
	if in.SelectInState != nil {
		inState = &in.SelectInState.Value
	}
	return &protoapi.GetNewAssetProposalsResponse{
		Data: h.governanceService.GetNewAssetProposals(inState),
	}, nil
}

func (h *tradingDataService) GetProposalByID(_ context.Context,
	in *protoapi.GetProposalByIDRequest,
) (*protoapi.GetProposalByIDResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetProposalByID", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	proposal, err := h.governanceService.GetProposalByID(in.ProposalID)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMissingProposalID, err)
	}
	return &protoapi.GetProposalByIDResponse{Data: proposal}, nil
}

func (h *tradingDataService) GetProposalByReference(_ context.Context,
	in *protoapi.GetProposalByReferenceRequest,
) (*protoapi.GetProposalByReferenceResponse, error) {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("GetProposalByReference", startTime)

	if err := in.Validate(); err != nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	proposal, err := h.governanceService.GetProposalByReference(in.Reference)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMissingProposalReference, err)
	}
	return &protoapi.GetProposalByReferenceResponse{Data: proposal}, nil
}

func (h *tradingDataService) ObserveGovernance(
	_ *empty.Empty,
	stream protoapi.TradingData_ObserveGovernanceServer,
) error {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("ObserveGovernance", startTime)
	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("starting streaming governance updates")
	}
	ch := h.governanceService.ObserveGovernance(ctx, h.Config.StreamRetries)
	for {
		select {
		case props, ok := <-ch:
			if !ok {
				cfunc()
				return nil
			}
			for _, p := range props {
				if err := stream.Send(&p); err != nil {
					h.log.Error("failed to send governance data into stream",
						logging.Error(err))
					return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
				}
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-h.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}
func (h *tradingDataService) ObservePartyProposals(
	in *protoapi.ObservePartyProposalsRequest,
	stream protoapi.TradingData_ObservePartyProposalsServer,
) error {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("ObservePartyProposals", startTime)

	if err := in.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("starting streaming party proposals")
	}
	ch := h.governanceService.ObservePartyProposals(ctx, h.Config.StreamRetries, in.PartyID)
	for {
		select {
		case props, ok := <-ch:
			if !ok {
				cfunc()
				return nil
			}
			for _, p := range props {
				if err := stream.Send(&p); err != nil {
					h.log.Error("failed to send party proposal into stream",
						logging.Error(err))
					return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
				}
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-h.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}

func (h *tradingDataService) ObservePartyVotes(
	in *protoapi.ObservePartyVotesRequest,
	stream protoapi.TradingData_ObservePartyVotesServer,
) error {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("ObservePartyVotes", startTime)

	if err := in.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("starting streaming party votes")
	}
	ch := h.governanceService.ObservePartyVotes(ctx, h.Config.StreamRetries, in.PartyID)
	for {
		select {
		case votes, ok := <-ch:
			if !ok {
				cfunc()
				return nil
			}
			for _, p := range votes {
				if err := stream.Send(&p); err != nil {
					h.log.Error("failed to send party vote into stream",
						logging.Error(err))
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-h.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}

func (h *tradingDataService) ObserveProposalVotes(
	in *protoapi.ObserveProposalVotesRequest,
	stream protoapi.TradingData_ObserveProposalVotesServer,
) error {

	startTime := vegatime.Now()
	defer metrics.APIRequestAndTimeGRPC("ObserveProposalVotes", startTime)

	if err := in.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if h.log.GetLevel() == logging.DebugLevel {
		h.log.Debug("starting streaming proposal votes")
	}
	ch := h.governanceService.ObserveProposalVotes(ctx, h.Config.StreamRetries, in.ProposalID)
	for {
		select {
		case votes, ok := <-ch:
			if !ok {
				cfunc()
				return nil
			}
			for _, p := range votes {
				if err := stream.Send(&p); err != nil {
					h.log.Error("failed to send proposal vote into stream",
						logging.Error(err))
					return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
				}
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		case <-h.ctx.Done():
			return apiError(codes.Aborted, ErrServerShutdown)
		}
	}
}

// func (h *tradingDataService) TransferResponsesSubscribe(
// req *empty.Empty, srv protoapi.TradingData_TransferResponsesSubscribeServer) error {
