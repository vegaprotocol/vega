package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"code.vegaprotocol.io/data-node/candlesv2"
	"code.vegaprotocol.io/data-node/service"

	"code.vegaprotocol.io/data-node/accounts"
	"code.vegaprotocol.io/data-node/assets"
	"code.vegaprotocol.io/data-node/candles"
	"code.vegaprotocol.io/data-node/checkpoint"
	"code.vegaprotocol.io/data-node/contextutil"
	"code.vegaprotocol.io/data-node/delegations"
	"code.vegaprotocol.io/data-node/epochs"
	"code.vegaprotocol.io/data-node/fee"
	"code.vegaprotocol.io/data-node/governance"
	"code.vegaprotocol.io/data-node/liquidity"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/netparams"
	"code.vegaprotocol.io/data-node/nodes"
	"code.vegaprotocol.io/data-node/notary"
	"code.vegaprotocol.io/data-node/oracles"
	"code.vegaprotocol.io/data-node/orders"
	"code.vegaprotocol.io/data-node/parties"
	"code.vegaprotocol.io/data-node/plugins"
	"code.vegaprotocol.io/data-node/risk"
	"code.vegaprotocol.io/data-node/staking"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"
	"github.com/fullstorydev/grpcui/standalone"
	"golang.org/x/sync/errgroup"

	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	protoapi2 "code.vegaprotocol.io/protos/data-node/api/v2"
	vegaprotoapi "code.vegaprotocol.io/protos/vega/api/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
)

// GRPCServer represent the grpc api provided by the vega node
type GRPCServer struct {
	Config
	useSQLStores          bool
	log                   *logging.Logger
	srv                   *grpc.Server
	vegaCoreServiceClient CoreServiceClient

	accountsService         *accounts.Svc
	candleService           *candles.Svc
	marketService           MarketService
	orderService            *orders.Svc
	liquidityService        *liquidity.Svc
	partyService            *parties.Svc
	timeService             *vegatime.Svc
	tradeService            *trades.Svc
	transferResponseService *transfers.Svc
	riskService             *risk.Svc
	governanceService       *governance.Svc
	notaryService           *notary.Svc
	assetService            *assets.Svc
	feeService              *fee.Svc
	eventService            *subscribers.Service
	withdrawalService       *plugins.Withdrawal
	depositService          *plugins.Deposit
	netParamsService        *netparams.Service
	oracleService           *oracles.Service
	stakingService          *staking.Service
	coreProxySvc            *coreProxyService
	tradingDataService      protoapi.TradingDataServiceServer
	nodeService             *nodes.Service
	epochService            *epochs.Service
	delegationService       *delegations.Service
	rewardsService          *subscribers.RewardCounters
	checkpointSvc           *checkpoint.Svc

	marketDepthService *subscribers.MarketDepthBuilder

	orderServiceV2              *service.Order
	candleServiceV2             *candlesv2.Svc
	networkLimitsServiceV2      *service.NetworkLimits
	marketDataServiceV2         *service.MarketData
	tradeServiceV2              *service.Trade
	assetServiceV2              *service.Asset
	accountServiceV2            *service.Account
	rewardServiceV2             *service.Reward
	marketsServiceV2            *service.Markets
	delegationServiceV2         *service.Delegation
	epochServiceV2              *service.Epoch
	depositServiceV2            *service.Deposit
	withdrawalServiceV2         *service.Withdrawal
	governanceServiceV2         *service.Governance
	riskFactorServiceV2         *service.RiskFactor
	riskServiceV2               *service.Risk
	networkParameterServiceV2   *service.NetworkParameter
	blockServiceV2              *service.Block
	partyServiceV2              *service.Party
	checkpointServiceV2         *service.Checkpoint
	oracleSpecServiceV2         *service.OracleSpec
	oracleDataServiceV2         *service.OracleData
	liquidityProvisionServiceV2 *service.LiquidityProvision
	positionServiceV2           *service.Position
	transferServiceV2           *service.Transfer
	stakeLinkingServiceV2       *service.StakeLinking
	notaryServiceV2             *service.Notary
	multiSigServiceV2           *service.MultiSig
	keyRotationServiceV2        *service.KeyRotations
	nodeServiceV2               *service.Node
	marketDepthServiceV2        *service.MarketDepth
	ledgerServiceV2             *service.Ledger

	eventObserver *eventObserver

	// used in order to gracefully close streams
	ctx   context.Context
	cfunc context.CancelFunc
}

// NewGRPCServer create a new instance of the GPRC api for the vega node
func NewGRPCServer(
	log *logging.Logger,
	config Config,
	useSQLStores bool,
	coreServiceClient CoreServiceClient,
	timeService *vegatime.Svc,
	marketService MarketService,
	partyService *parties.Svc,
	orderService *orders.Svc,
	liquidityService *liquidity.Svc,
	tradeService *trades.Svc,
	candleService *candles.Svc,
	accountsService *accounts.Svc,
	transferResponseService *transfers.Svc,
	riskService *risk.Svc,
	governanceService *governance.Svc,
	notaryService *notary.Svc,
	assetService *assets.Svc,
	feeService *fee.Svc,
	eventService *subscribers.Service,
	oracleService *oracles.Service,
	withdrawalService *plugins.Withdrawal,
	depositService *plugins.Deposit,
	marketDepthService *subscribers.MarketDepthBuilder,
	netParamsService *netparams.Service,
	nodeService *nodes.Service,
	epochService *epochs.Service,
	delegationService *delegations.Service,
	rewardsService *subscribers.RewardCounters,
	stakingService *staking.Service,
	checkpointSvc *checkpoint.Svc,
	orderServiceV2 *service.Order,
	networkLimitsServiceV2 *service.NetworkLimits,
	marketDataServiceV2 *service.MarketData,
	tradeServiceV2 *service.Trade,
	assetServiceV2 *service.Asset,
	accountServiceV2 *service.Account,
	rewardServiceV2 *service.Reward,
	marketsServiceV2 *service.Markets,
	delegationServiceV2 *service.Delegation,
	epochServiceV2 *service.Epoch,
	depositServiceV2 *service.Deposit,
	withdrawalServiceV2 *service.Withdrawal,
	governanceServiceV2 *service.Governance,
	riskFactorServiceV2 *service.RiskFactor,
	riskServiceV2 *service.Risk,
	networkParameterServiceV2 *service.NetworkParameter,
	blockServiceV2 *service.Block,
	checkpointServiceV2 *service.Checkpoint,
	partyServiceV2 *service.Party,
	candleServiceV2 *candlesv2.Svc,
	oracleSpecServiceV2 *service.OracleSpec,
	oracleDataServiceV2 *service.OracleData,
	liquidityProvisionServiceV2 *service.LiquidityProvision,
	positionServiceV2 *service.Position,
	transferServiceV2 *service.Transfer,
	stakeLinkingServiceV2 *service.StakeLinking,
	notaryServiceV2 *service.Notary,
	multiSigServiceV2 *service.MultiSig,
	keyRotationServiceV2 *service.KeyRotations,
	nodeServiceV2 *service.Node,
	marketDepthServiceV2 *service.MarketDepth,
	ledgerServiceV2 *service.Ledger,
) *GRPCServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	ctx, cfunc := context.WithCancel(context.Background())

	return &GRPCServer{
		log:                     log,
		Config:                  config,
		useSQLStores:            useSQLStores,
		vegaCoreServiceClient:   coreServiceClient,
		orderService:            orderService,
		liquidityService:        liquidityService,
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
		assetService:            assetService,
		feeService:              feeService,
		eventService:            eventService,
		withdrawalService:       withdrawalService,
		depositService:          depositService,
		marketDepthService:      marketDepthService,
		netParamsService:        netParamsService,
		oracleService:           oracleService,
		nodeService:             nodeService,
		epochService:            epochService,
		delegationService:       delegationService,
		rewardsService:          rewardsService,
		stakingService:          stakingService,
		checkpointSvc:           checkpointSvc,

		orderServiceV2:              orderServiceV2,
		networkLimitsServiceV2:      networkLimitsServiceV2,
		tradeServiceV2:              tradeServiceV2,
		assetServiceV2:              assetServiceV2,
		accountServiceV2:            accountServiceV2,
		rewardServiceV2:             rewardServiceV2,
		marketsServiceV2:            marketsServiceV2,
		delegationServiceV2:         delegationServiceV2,
		epochServiceV2:              epochServiceV2,
		depositServiceV2:            depositServiceV2,
		withdrawalServiceV2:         withdrawalServiceV2,
		multiSigServiceV2:           multiSigServiceV2,
		governanceServiceV2:         governanceServiceV2,
		riskFactorServiceV2:         riskFactorServiceV2,
		networkParameterServiceV2:   networkParameterServiceV2,
		blockServiceV2:              blockServiceV2,
		checkpointServiceV2:         checkpointServiceV2,
		partyServiceV2:              partyServiceV2,
		candleServiceV2:             candleServiceV2,
		oracleSpecServiceV2:         oracleSpecServiceV2,
		oracleDataServiceV2:         oracleDataServiceV2,
		liquidityProvisionServiceV2: liquidityProvisionServiceV2,
		positionServiceV2:           positionServiceV2,
		transferServiceV2:           transferServiceV2,
		stakeLinkingServiceV2:       stakeLinkingServiceV2,
		notaryServiceV2:             notaryServiceV2,
		keyRotationServiceV2:        keyRotationServiceV2,
		nodeServiceV2:               nodeServiceV2,
		marketDepthServiceV2:        marketDepthServiceV2,
		riskServiceV2:               riskServiceV2,
		marketDataServiceV2:         marketDataServiceV2,
		ledgerServiceV2:             ledgerServiceV2,

		eventObserver: &eventObserver{
			log:          log,
			eventService: eventService,
			Config:       config,
		},
		ctx:   ctx,
		cfunc: cfunc,
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

func (g *GRPCServer) getTCPListener() (net.Listener, error) {
	ip := g.IP
	port := strconv.Itoa(g.Port)

	g.log.Info("Starting gRPC based API", logging.Bool("v1 API using sql stores", g.useSQLStores), logging.String("addr", ip), logging.String("port", port))

	tpcLis, err := net.Listen("tcp", net.JoinHostPort(ip, port))
	if err != nil {
		return nil, err
	}

	return tpcLis, nil
}

// Start start the grpc server.
// Uses default TCP listener if no provided.
func (g *GRPCServer) Start(ctx context.Context, lis net.Listener) error {
	if lis == nil {
		tpcLis, err := g.getTCPListener()
		if err != nil {
			return err
		}

		lis = tpcLis
	}

	intercept := grpc.UnaryInterceptor(remoteAddrInterceptor(g.log))
	g.srv = grpc.NewServer(intercept)

	coreProxySvc := &coreProxyService{
		log:               g.log,
		conf:              g.Config,
		coreServiceClient: g.vegaCoreServiceClient,
		eventObserver:     g.eventObserver,
	}
	g.coreProxySvc = coreProxySvc
	vegaprotoapi.RegisterCoreServiceServer(g.srv, coreProxySvc)

	tradingDataSvc := &tradingDataService{
		log:                     g.log,
		Config:                  g.Config,
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
		WithdrawalService:       g.withdrawalService,
		DepositService:          g.depositService,
		MarketDepthService:      g.marketDepthService,
		NetParamsService:        g.netParamsService,
		LiquidityService:        g.liquidityService,
		oracleService:           g.oracleService,
		nodeService:             g.nodeService,
		epochService:            g.epochService,
		delegationService:       g.delegationService,
		rewardsService:          g.rewardsService,
		stakingService:          g.stakingService,
		checkpointService:       g.checkpointSvc,
	}
	if g.useSQLStores {
		g.tradingDataService = &tradingDataDelegator{
			log:          g.log,
			Config:       g.Config,
			eventService: g.eventService,
			// tradingDataService:          tradingDataSvc,
			orderServiceV2:              g.orderServiceV2,
			tradeServiceV2:              g.tradeServiceV2,
			assetServiceV2:              g.assetServiceV2,
			accountServiceV2:            g.accountServiceV2,
			rewardServiceV2:             g.rewardServiceV2,
			marketServiceV2:             g.marketsServiceV2,
			delegationServiceV2:         g.delegationServiceV2,
			epochServiceV2:              g.epochServiceV2,
			depositServiceV2:            g.depositServiceV2,
			withdrawalServiceV2:         g.withdrawalServiceV2,
			governanceServiceV2:         g.governanceServiceV2,
			riskFactorServiceV2:         g.riskFactorServiceV2,
			networkParameterServiceV2:   g.networkParameterServiceV2,
			blockServiceV2:              g.blockServiceV2,
			checkpointServiceV2:         g.checkpointServiceV2,
			partyServiceV2:              g.partyServiceV2,
			candleServiceV2:             g.candleServiceV2,
			oracleSpecServiceV2:         g.oracleSpecServiceV2,
			oracleDataServiceV2:         g.oracleDataServiceV2,
			liquidityProvisionServiceV2: g.liquidityProvisionServiceV2,
			positionServiceV2:           g.positionServiceV2,
			transferServiceV2:           g.transferServiceV2,
			stakeLinkingServiceV2:       g.stakeLinkingServiceV2,
			notaryServiceV2:             g.notaryServiceV2,
			keyRotationServiceV2:        g.keyRotationServiceV2,
			nodeServiceV2:               g.nodeServiceV2,
			marketDepthService:          g.marketDepthServiceV2,
			riskServiceV2:               g.riskServiceV2,
			marketDataServiceV2:         g.marketDataServiceV2,
			ledgerServiceV2:             g.ledgerServiceV2,
		}
	} else {
		g.tradingDataService = tradingDataSvc
	}

	protoapi.RegisterTradingDataServiceServer(g.srv, g.tradingDataService)

	tradingDataSvcV2 := &tradingDataServiceV2{
		log:                  g.log,
		v2ApiEnabled:         g.useSQLStores,
		orderService:         g.orderServiceV2,
		networkLimitsService: g.networkLimitsServiceV2,
		marketDataService:    g.marketDataServiceV2,
		tradeService:         g.tradeServiceV2,
		multiSigService:      g.multiSigServiceV2,
		notaryService:        g.notaryServiceV2,
		assetService:         g.assetServiceV2,
		candleService:        g.candleServiceV2,
		marketsService:       g.marketsServiceV2,
		partyService:         g.partyServiceV2,
		riskService:          g.riskServiceV2,
		accountService:       g.accountServiceV2,
	}
	protoapi2.RegisterTradingDataServiceServer(g.srv, tradingDataSvcV2)

	eg, ctx := errgroup.WithContext(ctx)

	if g.Reflection || g.WebUIEnabled {
		reflection.Register(g.srv)
	}

	eg.Go(func() error {
		<-ctx.Done()
		g.stop()
		return ctx.Err()
	})

	eg.Go(func() error {
		return g.srv.Serve(lis)
	})

	if g.WebUIEnabled {
		g.startWebUI(ctx)
	}

	return eg.Wait()
}

func (g *GRPCServer) stop() {
	if g.srv == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		g.log.Info("Gracefully stopping gRPC based API")
		g.srv.GracefulStop()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		g.log.Info("Force stopping gRPC based API")
		g.srv.Stop()
	}
}

func (g *GRPCServer) startWebUI(ctx context.Context) {
	cc, err := grpc.Dial(fmt.Sprintf("127.0.0.1:%d", g.Port), grpc.WithInsecure())
	if err != nil {
		g.log.Error("failed to create client to local grpc server", logging.Error(err))
		return
	}

	uiHandler, err := standalone.HandlerViaReflection(ctx, cc, "vega data node")
	if err != nil {
		g.log.Error("failed to create grpc-ui server", logging.Error(err))
		return
	}

	uiListener, err := net.Listen("tcp", net.JoinHostPort(g.IP, strconv.Itoa(g.WebUIPort)))
	if err != nil {
		g.log.Error("failed to open listen socket on port", logging.Int("port", g.WebUIPort), logging.Error(err))
		return
	}

	g.log.Info("Starting gRPC Web UI", logging.String("addr", g.IP), logging.Int("port", g.WebUIPort))
	go http.Serve(uiListener, uiHandler)
}
