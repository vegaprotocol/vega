package grpc

import (
	"context"
	"vega/proto"
	"vega/api"
)

type Handlers struct {
	OrderService api.OrderService
	TradeService api.TradeService
}

func (h *Handlers) CreateOrder(ctx context.Context, order *msg.Order) (*api.OrderResponse, error) {
	// Call into API Order service layer
	success, err := h.OrderService.CreateOrder(ctx, order)
	return &api.OrderResponse{Success: success}, err
}

func (h *Handlers) OrdersByMarket(ctx context.Context, request *api.OrdersByMarketRequest) (r *api.OrdersByMarketResponse, err error) {
	orders, err := h.OrderService.GetByMarket(ctx, request.Market, request.Params.Limit)
	if err != nil {
		return nil, err
	}
	r.Orders = orders
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