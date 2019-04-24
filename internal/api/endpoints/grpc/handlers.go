package grpc

import (
	"context"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/vegatime"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

var ErrChainNotConnected = errors.New("Chain not connected")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/vega_time_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc VegaTime
type VegaTime interface {
	GetTimeNow() (time.Time, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc OrderService
type OrderService interface {
	CreateOrder(ctx context.Context, order *types.OrderSubmission) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) (*types.PendingOrder, error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error)
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (order *types.Order, err error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/trade_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc TradeService
type TradeService interface {
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (trades []*types.Trade, err error)
	GetPositionsByParty(ctx context.Context, party string) (positions []*types.MarketPosition, err error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/candle_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc CandleService
type CandleService interface {
	GetCandles(ctx context.Context, market string, since time.Time, interval types.Interval) (candles []*types.Candle, err error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc MarketService
type MarketService interface {
	GetAll(ctx context.Context) ([]*types.Market, error)
	GetDepth(ctx context.Context, market string) (marketDepth *types.MarketDepth, err error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/party_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc PartyService
type PartyService interface {
	GetAll(ctx context.Context) ([]*types.Party, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_client_mock.go -package mocks code.vegaprotocol.io/vega/internal/api/endpoints/grpc BlockchainClient
type BlockchainClient interface {
	GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error)
	GetUnconfirmedTxCount(ctx context.Context) (count int, err error)
	GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error)
}

type Handlers struct {
	Client        BlockchainClient
	Stats         *internal.Stats
	TimeService   VegaTime
	OrderService  OrderService
	TradeService  TradeService
	CandleService CandleService
	MarketService MarketService
	PartyService  PartyService
	statusChecker *monitoring.Status
}

// If no limit is provided at the gRPC API level, the system will use this limit instead.
// (Prevent returning all results every time a careless query is made)
const defaultLimit = uint64(1000)

// CreateOrder is used to request sending an order into the VEGA platform, via consensus.
func (h *Handlers) SubmitOrder(ctx context.Context, order *types.OrderSubmission) (*types.PendingOrder, error) {
	if h.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return nil, ErrChainNotConnected
	}
	pendingOrder, err := h.OrderService.CreateOrder(ctx, order)
	return pendingOrder, err
}

// CancelOrder is used to request cancelling an order into the VEGA platform, via consensus.
func (h *Handlers) CancelOrder(ctx context.Context, order *types.OrderCancellation) (*types.PendingOrder, error) {
	if h.statusChecker.ChainStatus() != types.ChainStatus_CONNECTED {
		return nil, ErrChainNotConnected
	}
	pendingOrder, err := h.OrderService.CancelOrder(ctx, order)
	return pendingOrder, err
}

// AmendOrder is used to request editing an order onto the VEGA platform, via consensus.
func (h *Handlers) AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (*api.OrderResponse, error) {
	success, err := h.OrderService.AmendOrder(ctx, amendment)
	return &api.OrderResponse{Success: success}, err
}

// OrdersByMarket provides a list of orders for a given market. Optional limits can be provided. Most recent first.
func (h *Handlers) OrdersByMarket(ctx context.Context,
	request *api.OrdersByMarketRequest) (*api.OrdersByMarketResponse, error) {

	if request.Market == "" {
		return nil, errors.New("Market empty or missing")
	}

	var (
		skip, limit uint64
		descending  bool
		open        *bool
	)
	if request.Params != nil && request.Params.Limit > 0 {
		descending = true
		limit = request.Params.Limit
	}

	o, err := h.OrderService.GetByMarket(ctx, request.Market, skip, limit, descending, open)
	if err != nil {
		return nil, err
	}

	var response = &api.OrdersByMarketResponse{}
	if len(o) > 0 {
		response.Orders = o
	}

	return response, nil
}

// OrdersByParty provides a list of orders for a given party. Optional limits can be provided. Most recent first.
func (h *Handlers) OrdersByParty(ctx context.Context,
	request *api.OrdersByPartyRequest) (*api.OrdersByPartyResponse, error) {

	if request.Party == "" {
		return nil, errors.New("Party empty or missing")
	}

	var limit uint64
	if request.Params != nil && request.Params.Limit > 0 {
		limit = request.Params.Limit
	}

	o, err := h.OrderService.GetByParty(ctx, request.Party, 0, limit, true, nil)
	if err != nil {
		return nil, err
	}

	var response = &api.OrdersByPartyResponse{}
	if len(o) > 0 {
		response.Orders = o
	}

	return response, nil
}

// Markets provides a list of all current markets that exist on the VEGA platform.
func (h *Handlers) Markets(ctx context.Context, request *api.MarketsRequest) (*api.MarketsResponse, error) {
	m, err := h.MarketService.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	var response = &api.MarketsResponse{}
	if len(m) > 0 {
		var res []string
		for _, mv := range m {
			res = append(res, mv.Id)
		}
		response.Markets = res
	}
	return response, nil
}

// OrdersByMarketAndId searches for the given order by Id and Market. If found it will return
// an Order types otherwise it will return an error.
func (h *Handlers) OrderByMarketAndId(ctx context.Context,
	request *api.OrderByMarketAndIdRequest) (*api.OrderByMarketAndIdResponse, error) {

	if request.Market == "" {
		return nil, errors.New("Market empty or missing")
	}
	if request.Id == "" {
		return nil, errors.New("Id empty or missing")
	}
	order, err := h.OrderService.GetByMarketAndId(ctx, request.Market, request.Id)
	if err != nil {
		return nil, err
	}

	var response = &api.OrderByMarketAndIdResponse{}
	response.Order = order
	return response, nil
}

// TradeCandles returns trade open/close/volume data for the given time period and interval.
// It will fill in any trade-less intervals with zero based candles. Since time period must be in RFC3339 string format.
func (h *Handlers) Candles(ctx context.Context,
	request *api.CandlesRequest) (*api.CandlesResponse, error) {

	market := request.Market
	if market == "" {
		return nil, errors.New("Market empty or missing")
	}

	if request.SinceTimestamp == 0 {
		return nil, errors.New("Since date is missing")
	}

	c, err := h.CandleService.GetCandles(ctx, market, vegatime.UnixNano(request.SinceTimestamp), request.Interval)
	if err != nil {
		return nil, err
	}

	response := &api.CandlesResponse{}
	response.Candles = c
	return response, nil
}

func (h *Handlers) MarketDepth(ctx context.Context, request *api.MarketDepthRequest) (*api.MarketDepthResponse, error) {
	if request.Market == "" {
		return nil, errors.New("Market empty or missing")
	}
	// Query market depth statistics
	depth, err := h.MarketService.GetDepth(ctx, request.Market)
	if err != nil {
		return nil, err
	}
	t, err := h.TradeService.GetByMarket(ctx, request.Market, 0, 1, true)
	if err != nil {
		return nil, err
	}
	// Build market depth response, including last trade (if available)
	var response = &api.MarketDepthResponse{}
	response.Buy = depth.Buy
	response.Name = depth.Name
	response.Sell = depth.Sell
	if t != nil && t[0] != nil {
		response.LastTrade = t[0]
	}
	return response, nil
}

func (h *Handlers) TradesByMarket(ctx context.Context, request *api.TradesByMarketRequest) (*api.TradesByMarketResponse, error) {
	if request.Market == "" {
		return nil, errors.New("Market empty or missing")
	}
	limit := defaultLimit
	if request.Params != nil && request.Params.Limit > 0 {
		limit = request.Params.Limit
	}

	t, err := h.TradeService.GetByMarket(ctx, request.Market, 0, limit, true)
	if err != nil {
		return nil, err
	}
	var response = &api.TradesByMarketResponse{}
	response.Trades = t
	return response, nil
}

func (h *Handlers) PositionsByParty(ctx context.Context, request *api.PositionsByPartyRequest) (*api.PositionsByPartyResponse, error) {
	if request.Party == "" {
		return nil, errors.New("Party empty or missing")
	}
	positions, err := h.TradeService.GetPositionsByParty(ctx, request.Party)
	if err != nil {
		return nil, err
	}
	var response = &api.PositionsByPartyResponse{}
	response.Positions = positions
	return response, nil
}

func (h *Handlers) Statistics(ctx context.Context, request *api.StatisticsRequest) (*types.Statistics, error) {
	// Call out to tendermint and related services to get related information for statistics
	// We load read-only internal statistics through each package level statistics structs
	epochTime, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	if h.Stats == nil || h.Stats.Blockchain == nil {
		return nil, errors.New("Internal error: statistics not available")
	}

	// Call out to tendermint via rpc client
	backlogLength, numPeers, gt, err := h.getTendermintStats(ctx)
	if err != nil {
		return nil, err
	}

	// If the chain is replaying then genesis time can be nil
	genesisTime := ""
	if gt != nil {
		genesisTime = vegatime.Format(*gt)
	}

	// Load current markets details
	m, err := h.MarketService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Load current parties details
	p, err := h.PartyService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Extract names for ease of reading in stats
	partyNames := make([]string, 0, len(p))
	for _, v := range p {
		if v != nil {
			pp := *v
			partyNames = append(partyNames, pp.Name)
		}
	}

	return &types.Statistics{
		BlockHeight:           h.Stats.Blockchain.Height(),
		BacklogLength:         uint64(backlogLength),
		TotalPeers:            uint64(numPeers),
		GenesisTime:           genesisTime,
		CurrentTime:           vegatime.Format(vegatime.Now()),
		VegaTime:              vegatime.Format(epochTime),
		TxPerBlock:            uint64(h.Stats.Blockchain.AverageTxPerBatch()),
		AverageTxBytes:        uint64(h.Stats.Blockchain.AverageTxSizeBytes()),
		AverageOrdersPerBlock: uint64(h.Stats.Blockchain.AverageOrdersPerBatch()),
		TradesPerSecond:       uint64(h.Stats.Blockchain.TradesPerSecond()),
		OrdersPerSecond:       uint64(h.Stats.Blockchain.OrdersPerSecond()),
		Status:                h.statusChecker.ChainStatus(),
		TotalMarkets:          uint64(len(m)),
		TotalParties:          uint64(len(p)),
		Parties:               partyNames,
		AppVersionHash:        h.Stats.GetVersionHash(),
		AppVersion:            h.Stats.GetVersion(),
		TotalAmendOrder:       h.Stats.Blockchain.TotalAmendOrder(),
		TotalCancelOrder:      h.Stats.Blockchain.TotalCancelOrder(),
		TotalCreateOrder:      h.Stats.Blockchain.TotalCreateOrder(),
		TotalOrders:           h.Stats.Blockchain.TotalOrders(),
		TotalTrades:           h.Stats.Blockchain.TotalTrades(),
	}, nil
}

func (h *Handlers) GetVegaTime(ctx context.Context, request *api.VegaTimeRequest) (*api.VegaTimeResponse, error) {
	epochTime, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	var response = &api.VegaTimeResponse{}
	response.Time = vegatime.Format(epochTime)
	return response, nil
}

func (h *Handlers) getTendermintStats(ctx context.Context) (backlogLength int,
	numPeers int, genesis *time.Time, err error) {

	refused := "connection refused"

	// Unconfirmed TX count == current transaction backlog length
	backlogLength, err = h.Client.GetUnconfirmedTxCount(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return 0, 0, nil, nil
		}
		return 0, 0, nil, err
	}

	// Net info provides peer stats etc (block chain network info) == number of peers
	netInfo, err := h.Client.GetNetworkInfo(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return backlogLength, 0, nil, nil
		}
		return backlogLength, 0, nil, err
	}

	// Genesis retrieves the current genesis date/time for the blockchain
	genesisTime, err := h.Client.GetGenesisTime(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return backlogLength, 0, nil, nil
		}
		return backlogLength, 0, nil, err
	}

	return backlogLength, netInfo.NPeers, &genesisTime, nil
}
