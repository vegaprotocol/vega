package grpc

import (
	"fmt"
	"net"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/appstatus"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type grpcServer struct {
	*api.Config
	stats         *internal.Stats
	client        blockchain.Client
	orderService  orders.Service
	tradeService  trades.Service
	candleService candles.Service
	marketService markets.Service
	timeService   vegatime.Service
	srv           *grpc.Server
	appst         *appstatus.AppStatus
}

func NewGRPCServer(
	config *api.Config,
	stats *internal.Stats,
	client blockchain.Client,
	timeService vegatime.Service,
	marketService markets.Service,
	orderService orders.Service,
	tradeService trades.Service,
	candleService candles.Service,
	appst *appstatus.AppStatus,
) *grpcServer {

	return &grpcServer{
		Config:        config,
		stats:         stats,
		client:        client,
		orderService:  orderService,
		tradeService:  tradeService,
		candleService: candleService,
		timeService:   timeService,
		marketService: marketService,
		appst:         appst,
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
		Stats:         g.stats,
		Client:        g.client,
		OrderService:  g.orderService,
		TradeService:  g.tradeService,
		CandleService: g.candleService,
		MarketService: g.marketService,
		TimeService:   g.timeService,
		appst:         g.appst,
	}
	g.srv = grpc.NewServer()
	api.RegisterTradingServer(g.srv, handlers)
	err = g.srv.Serve(lis)
	if err != nil {
		logger.Panic("Failure serving gRPC API", logging.Error(err))
	}
}

func (g *grpcServer) Stop() error {
	if g.srv != nil {
		g.srv.GracefulStop()
		return nil
	}
	return errors.New("GRPC server not started")
}
