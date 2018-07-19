package grpc

import (
	"context"
	"vega/proto"
	"vega/api"
	"time"
	"github.com/pkg/errors"
)

type Handlers struct {
	OrderService api.OrderService
	TradeService api.TradeService
}

const limitMax = uint64(10000)

func (h *Handlers) CreateOrder(ctx context.Context, order *msg.Order) (*api.OrderResponse, error) {
	success, err := h.OrderService.CreateOrder(ctx, order)
	return &api.OrderResponse{Success: success}, err
}

func (h *Handlers) OrdersByMarket(ctx context.Context, request *api.OrdersByMarketRequest) (*api.OrdersByMarketResponse, error) {
	market := request.Market
	if market == "" {
		return nil, errors.New("Market empty or missing")
	}
	limit := limitMax
	if request.Params != nil && request.Params.Limit > 0 {
		limit = request.Params.Limit
	}
	orders, err := h.OrderService.GetByMarket(ctx, market, limit)
	if err != nil {
		return nil, err
	}
	var response = &api.OrdersByMarketResponse{}
	if len(orders) > 0 {
	   response.Orders = orders
	}
	return response, nil
}

func (h *Handlers) OrdersByParty(ctx context.Context, request *api.OrdersByPartyRequest) (*api.OrdersByPartyResponse, error) {
	party := request.Party
	if party == "" {
		return nil, errors.New("Party empty or missing")
	}
	limit := limitMax
	if request.Params != nil && request.Params.Limit > 0 {
		limit = request.Params.Limit
	}
	orders, err := h.OrderService.GetByParty(ctx, party, limit)
	if err != nil {
		return nil, err
	}
	var response = &api.OrdersByPartyResponse{}
	if len(orders) > 0 {
		response.Orders = orders
	}
	return response, nil
}

func (h *Handlers) Markets(ctx context.Context, request *api.MarketsRequest) (*api.MarketsResponse, error) {
	markets, err := h.OrderService.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}
	var response = &api.MarketsResponse{}
	if len(markets) > 0 {
		response.Markets = markets
	}
	return response, nil
}

func (h *Handlers) OrderByMarketAndId(ctx context.Context, request *api.OrderByMarketAndIdRequest) (*api.OrderByMarketAndIdResponse, error) {
	order, err := h.OrderService.GetByMarketAndId(ctx, request.Market, request.Id)
	if err != nil {
		return nil, err
	}
	var response = &api.OrderByMarketAndIdResponse{}
	response.Order = order
	return response, nil
}

func (h *Handlers) TradeCandles(ctx context.Context, request *api.TradeCandlesRequest) (*api.TradeCandlesResponse, error) {
	market := request.Market
	if market == "" {
		market = "BTC/DEC18"
	}
	sinceStr := request.Since
	if sinceStr == "" {
		market = "2018-07-09T12:00:00Z"
	}
	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		return nil, err
	}
	interval := request.Interval
	if interval < 1 {
		interval = 2
	}
	res, err := h.TradeService.GetCandles(ctx, market, since, interval)
	if err != nil {
		return nil, err
	}
	var response = &api.TradeCandlesResponse{}
	if len(res.Candles) > 0 {
		response.Candles = res.Candles
	}
	return response, nil

}