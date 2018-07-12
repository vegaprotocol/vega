package grpc

import (
	"fmt"
	"log"
	"net"
	"vega/api"

	"google.golang.org/grpc"
)

type grpcServer struct {
	orderService api.OrderService
	tradeService api.TradeService
}

func NewGRPCServer(orderService api.OrderService, tradeService api.TradeService) *grpcServer {
	return &grpcServer{
		orderService: orderService,
		tradeService: tradeService,
	}
}

func (g *grpcServer) Start() {
	var port = 3004
	fmt.Printf("Starting GRPC based server on port %d...\n", port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var handlers = &Handlers{
		OrderService: g.orderService,
		TradeService: g.tradeService,
	}
	grpcServer := grpc.NewServer()
	api.RegisterTradingServer(grpcServer, handlers)
	grpcServer.Serve(lis)
}
