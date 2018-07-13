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