package grpc

import (
	"fmt"
	"net"
	
	"vega/api"
	"vega/internal/orders"
	"vega/internal/trades"
	"vega/internal/candles"
	"vega/internal/vegatime"
	"vega/internal/markets"

	"google.golang.org/grpc"
)

type grpcServer struct {
	*api.Config
	orderService orders.Service
	tradeService trades.Service
	candleService candles.Service
	marketService markets.Service
	timeService vegatime.Service
}

func NewGRPCServer(config *api.Config, orderService orders.Service,
	tradeService trades.Service, candleService candles.Service) *grpcServer {
	return &grpcServer{
		Config: config,
		orderService: orderService,
		tradeService: tradeService,
		candleService: candleService,
	}
}

func (g *grpcServer) Start() {
	logger := *g.GetLogger()
	port := g.GrpcServerPort
	ip := g.GrpcServerIpAddress
	logger.Infof("Starting GRPC based server on port %d...\n", port)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}

	var handlers = &Handlers{
		OrderService: g.orderService,
		TradeService: g.tradeService,
		CandleService: g.candleService,
		MarketService: g.marketService,
		TimeService: g.timeService,
	}
	grpcServer := grpc.NewServer()
	api.RegisterTradingServer(grpcServer, handlers)
	grpcServer.Serve(lis)
}
