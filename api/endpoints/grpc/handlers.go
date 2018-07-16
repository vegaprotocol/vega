package grpc

import (
	"context"
	"vega/msg"
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

func (h *Handlers) GetOrderBookDepth(ctx context.Context, market *msg.Market) (orderBookDepth *msg.OrderBookDepth, err error) {
	return h.OrderService.GetOrderBookDepthChart(ctx, market.Name)
}