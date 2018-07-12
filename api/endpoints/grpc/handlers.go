package grpc

import (
	"context"
	"fmt"
	"vega/api"
	"vega/proto"
)

type Handlers struct {
	OrderService api.OrderService
	TradeService api.TradeService
}

func (h *Handlers) CreateOrder(ctx context.Context, order *msg.Order) (*api.OrderResponse, error) {
	fmt.Println(order.Market)

	success := true
	
	// Call into API Order service layer
	//success, err := h.OrderService.CreateOrder(ctx, *order)

	return &api.OrderResponse{Success: success}, nil
}
