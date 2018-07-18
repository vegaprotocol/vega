package grpc

import (
	"context"
	"vega/proto"
	"vega/api"
	"time"
	"fmt"
)

type Handlers struct {
	OrderService api.OrderService
	TradeService api.TradeService
}

const LIMIT_MAX = uint64(10000)

func (h *Handlers) CreateOrder(ctx context.Context, order *msg.Order) (*api.OrderResponse, error) {
	// Call into API Order service layer
	success, err := h.OrderService.CreateOrder(ctx, order)
	return &api.OrderResponse{Success: success}, err
}

func (h *Handlers) OrdersByMarket(ctx context.Context, request *api.OrdersByMarketRequest) (r *api.OrdersByMarketResponse, err error) {
	market := request.Market
	if market == "" {
		market = "BTC/DEC18"
	}
	limit := LIMIT_MAX
	if request.Params != nil && request.Params.Limit > 0 {
		limit = request.Params.Limit
	}

	fmt.Println(market, limit)

	orders, err := h.OrderService.GetByMarket(ctx, market, limit)
	if err != nil {
		return nil, err
	}
	fmt.Println("orders len: ", len(orders))
	if orders != nil && len(orders) > 0 {
		r.Orders = orders
	}

	return r, nil
}

func (h *Handlers) OrdersByParty(ctx context.Context, request *api.OrdersByPartyRequest) (r *api.OrdersByPartyResponse, err error) {
	orders, err := h.OrderService.GetByParty(ctx, request.Party, request.Params.Limit)
	if err != nil {
		return nil, err
	}
	r.Orders = orders
	return r, nil
}

func (h *Handlers) Markets(ctx context.Context, request *api.MarketsRequest) (r *api.MarketsResponse, err error) {
	markets, err := h.OrderService.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}
	r.Markets = markets
	return r, nil
}

func (h *Handlers) OrderByMarketAndId(ctx context.Context, request *api.OrderByMarketAndIdRequest) (r *api.OrderByMarketAndIdResponse, err error) {
	order, err := h.OrderService.GetByMarketAndId(ctx, request.Market, request.Id)
	if err != nil {
		return nil, err
	}
	r.Order = order
	return r, nil
}

func (h *Handlers) TradeCandles(ctx context.Context, request *api.TradeCandlesRequest) (r *api.TradeCandlesResponse, err error) {
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
	r.Candles = res.Candles
	return r, nil

}