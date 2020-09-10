package api

import (
	"context"
	"fmt"
	"net"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/plugins"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/vegatime"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

// GRPCServer represent the grpc api provided by the vega node
type GRPCServer struct {
	Config

	client BlockchainClient
	log    *logging.Logger
	srv    *grpc.Server
	stats  *stats.Stats

	accountsService         *accounts.Svc
	candleService           *candles.Svc
	marketService           MarketService
	orderService            *orders.Svc
	partyService            *parties.Svc
	timeService             *vegatime.Svc
	tradeService            *trades.Svc
	transferResponseService *transfers.Svc
	riskService             *risk.Svc
	governanceService       *governance.Svc
	notaryService           *notary.Svc
	evtfwd                  EvtForwarder
	assetService            *assets.Svc
	feeService              *fee.Svc
	eventService            *subscribers.Service
	withdrawalService       *plugins.Withdrawal

	tradingService     *tradingService
	tradingDataService *tradingDataService

	statusChecker *monitoring.Status

	// used in order to gracefully close streams
	ctx   context.Context
	cfunc context.CancelFunc
}

// NewGRPCServer create a new instance of the GPRC api for the vega node
func NewGRPCServer(
	log *logging.Logger,
	config Config,
	stats *stats.Stats,
	client BlockchainClient,
	timeService *vegatime.Svc,
	marketService MarketService,
	partyService *parties.Svc,
	orderService *orders.Svc,
	tradeService *trades.Svc,
	candleService *candles.Svc,
	accountsService *accounts.Svc,
	transferResponseService *transfers.Svc,
	riskService *risk.Svc,
	governanceService *governance.Svc,
	notaryService *notary.Svc,
	evtfwd EvtForwarder,
	assetService *assets.Svc,
	feeService *fee.Svc,
	eventService *subscribers.Service,
	withdrawalService *plugins.Withdrawal,
	statusChecker *monitoring.Status,
) *GRPCServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	ctx, cfunc := context.WithCancel(context.Background())

	return &GRPCServer{
		log:                     log,
		Config:                  config,
		stats:                   stats,
		client:                  client,
		orderService:            orderService,
		tradeService:            tradeService,
		candleService:           candleService,
		timeService:             timeService,
		marketService:           marketService,
		partyService:            partyService,
		accountsService:         accountsService,
		transferResponseService: transferResponseService,
		riskService:             riskService,
		governanceService:       governanceService,
		notaryService:           notaryService,
		evtfwd:                  evtfwd,
		assetService:            assetService,
		feeService:              feeService,
		eventService:            eventService,
		withdrawalService:       withdrawalService,
		statusChecker:           statusChecker,
		ctx:                     ctx,
		cfunc:                   cfunc,
	}
}

// ReloadConf update the internal configuration of the GRPC server
func (g *GRPCServer) ReloadConf(cfg Config) {
	g.log.Info("reloading configuration")
	if g.log.GetLevel() != cfg.Level.Get() {
		g.log.Info("updating log level",
			logging.String("old", g.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		g.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	g.Config = cfg
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

		ctx = contextutil.WithRemoteIPAddr(ctx, ip)

		// Calls the handler
		h, err := handler(ctx, req)

		log.Debug("Invoked RPC call",
			logging.String("method", info.FullMethod),
			logging.Error(err),
		)

		return h, err
	}
}

// Start start the grpc server
func (g *GRPCServer) Start() {

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
		log:               g.log,
		blockchain:        g.client,
		tradeOrderService: g.orderService,
		accountService:    g.accountsService,
		marketService:     g.marketService,
		governanceService: g.governanceService,
		evtForwarder:      g.evtfwd,
		statusChecker:     g.statusChecker,
	}
	g.tradingService = tradingSvc
	protoapi.RegisterTradingServer(g.srv, tradingSvc)

	tradingDataSvc := &tradingDataService{
		log:                     g.log,
		Config:                  g.Config,
		Stats:                   g.stats,
		Client:                  g.client,
		OrderService:            g.orderService,
		TradeService:            g.tradeService,
		CandleService:           g.candleService,
		MarketService:           g.marketService,
		PartyService:            g.partyService,
		TimeService:             g.timeService,
		AccountsService:         g.accountsService,
		TransferResponseService: g.transferResponseService,
		RiskService:             g.riskService,
		NotaryService:           g.notaryService,
		governanceService:       g.governanceService,
		AssetService:            g.assetService,
		FeeService:              g.feeService,
		eventService:            g.eventService,
		statusChecker:           g.statusChecker,
		WithdrawalService:       g.withdrawalService,
		ctx:                     g.ctx,
	}
	g.tradingDataService = tradingDataSvc
	protoapi.RegisterTradingDataServer(g.srv, tradingDataSvc)

	err = g.srv.Serve(lis)
	if err != nil {
		g.log.Panic("Failure serving gRPC API", logging.Error(err))
	}
}

// Stop stops the GRPC server
func (g *GRPCServer) Stop() {
	if g.srv != nil {
		g.log.Info("Stopping gRPC based API")
		g.cfunc()
		g.srv.GracefulStop()
	}
}
