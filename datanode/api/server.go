// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/contextutil"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/ratelimit"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/libs/subscribers"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/fullstorydev/grpcui/standalone"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// EventService ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api EventService
type EventService interface {
	ObserveEvents(ctx context.Context, retries int, eTypes []events.Type, batchSize int, filters ...subscribers.EventFilter) (<-chan []*eventspb.BusEvent, chan<- int)
}

// BlockService ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/block_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api BlockService
type BlockService interface {
	GetLastBlock(ctx context.Context) (entities.Block, error)
}

// NetworkHistoryService ...
//
// it would be nice to use go:generate go run github.com/golang/mock/mockgen -destination mocks/networkhistory_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api NetworkHistoryService
// however it currently can't handle generic arguments and the generated code is not compilable without a bit of manual tweaking.
type NetworkHistoryService interface {
	GetHighestBlockHeightHistorySegment() (segment.Full, error)
	ListAllHistorySegments() (segment.Segments[segment.Full], error)
	FetchHistorySegment(ctx context.Context, historySegmentID string) (segment.Full, error)
	GetActivePeerIPAddresses() []string
	CopyHistorySegmentToFile(ctx context.Context, historySegmentID string, outFile string) error
	GetHistorySegmentReader(ctx context.Context, historySegmentID string) (io.ReadSeekCloser, int64, error)
	GetSwarmKeySeed() string
	GetConnectedPeerAddresses() ([]string, error)
	GetIpfsAddress() (string, error)
	GetSwarmKey() string
	GetBootstrapPeers() []string
}

// MarketsService ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/markets_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api MarketsService
type MarketsService interface {
	GetByID(ctx context.Context, marketID string) (entities.Market, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Market, error)
	GetAllPaged(ctx context.Context, marketID string, pagination entities.CursorPagination, includeSettled bool) ([]entities.Market, entities.PageInfo, error)
	ListSuccessorMarkets(ctx context.Context, marketID string, childrenOnly bool, pagination entities.CursorPagination) ([]entities.SuccessorMarket, entities.PageInfo, error)
}

// MarketDataService ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_data_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api MarketDataService
type MarketDataService interface {
	GetMarketDataByID(ctx context.Context, marketID string) (entities.MarketData, error)
	GetMarketsData(ctx context.Context) ([]entities.MarketData, error)
	GetHistoricMarketData(ctx context.Context, marketID string, start, end *time.Time, pagination entities.Pagination) ([]entities.MarketData, entities.PageInfo, error)
	ObserveMarketData(ctx context.Context, retries int, marketID []string) (<-chan []*entities.MarketData, uint64)
}

// RiskFactorService ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_factor_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api RiskFactorService
type RiskFactorService interface {
	GetMarketRiskFactors(ctx context.Context, marketID string) (entities.RiskFactor, error)
}

// Asset service ...
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/asset_service_mock.go -package mocks code.vegaprotocol.io/vega/datanode/api AssetService
type AssetService interface {
	GetByID(ctx context.Context, id string) (entities.Asset, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.Asset, error)
	GetAll(ctx context.Context) ([]entities.Asset, error)
	GetAllWithCursorPagination(ctx context.Context, pagination entities.CursorPagination) ([]entities.Asset, entities.PageInfo, error)
}

// GRPCServer represent the grpc api provided by the vega node.
type GRPCServer struct {
	Config
	log                   *logging.Logger
	srv                   *grpc.Server
	vegaCoreServiceClient CoreServiceClient

	eventService                  *subscribers.Service
	coreProxySvc                  *coreProxyService
	orderService                  *service.Order
	candleService                 *candlesv2.Svc
	networkLimitsService          *service.NetworkLimits
	marketDataService             *service.MarketData
	tradeService                  *service.Trade
	AssetService                  AssetService
	accountService                *service.Account
	rewardService                 *service.Reward
	marketsService                *service.Markets
	delegationService             *service.Delegation
	epochService                  *service.Epoch
	depositService                *service.Deposit
	withdrawalService             *service.Withdrawal
	governanceService             *service.Governance
	riskFactorService             *service.RiskFactor
	riskService                   *service.Risk
	networkParameterService       *service.NetworkParameter
	blockService                  BlockService
	partyService                  *service.Party
	checkpointService             *service.Checkpoint
	oracleSpecService             *service.OracleSpec
	oracleDataService             *service.OracleData
	liquidityProvisionService     *service.LiquidityProvision
	positionService               *service.Position
	transferService               *service.Transfer
	stakeLinkingService           *service.StakeLinking
	notaryService                 *service.Notary
	multiSigService               *service.MultiSig
	keyRotationService            *service.KeyRotations
	ethereumKeyRotationService    *service.EthereumKeyRotation
	nodeService                   *service.Node
	marketDepthService            *service.MarketDepth
	ledgerService                 *service.Ledger
	protocolUpgradeService        *service.ProtocolUpgrade
	networkHistoryService         NetworkHistoryService
	coreSnapshotService           *service.SnapshotData
	stopOrderService              *service.StopOrders
	fundingPeriodService          *service.FundingPeriods
	partyActivityStreak           *service.PartyActivityStreak
	referralProgramService        *service.ReferralPrograms
	referralSetsService           *service.ReferralSets
	teamsService                  *service.Teams
	vestingStatsService           *service.VestingStats
	FeesStatsService              *service.FeesStats
	fundingPaymentService         *service.FundingPayment
	volumeDiscountStatsService    *service.VolumeDiscountStats
	volumeDiscountProgramService  *service.VolumeDiscountPrograms
	paidLiquidityFeesStatsService *service.PaidLiquidityFeesStats
	partyLockedBalances           *service.PartyLockedBalances
	partyVestingBalances          *service.PartyVestingBalances
	transactionResults            *service.TransactionResults
	gamesService                  *service.Games
	marginModesService            *service.MarginModes
	ammPoolService                *service.AMMPools

	eventObserver *eventObserver

	// used in order to gracefully close streams
	ctx   context.Context
	cfunc context.CancelFunc

	trustedProxies map[string]struct{}
}

// NewGRPCServer create a new instance of the GPRC api for the vega node.
func NewGRPCServer(
	log *logging.Logger,
	config Config,
	coreServiceClient CoreServiceClient,
	eventService *subscribers.Service,
	orderService *service.Order,
	networkLimitsService *service.NetworkLimits,
	marketDataService *service.MarketData,
	tradeService *service.Trade,
	assetService *service.Asset,
	accountService *service.Account,
	rewardService *service.Reward,
	marketsService *service.Markets,
	delegationService *service.Delegation,
	epochService *service.Epoch,
	depositService *service.Deposit,
	withdrawalService *service.Withdrawal,
	governanceService *service.Governance,
	riskFactorService *service.RiskFactor,
	riskService *service.Risk,
	networkParameterService *service.NetworkParameter,
	blockService BlockService,
	checkpointService *service.Checkpoint,
	partyService *service.Party,
	candleService *candlesv2.Svc,
	oracleSpecService *service.OracleSpec,
	oracleDataService *service.OracleData,
	liquidityProvisionService *service.LiquidityProvision,
	positionService *service.Position,
	transferService *service.Transfer,
	stakeLinkingService *service.StakeLinking,
	notaryService *service.Notary,
	multiSigService *service.MultiSig,
	keyRotationService *service.KeyRotations,
	ethereumKeyRotationService *service.EthereumKeyRotation,
	nodeService *service.Node,
	marketDepthService *service.MarketDepth,
	ledgerService *service.Ledger,
	protocolUpgradeService *service.ProtocolUpgrade,
	networkHistoryService NetworkHistoryService,
	coreSnapshotService *service.SnapshotData,
	stopOrderService *service.StopOrders,
	fundingPeriodService *service.FundingPeriods,
	partyActivityStreak *service.PartyActivityStreak,
	referralProgramService *service.ReferralPrograms,
	referralSetsService *service.ReferralSets,
	teamsService *service.Teams,
	vestingStatsService *service.VestingStats,
	FeesStatsService *service.FeesStats,
	fundingPaymentService *service.FundingPayment,
	volumeDiscountStatsService *service.VolumeDiscountStats,
	volumeDiscountProgramService *service.VolumeDiscountPrograms,
	paidLiquidityFeesStatsService *service.PaidLiquidityFeesStats,
	partyLockedBalances *service.PartyLockedBalances,
	partyVestingBalances *service.PartyVestingBalances,
	transactionResults *service.TransactionResults,
	gameService *service.Games,
	marginModesService *service.MarginModes,
	ammPoolService *service.AMMPools,
) *GRPCServer {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	ctx, cfunc := context.WithCancel(context.Background())
	tps := make(map[string]struct{}, len(config.RateLimit.TrustedProxies))
	for _, ip := range config.RateLimit.TrustedProxies {
		tps[ip] = struct{}{}
	}

	return &GRPCServer{
		log:                           log,
		Config:                        config,
		vegaCoreServiceClient:         coreServiceClient,
		eventService:                  eventService,
		orderService:                  orderService,
		networkLimitsService:          networkLimitsService,
		tradeService:                  tradeService,
		AssetService:                  assetService,
		accountService:                accountService,
		rewardService:                 rewardService,
		marketsService:                marketsService,
		delegationService:             delegationService,
		epochService:                  epochService,
		depositService:                depositService,
		withdrawalService:             withdrawalService,
		multiSigService:               multiSigService,
		governanceService:             governanceService,
		riskFactorService:             riskFactorService,
		networkParameterService:       networkParameterService,
		blockService:                  blockService,
		checkpointService:             checkpointService,
		partyService:                  partyService,
		candleService:                 candleService,
		oracleSpecService:             oracleSpecService,
		oracleDataService:             oracleDataService,
		liquidityProvisionService:     liquidityProvisionService,
		positionService:               positionService,
		transferService:               transferService,
		stakeLinkingService:           stakeLinkingService,
		notaryService:                 notaryService,
		keyRotationService:            keyRotationService,
		ethereumKeyRotationService:    ethereumKeyRotationService,
		nodeService:                   nodeService,
		marketDepthService:            marketDepthService,
		riskService:                   riskService,
		marketDataService:             marketDataService,
		ledgerService:                 ledgerService,
		protocolUpgradeService:        protocolUpgradeService,
		networkHistoryService:         networkHistoryService,
		coreSnapshotService:           coreSnapshotService,
		stopOrderService:              stopOrderService,
		fundingPeriodService:          fundingPeriodService,
		partyActivityStreak:           partyActivityStreak,
		referralProgramService:        referralProgramService,
		referralSetsService:           referralSetsService,
		teamsService:                  teamsService,
		vestingStatsService:           vestingStatsService,
		FeesStatsService:              FeesStatsService,
		fundingPaymentService:         fundingPaymentService,
		volumeDiscountStatsService:    volumeDiscountStatsService,
		volumeDiscountProgramService:  volumeDiscountProgramService,
		paidLiquidityFeesStatsService: paidLiquidityFeesStatsService,
		partyLockedBalances:           partyLockedBalances,
		partyVestingBalances:          partyVestingBalances,
		transactionResults:            transactionResults,
		gamesService:                  gameService,
		marginModesService:            marginModesService,
		ammPoolService:                ammPoolService,

		eventObserver: &eventObserver{
			log:          log,
			eventService: eventService,
			Config:       config,
		},
		ctx:            ctx,
		cfunc:          cfunc,
		trustedProxies: tps,
	}
}

// ReloadConf update the internal configuration of the GRPC server.
func (g *GRPCServer) ReloadConf(cfg Config) {
	g.log.Info("reloading configuration")
	if g.log.GetLevel() != cfg.Level.Get() {
		g.log.Info("updating log level",
			logging.String("old", g.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		g.log.SetLevel(cfg.Level.Get())
	}
	tps := make(map[string]struct{}, len(cfg.RateLimit.TrustedProxies))
	for _, ip := range cfg.RateLimit.TrustedProxies {
		tps[ip] = struct{}{}
	}

	// TODO(): not updating the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	g.Config = cfg
	g.trustedProxies = tps
}

func (g *GRPCServer) ipFromContext(ctx context.Context, method string, log *logging.Logger) (string, error) {
	// first check if the request is forwarded from our restproxy
	tps := g.trustedProxies
	if len(tps) > 0 {
		// get the metadata
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			// if trusted proxies are specified, the XFF header will be used to rate-limit the IP
			// for which the request is forwarded. If no proxies are specified, or no trusted proxies
			// are found, the peer is rate limited.
			if forwardedFor, ok := md["x-forwarded-for"]; ok && len(forwardedFor) >= 2 {
				// check the proxies for trusted
				for _, pip := range forwardedFor[1:] {
					// trusted proxy found, return
					if _, ok := tps[pip]; ok {
						log.Debug("grpc request x-forwarded-for",
							logging.String("method", method),
							logging.String("remote-ip-addr", forwardedFor[0]),
						)
						return forwardedFor[0], nil
					}
				}
			}
		}
	}

	// if the request is not forwarded let's get it from the peer infos
	p, ok := peer.FromContext(ctx)
	if ok && p != nil {
		log.Debug("grpc peer client request",
			logging.String("method", method),
			logging.String("remote-ip-addr", p.Addr.String()))
		return p.Addr.String(), nil
	}

	return "", nil
}

func (g *GRPCServer) remoteAddrInterceptor(log *logging.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		ip, err := g.ipFromContext(ctx, info.FullMethod, log)
		if err != nil {
			return nil, status.Error(codes.PermissionDenied, err.Error())
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

func headersInterceptor(
	getLastBlock func(context.Context) (entities.Block, error),
	log *logging.Logger,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		var (
			height    int64
			timestamp int64
		)

		block, bErr := getLastBlock(ctx)
		if bErr != nil {
			log.Debug("failed to get last block", logging.Error(bErr))
		} else {
			height = block.Height
			timestamp = block.VegaTime.UnixNano()
		}

		for _, h := range []metadata.MD{
			// Deprecated: use 'X-Block-Height' and 'X-Block-Timestamp' instead to determine if data is fresh.
			metadata.Pairs("X-Block-Height", strconv.FormatInt(height, 10)),
			metadata.Pairs("X-Block-Timestamp", strconv.FormatInt(timestamp, 10)),
		} {
			if errH := grpc.SetHeader(ctx, h); errH != nil {
				log.Error("failed to set header", logging.Error(errH))
			}
		}

		return handler(ctx, req)
	}
}

func (g *GRPCServer) getTCPListener() (net.Listener, error) {
	ip := g.IP
	port := strconv.Itoa(g.Port)

	g.log.Info("Starting gRPC based API", logging.String("addr", ip), logging.String("port", port))

	tpcLis, err := net.Listen("tcp", net.JoinHostPort(ip, port))
	if err != nil {
		return nil, err
	}

	return tpcLis, nil
}

// Start starts the grpc server.
// Uses default TCP listener if no provided.
func (g *GRPCServer) Start(ctx context.Context, lis net.Listener) error {
	if lis == nil {
		tpcLis, err := g.getTCPListener()
		if err != nil {
			return err
		}

		lis = tpcLis
	}

	subscriptionRateLimiter := gateway.NewSubscriptionRateLimiter(g.log, g.Config.MaxSubscriptionPerClient)

	rateLimit := ratelimit.NewFromConfig(&g.RateLimit, g.log)
	intercept := grpc.ChainUnaryInterceptor(
		g.remoteAddrInterceptor(g.log),
		headersInterceptor(g.blockService.GetLastBlock, g.log),
		rateLimit.GRPCInterceptor,
	)

	streamIntercept := grpc.StreamInterceptor(subscriptionRateLimiter.WithGrpcInterceptor(g.ipFromContext))

	g.srv = grpc.NewServer(intercept, streamIntercept)

	coreProxySvc := &coreProxyService{
		conf:              g.Config,
		coreServiceClient: g.vegaCoreServiceClient,
		eventObserver:     g.eventObserver,
	}
	g.coreProxySvc = coreProxySvc
	vegaprotoapi.RegisterCoreServiceServer(g.srv, coreProxySvc)

	tradingDataSvcV2 := &TradingDataServiceV2{
		config:               g.Config,
		log:                  g.log,
		orderService:         g.orderService,
		networkLimitsService: g.networkLimitsService,
		MarketDataService:    g.marketDataService,
		tradeService:         g.tradeService,
		multiSigService:      g.multiSigService,
		notaryService:        g.notaryService,
		AssetService:         g.AssetService,
		candleService:        g.candleService,
		MarketsService:       g.marketsService,

		partyService:                  g.partyService,
		riskService:                   g.riskService,
		positionService:               g.positionService,
		accountService:                g.accountService,
		rewardService:                 g.rewardService,
		depositService:                g.depositService,
		withdrawalService:             g.withdrawalService,
		oracleSpecService:             g.oracleSpecService,
		oracleDataService:             g.oracleDataService,
		liquidityProvisionService:     g.liquidityProvisionService,
		governanceService:             g.governanceService,
		transfersService:              g.transferService,
		delegationService:             g.delegationService,
		marketDepthService:            g.marketDepthService,
		nodeService:                   g.nodeService,
		epochService:                  g.epochService,
		RiskFactorService:             g.riskFactorService,
		networkParameterService:       g.networkParameterService,
		checkpointService:             g.checkpointService,
		stakeLinkingService:           g.stakeLinkingService,
		eventService:                  g.eventService,
		ledgerService:                 g.ledgerService,
		keyRotationService:            g.keyRotationService,
		ethereumKeyRotationService:    g.ethereumKeyRotationService,
		blockService:                  g.blockService,
		protocolUpgradeService:        g.protocolUpgradeService,
		NetworkHistoryService:         g.networkHistoryService,
		coreSnapshotService:           g.coreSnapshotService,
		stopOrderService:              g.stopOrderService,
		fundingPeriodService:          g.fundingPeriodService,
		partyActivityStreak:           g.partyActivityStreak,
		referralProgramService:        g.referralProgramService,
		referralSetsService:           g.referralSetsService,
		teamsService:                  g.teamsService,
		feesStatsService:              g.FeesStatsService,
		fundingPaymentService:         g.fundingPaymentService,
		volumeDiscountStatsService:    g.volumeDiscountStatsService,
		volumeDiscountProgramService:  g.volumeDiscountProgramService,
		paidLiquidityFeesStatsService: g.paidLiquidityFeesStatsService,
		partyLockedBalances:           g.partyLockedBalances,
		partyVestingBalances:          g.partyVestingBalances,
		vestingStats:                  g.vestingStatsService,
		transactionResults:            g.transactionResults,
		gamesService:                  g.gamesService,
		marginModesService:            g.marginModesService,
		ammPoolService:                g.ammPoolService,
	}

	protoapi.RegisterTradingDataServiceServer(g.srv, tradingDataSvcV2)

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
