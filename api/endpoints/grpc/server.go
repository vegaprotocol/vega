package grpc

import (
	"fmt"
	"net"
	"vega/api"
	"vega/log"
	"google.golang.org/grpc"
)

type grpcServer struct {
	orderService api.OrderService
	tradeService api.TradeService
	candleService api.CandleService
}

func NewGRPCServer(orderService api.OrderService, tradeService api.TradeService, candleService api.CandleService) *grpcServer {
	return &grpcServer{
		orderService: orderService,
		tradeService: tradeService,
		candleService: candleService,
	}
}

func (g *grpcServer) Start() {
	var port = 3002
	log.Infof("Starting GRPC based server on port %d...\n", port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var handlers = &Handlers{
		OrderService: g.orderService,
		TradeService: g.tradeService,
		CandleService: g.candleService,
	}
	grpcServer := grpc.NewServer()
	api.RegisterTradingServer(grpcServer, handlers)
	grpcServer.Serve(lis)
}
