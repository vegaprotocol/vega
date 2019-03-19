package grpc

import (
	"code.vegaprotocol.io/vega/internal/parties"
	"context"
	"fmt"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/filtering"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var ErrChainNotConnected = errors.New("Chain not connected")

type Handlers struct {
	Client        blockchain.Client
	Stats         *internal.Stats
	TimeService   vegatime.Service
	OrderService  orders.Service
	TradeService  trades.Service
	CandleService candles.Service
	MarketService markets.Service
	PartyService  parties.Service
	statusChecker *monitoring.Status
}

// If no limit is provided at the gRPC API level, the system will use this limit instead.
// (Prevent returning all results every time a careless query is made)
const defaultLimit = uint64(1000)

// CreateOrder is used to request sending an order into the VEGA platform, via consensus.
func (h *Handlers) CreateOrder(ctx context.Context, order *types.Order) (*api.OrderResponse, error) {
	if h.statusChecker.Blockchain.Status() != types.ChainStatus_CONNECTED {
		return nil, ErrChainNotConnected
	}
	success, reference, err := h.OrderService.CreateOrder(ctx, order)
	return &api.OrderResponse{Success: success, Reference: reference}, err
}

// CancelOrder is used to request cancelling an order into the VEGA platform, via consensus.
func (h *Handlers) CancelOrder(ctx context.Context, order *types.Order) (*api.OrderResponse, error) {
	if h.statusChecker.Blockchain.Status() != types.ChainStatus_CONNECTED {
		return nil, ErrChainNotConnected
	}
	success, err := h.OrderService.CancelOrder(ctx, order)
	return &api.OrderResponse{Success: success}, err
}

// AmendOrder is used to request editing an order onto the VEGA platform, via consensus.
func (h *Handlers) AmendOrder(ctx context.Context, amendment *types.Amendment) (*api.OrderResponse, error) {
	success, err := h.OrderService.AmendOrder(ctx, amendment)
	return &api.OrderResponse{Success: success}, err
}

// OrdersByMarket provides a list of orders for a given market. Optional limits can be provided. Most recent first.
func (h *Handlers) OrdersByMarket(ctx context.Context,
	request *api.OrdersByMarketRequest) (*api.OrdersByMarketResponse, error) {

	if request.Market == "" {
		return nil, errors.New("Market empty or missing")
	}

	orderFilters := &filtering.OrderQueryFilters{}
	if request.Params != nil && request.Params.Limit > 0 {
		orderFilters.Last = &request.Params.Limit
	}

	o, err := h.OrderService.GetByMarket(ctx, request.Market, orderFilters)
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

	orderFilters := &filtering.OrderQueryFilters{}
	if request.Params != nil && request.Params.Limit > 0 {
		orderFilters.Last = &request.Params.Limit
	}

	o, err := h.OrderService.GetByParty(ctx, request.Party, orderFilters)
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
			res = append(res, mv.Name)
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

	c, err := h.CandleService.GetCandles(ctx, market, request.SinceTimestamp, request.Interval)
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
	// Query last 1 trades from store
	queryFilters := &filtering.TradeQueryFilters{}
	last := uint64(1)
	queryFilters.Last = &last
	t, err := h.TradeService.GetByMarket(ctx, request.Market, queryFilters)
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

	f := &filtering.TradeQueryFilters{}
	*f.Last = limit

	t, err := h.TradeService.GetByMarket(ctx, request.Market, f)
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
	epochTimeNano, _, err := h.TimeService.GetTimeNow()
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
		genesisTime = gt.Format(time.RFC3339)
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
	var partyNames []string
	for _,v := range p {
		partyNames = append(partyNames, v.Name)
	}

	return &types.Statistics{
		BlockHeight:           h.Stats.Blockchain.Height(),
		BacklogLength:         uint64(backlogLength),
		TotalPeers:            uint64(numPeers),
		GenesisTime:           genesisTime,
		CurrentTime:           time.Now().UTC().Format(time.RFC3339),
		VegaTime:              epochTimeNano.Rfc3339(),
		TxPerBlock:            uint64(h.Stats.Blockchain.AverageTxPerBatch()),
		AverageTxBytes:        uint64(h.Stats.Blockchain.AverageTxSizeBytes()),
		AverageOrdersPerBlock: uint64(h.Stats.Blockchain.AverageOrdersPerBatch()),
		TradesPerSecond:       uint64(h.Stats.Blockchain.TotalTradesLastBatch()),
		OrdersPerSecond:       uint64(h.Stats.Blockchain.TotalOrdersLastBatch()),
		Status:                h.statusChecker.Blockchain.Status(),
		TotalMarkets:          uint64(len(m)),
		TotalParties:          uint64(len(p)),
		Parties:               partyNames,
		LastTrade:             h.TradeService.GetLastTrade(ctx),
		LastOrder:             h.OrderService.GetLastOrder(ctx),
		AppVersionHash:        h.Stats.GetVersionHash(),
		AppVersion:            h.Stats.GetVersion(),
	}, nil
}

func (h *Handlers) GetVegaTime(ctx context.Context, request *api.VegaTimeRequest) (*api.VegaTimeResponse, error) {
	epochTimeNano, _, err := h.TimeService.GetTimeNow()
	if err != nil {
		return nil, err
	}
	var response = &api.VegaTimeResponse{}
	response.Time = fmt.Sprintf("%s", epochTimeNano.Rfc3339())
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
