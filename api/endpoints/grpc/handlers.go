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

func (h *Handlers) GetOpenOrdersByMarket(ctx context.Context, request *api.GetOpenOrdersByMarketRequest) (r *api.GetOpenOrdersByMarketResponse, err error) {
	orders, err := h.OrderService.GetOpenOrdersByMarket(ctx, request.Market, request.Params.Limit)
	if err != nil {
		return nil, err
	}
	r.Orders = orders
	return r, nil
}

func (h *Handlers) GetOpenOrdersByParty(ctx context.Context, request *api.GetOpenOrdersByPartyRequest) (r *api.GetOpenOrdersByPartyResponse, err error) {
	orders, err := h.OrderService.GetOpenOrdersByMarket(ctx, request.Party, request.Params.Limit)
	if err != nil {
		return nil, err
	}
	r.Orders = orders
	return r, nil
}


func (h *Handlers) GetMarkets(ctx context.Context, request *api.GetMarketsRequest) (r *api.GetMarketsResponse, err error) {
	markets, err := h.OrderService.GetMarkets(ctx)
	if err != nil {
		return nil, err
	}
	r.Markets = markets
	return r, nil
}

func (h *Handlers) GetOrderByMarketAndId(ctx context.Context, request *api.GetOrderByMarketAndIdRequest) (r *api.GetOrderByMarketAndIdResponse, err error) {
	order, err := h.OrderService.GetByMarketAndId(ctx, request.Market, request.Id)
	if err != nil {
		return nil, err
	}
	r.Order = order
	return r, nil
}