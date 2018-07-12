package grpc

import (
	"context"
	"fmt"
	"vega/proto"
	"vega/api"
)

type Handlers struct {
	OrderService api.OrderService
	TradeService api.TradeService
}

func (h *Handlers) CreateOrder(ctx context.Context, order *msg.Order) (*api.OrderResponse, error) {
	fmt.Println(order.Market)
	
	// Call into API Order service layer
	success, err := h.OrderService.CreateOrder(ctx, *order)

	return &api.OrderResponse{Success: success}, err
}
