package grpc

import (
	"code.vegaprotocol.io/vega/internal/parties"
	"fmt"
	"net"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/monitoring"
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
	partyService  parties.Service
	timeService   vegatime.Service
	srv           *grpc.Server
	statusChecker *monitoring.Status
}

func NewGRPCServer(
	config *api.Config,
	stats *internal.Stats,
	client blockchain.Client,
	timeService vegatime.Service,
	marketService markets.Service,
	partyService parties.Service,
	orderService orders.Service,
	tradeService trades.Service,
	candleService candles.Service,
	statusChecker *monitoring.Status,
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
		partyService:  partyService,
		statusChecker: statusChecker,
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
		PartyService:  g.partyService,
		TimeService:   g.timeService,
		statusChecker: g.statusChecker,
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
