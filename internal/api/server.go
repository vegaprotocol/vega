package api

import (
	"context"
	"fmt"
	"net"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/candles"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/markets"
	"code.vegaprotocol.io/vega/internal/monitoring"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/parties"
	"code.vegaprotocol.io/vega/internal/trades"
	"code.vegaprotocol.io/vega/internal/vegatime"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type grpcServer struct {
	log *logging.Logger
	Config
	stats         *internal.Stats
	client        *blockchain.Client
	orderService  *orders.Svc
	tradeService  *trades.Svc
	candleService *candles.Svc
	marketService *markets.Svc
	partyService  *parties.Svc
	timeService   *vegatime.Svc
	srv           *grpc.Server
	statusChecker *monitoring.Status

	// used in order to gracefully close streams
	ctx   context.Context
	cfunc context.CancelFunc
}

func NewGRPCServer(
	log *logging.Logger,
	config Config,
	stats *internal.Stats,
	client *blockchain.Client,
	timeService *vegatime.Svc,
	marketService *markets.Svc,
	partyService *parties.Svc,
	orderService *orders.Svc,
	tradeService *trades.Svc,
	candleService *candles.Svc,
	statusChecker *monitoring.Status,
) *grpcServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	ctx, cfunc := context.WithCancel(context.Background())

	return &grpcServer{
		log:           log,
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
		ctx:           ctx,
		cfunc:         cfunc,
	}
}

func (s *grpcServer) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	s.Config = cfg
}

func remoteAddrInterceptor(log *logging.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {

		// first check if the request is forwarded from our restproxy
		// get the metadata
		var ip string
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			forwardedFor, ok := md["x-forwarded-for"]
			if ok && len(forwardedFor) > 0 {
				log.Debug("grpc request x-forwarded-for",
					logging.String("method", info.FullMethod),
					logging.String("remote-ip-addr", forwardedFor[0]),
				)
				ip = forwardedFor[0]
			}
		}

		// if the request is not forwarded let's get it from the peer infos
		if len(ip) <= 0 {
			p, ok := peer.FromContext(ctx)
			if ok && p != nil {
				log.Debug("grpc peer client request",
					logging.String("method", info.FullMethod),
					logging.String("remote-ip-addr", p.Addr.String()))
				ip = p.Addr.String()
			}
		}

		ctx = context.WithValue(ctx, "remote-ip-addr", ip)

		// Calls the handler
		h, err := handler(ctx, req)

		log.Debug("Invoked RPC call",
			logging.String("method", info.FullMethod),
			logging.Error(err),
		)

		return h, err
	}
}

func (g *grpcServer) Start() {

	ip := g.IP
	port := g.Port

	g.log.Info("Starting gRPC based API", logging.String("addr", ip), logging.Int("port", port))

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		g.log.Panic("Failure listening on gRPC port", logging.Int("port", port), logging.Error(err))
	}

	intercept := grpc.UnaryInterceptor(remoteAddrInterceptor(g.log))
	g.srv = grpc.NewServer(intercept)

	tradingSvc := &tradingService{
		tradeOrderService: g.orderService,
		statusChecker:     g.statusChecker,
	}
	protoapi.RegisterTradingServer(g.srv, tradingSvc)

	tradingDataSvc := &tradingDataService{
		log:           g.log,
		Config:        g.Config,
		Stats:         g.stats,
		Client:        g.client,
		OrderService:  g.orderService,
		TradeService:  g.tradeService,
		CandleService: g.candleService,
		MarketService: g.marketService,
		PartyService:  g.partyService,
		TimeService:   g.timeService,
		statusChecker: g.statusChecker,
		ctx:           g.ctx,
	}
	protoapi.RegisterTradingDataServer(g.srv, tradingDataSvc)

	err = g.srv.Serve(lis)
	if err != nil {
		g.log.Panic("Failure serving gRPC API", logging.Error(err))
	}
}

func (g *grpcServer) Stop() {
	if g.srv != nil {
		g.log.Info("Stopping gRPC based API")
		g.cfunc()
		g.srv.GracefulStop()
	}
}
