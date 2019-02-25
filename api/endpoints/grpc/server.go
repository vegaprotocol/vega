package grpc

import (
	"fmt"
	"net"
	"vega/internal"

	"vega/api"
	"vega/internal/orders"
	"vega/internal/trades"
	"vega/internal/candles"
	"vega/internal/vegatime"
	"vega/internal/markets"

	"google.golang.org/grpc"
	"vega/internal/logging"
)

type grpcServer struct {
	*api.Config
	stats *internal.Stats
	orderService orders.Service
	tradeService trades.Service
	candleService candles.Service
	marketService markets.Service
	timeService vegatime.Service
}

func NewGRPCServer(config *api.Config, stats *internal.Stats, orderService orders.Service,
	tradeService trades.Service, candleService candles.Service) *grpcServer {
	return &grpcServer{
		Config: config,
		stats: stats,
		orderService: orderService,
		tradeService: tradeService,
		candleService: candleService,
	}
}

func (g *grpcServer) Start() {
	logger := *g.GetLogger()

	ip := g.GrpcServerIpAddress
	port := g.GrpcServerPort

	logger.Info("Starting gRPC based API", logging.String("addr", ip), logging.Int("port", port))
	
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		logger.Panic("Failure listening on gRPC port", logging.Int("port", port), logging.Error(err))
	}
	
	var handlers = &Handlers{
		Stats: g.stats,
		OrderService: g.orderService,
		TradeService: g.tradeService,
		CandleService: g.candleService,
		MarketService: g.marketService,
		TimeService: g.timeService,
	}
	grpcServer := grpc.NewServer()
	api.RegisterTradingServer(grpcServer, handlers)
	err = grpcServer.Serve(lis)
	if err != nil {
		logger.Panic("Failure serving gRPC API", logging.Error(err))
	}
}
