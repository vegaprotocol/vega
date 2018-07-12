package grpc

import (
	"context"
	"fmt"
	"vega/grpc"
	"vega/proto"
	"vega/api"
)

type Handlers struct {
	OrderService api.OrderService
	TradeService api.TradeService
}

func (h *Handlers) CreateOrder(ctx context.Context, order *msg.Order) (*grpc.OrderResponse, error) {
	fmt.Println(order.Market)

	success := true
	
	// Call into API Order service layer
	//success, err := h.OrderService.CreateOrder(ctx, *order)

	return &grpc.OrderResponse{Success: success}, nil
}
