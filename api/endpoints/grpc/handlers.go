package grpc

import (
	"context"
	"vega/msg"
	"vega/api"
	"time"
	"github.com/pkg/errors"
	"vega/filters"
	"fmt"
)

type Handlers struct {
	OrderService api.OrderService
	TradeService api.TradeService
}

// If no limit is provided at the gRPC API level, the system will use this limit instead.
// (Prevent returning all results every time a careless query is made)
const defaultLimit = uint64(1000)

// CreateOrder is used to request sending an order into the VEGA platform, via consensus.
func (h *Handlers) CreateOrder(ctx context.Context, order *msg.Order) (*api.OrderResponse, error) {
	success, reference, err := h.OrderService.CreateOrder(ctx, order)
	return &api.OrderResponse{Success: success, Reference: reference}, err
}

// CancelOrder is used to request cancelling an order into the VEGA platform, via consensus.
func (h *Handlers) CancelOrder(ctx context.Context, order *msg.Order) (*api.OrderResponse, error) {
	success, err := h.OrderService.CancelOrder(ctx, order)
	return &api.OrderResponse{Success: success}, err
}

// AmendOrder is used to request editing an order onto the VEGA platform, via consensus.
func (h *Handlers) AmendOrder(ctx context.Context, amendment *msg.Amendment) (*api.OrderResponse, error) {
	success, err := h.OrderService.AmendOrder(ctx, amendment)
	return &api.OrderResponse{Success: success}, err
}

// OrdersByMarket provides a list of orders for a given market. Optional limits can be provided. Most recent first.
func (h *Handlers) OrdersByMarket(ctx context.Context, request *api.OrdersByMarketRequest) (*api.OrdersByMarketResponse, error) {
	if request.Market == "" {
		return nil, errors.New("Market empty or missing")
	}
	orderFilters := &filters.OrderQueryFilters{}
	if request.Params != nil && request.Params.Limit > 0 {
		orderFilters.Last = &request.Params.Limit
	}
	orders, err := h.OrderService.GetByMarket(ctx, request.Market, orderFilters)
	if err != nil {
		return nil, err
	}
	var response = &api.OrdersByMarketResponse{}
	if len(orders) > 0 {
	   response.Orders = orders
	}
	return response, nil
}

// OrdersByParty provides a list of orders for a given party. Optional limits can be provided. Most recent first.
func (h *Handlers) OrdersByParty(ctx context.Context, request *api.OrdersByPartyRequest) (*api.OrdersByPartyResponse, error) {
	if request.Party == "" {
		return nil, errors.New("Party empty or missing")
	}
	orderFilters := &filters.OrderQueryFilters{}
	if request.Params != nil && request.Params.Limit > 0 {
		orderFilters.Last = &request.Params.Limit
	}
	orders, err := h.OrderService.GetByParty(ctx, request.Party, orderFilters)
	if err != nil {
		return nil, err
	}
	var response = &api.OrdersByPartyResponse{}
	if len(orders) > 0 {
		response.Orders = orders
	}
	return response, nil
}

// Markets provides a list of all current markets that exist on the VEGA platform.
func (h *Handlers) Markets(ctx context.Context, request *api.MarketsRequest) (*api.MarketsResponse, error) {
	//markets, err := h.OrderService.GetMarkets(ctx)
	//if err != nil {
	//	return nil, err
	//}
	var markets = []string{"BTC/DEC18"}
	var response = &api.MarketsResponse{}
	if len(markets) > 0 {
		response.Markets = markets
	}
	return response, nil
}

// OrdersByMarketAndId searches for the given order by Id and Market. If found it will return
// an Order msg otherwise it will return an error.
func (h *Handlers) OrderByMarketAndId(ctx context.Context, request *api.OrderByMarketAndIdRequest) (*api.OrderByMarketAndIdResponse, error) {
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
// It will fill in any tradeless intervals with zero based candles. Since time period must be in RFC3339 string format.
func (h *Handlers) TradeCandles(ctx context.Context, request *api.TradeCandlesRequest) (*api.TradeCandlesResponse, error) {
	market := request.Market
	if market == "" {
		return nil, errors.New("Market empty or missing")
	}

	if request.Since == "" {
		return nil, errors.New("Since date is missing")
	}

	//since, err := time.Parse(time.RFC3339, request.Since)
	//if err != nil {
	//	return nil, err
	//}

	interval := request.Interval
	if interval < 2 {
		interval = 2
	}
	//res, err := h.TradeService.GetCandles(ctx, market, since, interval)
	//if err != nil {
	//	return nil, err
	//}
	var response = &api.TradeCandlesResponse{}
	//if len(res.Candles) > 0 {
	//	response.Candles = res.Candles
	//}
	return response, nil
}

func (h *Handlers) MarketDepth(ctx context.Context, request *api.MarketDepthRequest) (*api.MarketDepthResponse, error) {
	if request.Market == "" {
		return nil, errors.New("Market empty or missing")
	}
	// Query market depth statistics
	depth, err := h.OrderService.GetMarketDepth(ctx, request.Market)
	if err != nil {
		return nil, err
	}
	// Query last 1 trades from store
	queryFilters := &filters.TradeQueryFilters{}
	last := uint64(1)
	queryFilters.Last = &last
	trades, err := h.TradeService.GetByMarket(ctx, request.Market, queryFilters)
	if err != nil {
		return nil, err
	}
	// Build market depth response, including last trade (if available)
	var response = &api.MarketDepthResponse{}
	response.Buy = depth.Buy
	response.Name = depth.Name
	response.Sell = depth.Sell
	if trades != nil && trades[0] != nil {
	 	response.LastTrade = trades[0]
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

	filters := &filters.TradeQueryFilters{}
	*filters.Last = limit
	
	trades, err := h.TradeService.GetByMarket(ctx, request.Market, filters)
	if err != nil {
		return nil, err
	}
	var response = &api.TradesByMarketResponse{}
	response.Trades = trades
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

func (h *Handlers) Statistics(ctx context.Context, request *api.StatisticsRequest) (*msg.Statistics, error) {
	return h.OrderService.GetStatistics(ctx)
}

func (h *Handlers) GetVegaTime(ctx context.Context, request *api.VegaTimeRequest) (*api.VegaTimeResponse, error) {
	currentTime, _ := h.OrderService.GetCurrentTime(ctx)

	var response = &api.VegaTimeResponse{}
	response.Time = fmt.Sprintf("%s", currentTime.Format(time.RFC3339))
	return response, nil
}
