package api_test

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/candlesv2"

	vgtesting "code.vegaprotocol.io/data-node/libs/testing"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/data-node/accounts"
	"code.vegaprotocol.io/data-node/api"
	apimocks "code.vegaprotocol.io/data-node/api/mocks"
	"code.vegaprotocol.io/data-node/assets"
	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/candles"
	"code.vegaprotocol.io/data-node/checkpoint"
	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/delegations"
	"code.vegaprotocol.io/data-node/epochs"
	"code.vegaprotocol.io/data-node/fee"
	"code.vegaprotocol.io/data-node/governance"
	"code.vegaprotocol.io/data-node/liquidity"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/markets"
	"code.vegaprotocol.io/data-node/netparams"
	"code.vegaprotocol.io/data-node/nodes"
	"code.vegaprotocol.io/data-node/notary"
	"code.vegaprotocol.io/data-node/oracles"
	"code.vegaprotocol.io/data-node/orders"
	"code.vegaprotocol.io/data-node/parties"
	"code.vegaprotocol.io/data-node/plugins"
	"code.vegaprotocol.io/data-node/risk"
	"code.vegaprotocol.io/data-node/staking"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"
	types "code.vegaprotocol.io/protos/vega"
	vegaprotoapi "code.vegaprotocol.io/protos/vega/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

var logger = logging.NewTestLogger()

const (
	connBufSize   = 1024 * 1024
	defaultTimout = 30 * time.Second
)

type TestServer struct {
	ctrl       *gomock.Controller
	clientConn *grpc.ClientConn
	broker     *broker.Broker
	trStorage  *storage.TransferResponse
	dl         *delegations.Service
	rw         *subscribers.RewardCounters
}

// NewTestServer instantiates a new api.GRPCServer and returns a conn to it and the broker this server subscribes to.
// Any error will fail and terminate the test.
func NewTestServer(t testing.TB, ctx context.Context, blocking bool) *TestServer {
	t.Helper()

	var (
		eventBroker *broker.Broker
		conn        *grpc.ClientConn
	)
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()

	st, err := storage.InitialiseStorage(vegaPaths)
	require.NoError(t, err)

	conf := config.NewDefaultConfig()

	mockCtrl := gomock.NewController(t)

	mockCoreServiceClient := apimocks.NewMockCoreServiceClient(mockCtrl)
	mockCoreServiceClient.EXPECT().
		SubmitTransaction(gomock.Any(), gomock.Any()).
		Return(&vegaprotoapi.SubmitTransactionResponse{}, nil).AnyTimes()

	ctx, cancel := context.WithCancel(ctx)

	accountStore, err := storage.NewAccounts(logger, st.AccountsHome, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create account store: %v", err)
	}
	accountService := accounts.NewService(logger, conf.Accounts, accountStore)
	accountSub := subscribers.NewAccountSub(ctx, accountStore, logger, true)

	candleStore, err := storage.NewCandles(logger, st.CandlesHome, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create candle store: %v", err)
	}

	candleService := candles.NewService(logger, conf.Candles, candleStore)

	orderStore, err := storage.NewOrders(logger, st.OrdersHome, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create order store: %v", err)
	}

	timeService := vegatime.New(conf.Time)

	orderService := orders.NewService(logger, conf.Orders, orderStore, timeService)
	orderSub := subscribers.NewOrderEvent(ctx, conf.Subscribers, logger, orderStore, true)

	marketStore, err := storage.NewMarkets(logger, st.MarketsHome, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create market store: %v", err)
	}
	marketDataStore := storage.NewMarketData(logger, conf.Storage)

	delegationStore := storage.NewDelegations(logger, conf.Storage)

	marketDepth := subscribers.NewMarketDepthBuilder(ctx, logger, nil, false, true)

	marketService := markets.NewService(logger, conf.Markets, marketStore, orderStore, marketDataStore, marketDepth)
	newMarketSub := subscribers.NewMarketSub(ctx, marketStore, logger, true)

	partyStore, err := storage.NewParties(conf.Storage)
	if err != nil {
		t.Fatalf("failed to create party store: %v", err)
	}

	partyService, err := parties.NewService(logger, conf.Parties, partyStore)
	if err != nil {
		t.Fatalf("failed to create party service: %v", err)
	}
	partySub := subscribers.NewPartySub(ctx, partyStore, logger, true)

	riskStore := storage.NewRisks(logger, conf.Storage)
	riskService := risk.NewService(logger, conf.Risk, riskStore, marketStore, marketDataStore)

	transferResponseStore, err := storage.NewTransferResponses(logger, conf.Storage)
	if err != nil {
		t.Fatalf("failed to create risk store: %v", err)
	}
	transferResponseService := transfers.NewService(logger, conf.Transfers, transferResponseStore, nil)
	if err != nil {
		t.Fatalf("failed to create trade service: %v", err)
	}
	transferSub := subscribers.NewTransferResponse(ctx, transferResponseStore, logger, true)

	tradeStore, err := storage.NewTrades(logger, st.TradesHome, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create trade store: %v", err)
	}

	nodeStore := storage.NewNode(logger, conf.Storage)
	nodesSub := subscribers.NewNodesSub(ctx, nodeStore, logger, true)

	epochStore := storage.NewEpoch(logger, nodeStore, conf.Storage)
	delegationBalanceSub := subscribers.NewDelegationBalanceSub(ctx, nodeStore, epochStore, delegationStore, logger, true)

	tradeService := trades.NewService(logger, conf.Trades, tradeStore, nil)
	tradeSub := subscribers.NewTradeSub(ctx, tradeStore, logger, true)

	liquidityService := liquidity.NewService(ctx, logger, conf.Liquidity)

	gov, vote := govStub{}, voteStub{}

	governanceService := governance.NewService(logger, conf.Governance, eventBroker, gov, vote)

	nplugin := plugins.NewNotary(context.Background())
	notaryService := notary.NewService(logger, conf.Notary, nplugin)

	aplugin := plugins.NewAsset(context.Background())
	assetService := assets.NewService(logger, conf.Assets, aplugin)
	feeService := fee.NewService(logger, conf.Fee, marketStore, marketDataStore)
	eventService := subscribers.NewService(eventBroker)

	withdrawal := plugins.NewWithdrawal(ctx)
	deposit := plugins.NewDeposit(ctx)
	netparams := netparams.NewService(ctx)
	oracleService := oracles.NewService(ctx)
	delegationService := delegations.NewService(logger, conf.Delegations, delegationStore)

	nodeService := nodes.NewService(logger, conf.Nodes, nodeStore, epochStore)
	epochService := epochs.NewService(logger, conf.Epochs, epochStore)
	rewardsService := subscribers.NewRewards(ctx, logger, true)

	stakingService := staking.NewService(ctx, logger)

	checkpointStore, err := storage.NewCheckpoints(logger, st.CheckpointsHome, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create checkpoint store: %v", err)
	}
	checkpointSub := subscribers.NewCheckpointSub(ctx, logger, checkpointStore, true)
	checkpointSvc := checkpoint.NewService(logger, conf.Checkpoint, checkpointStore)

	chainInfoStore, err := storage.NewChainInfo(logger, st.ChainInfoHome, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create chain info store: %v", err)
	}

	sqlStore := sqlstore.SQLStore{}
	sqlBalanceStore := sqlstore.NewBalances(&sqlStore)
	sqlMarketDataStore := sqlstore.NewMarketData(&sqlStore)

	sqlOrderStore := sqlstore.NewOrders(&sqlStore)
	conf.CandlesV2.CandleStore.DefaultCandleIntervals = ""
	sqlCandleStore, err := sqlstore.NewCandles(ctx, &sqlStore, conf.CandlesV2.CandleStore)
	if err != nil {
		t.Fatalf("failed to create candle store: %v", err)
	}
	candlesServiceV2 := candlesv2.NewService(ctx, logger, conf.CandlesV2, sqlCandleStore)

	sqlTradeStore := sqlstore.NewTrades(&sqlStore)
	sqlPositionStore := sqlstore.NewPositions(&sqlStore)
	sqlNetworkLimitsStore := sqlstore.NewNetworkLimits(&sqlStore)
	sqlAssetStore := sqlstore.NewAssets(&sqlStore)
	sqlAccountStore := sqlstore.NewAccounts(&sqlStore)
	sqlRewardsStore := sqlstore.NewRewards(&sqlStore)
	sqlMarketsStore := sqlstore.NewMarkets(&sqlStore)
	sqlDelegationStore := sqlstore.NewDelegations(&sqlStore)
	sqlEpochStore := sqlstore.NewEpochs(&sqlStore)
	sqlDepositStore := sqlstore.NewDeposits(&sqlStore)
	sqlWithdrawalsStore := sqlstore.NewWithdrawals(&sqlStore)
	sqlProposalStore := sqlstore.NewProposals(&sqlStore)
	sqlVoteStore := sqlstore.NewVotes(&sqlStore)
	sqlRiskFactorsStore := sqlstore.NewRiskFactors(&sqlStore)
	sqlMarginLevelsStore := sqlstore.NewMarginLevels(&sqlStore)
	sqlNetParamStore := sqlstore.NewNetworkParameters(&sqlStore)
	sqlBlockStore := sqlstore.NewBlocks(&sqlStore)
	sqlCheckpointStore := sqlstore.NewCheckpoints(&sqlStore)
	sqlPartyStore := sqlstore.NewParties(&sqlStore)
	sqlOracleSpecStore := sqlstore.NewOracleSpec(&sqlStore)
	sqlOracleDataStore := sqlstore.NewOracleData(&sqlStore)
	sqlLPDataStore := sqlstore.NewLiquidityProvision(&sqlStore)
	sqlTransfersStore := sqlstore.NewTransfers(&sqlStore)
	sqlStakeLinkingStore := sqlstore.NewStakeLinking(&sqlStore)
	sqlNotaryStore := sqlstore.NewNotary(&sqlStore)

	eventSource, err := broker.NewEventSource(conf.Broker, logger)
	if err != nil {
		t.Fatalf("failed to create event source: %v", err)
	}

	eventBroker, err = broker.New(ctx, logger, conf.Broker, chainInfoStore, eventSource)

	if err != nil {
		t.Fatalf("failed to create broker: %v", err)
	}
	eventBroker.SubscribeBatch(
		accountSub,
		transferSub,
		orderSub,
		tradeSub,
		partySub,
		newMarketSub,
		oracleService,
		liquidityService,
		deposit,
		withdrawal,
		checkpointSub,
		delegationBalanceSub,
		rewardsService,
		nodesSub,
	)

	srv := api.NewGRPCServer(
		logger,
		conf.API,
		false,
		mockCoreServiceClient,
		timeService,
		marketService,
		partyService,
		orderService,
		liquidityService,
		tradeService,
		candleService,
		accountService,
		transferResponseService,
		riskService,
		governanceService,
		notaryService,
		assetService,
		feeService,
		eventService,
		oracleService,
		withdrawal,
		deposit,
		marketDepth,
		netparams,
		nodeService,
		epochService,
		delegationService,
		rewardsService,
		stakingService,
		checkpointSvc,
		sqlBalanceStore,
		sqlOrderStore,
		sqlNetworkLimitsStore,
		sqlMarketDataStore,
		sqlTradeStore,
		sqlAssetStore,
		sqlAccountStore,
		sqlRewardsStore,
		sqlMarketsStore,
		sqlDelegationStore,
		sqlEpochStore,
		sqlDepositStore,
		sqlWithdrawalsStore,
		sqlProposalStore,
		sqlVoteStore,
		sqlRiskFactorsStore,
		sqlMarginLevelsStore,
		sqlNetParamStore,
		sqlBlockStore,
		sqlCheckpointStore,
		sqlPartyStore,
		candlesServiceV2,
		sqlOracleSpecStore,
		sqlOracleDataStore,
		sqlLPDataStore,
		sqlPositionStore,
		sqlTransfersStore,
		sqlStakeLinkingStore,
		sqlNotaryStore,
	)
	if srv == nil {
		t.Fatal("failed to create gRPC server")
	}

	lis := bufconn.Listen(connBufSize)

	// Start the gRPC server, then wait for it to be ready.
	go srv.Start(ctx, lis)

	t.Cleanup(func() {
		// Close stores after test so files are properly closed
		accountStore.Close()
		candleStore.Close()
		orderStore.Close()
		marketStore.Close()
		checkpointStore.Close()
		riskStore.Close()
		tradeStore.Close()
		transferResponseStore.Close()

		cancel()
		st.Purge()
		cleanupFn()
	})

	if blocking {
		ctxDialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }
		conn, err = grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(ctxDialer), grpc.WithInsecure())
		if err != nil {
			t.Fatalf("failed to dial gRPC server: %v", err)
		}
	}

	return &TestServer{
		ctrl:       mockCtrl,
		broker:     eventBroker,
		clientConn: conn,
		trStorage:  transferResponseStore,
		dl:         delegationService,
		rw:         rewardsService,
	}
}

// PublishEvents reads JSON encoded BusEvents from golden file testdata/<type>-events.golden and publishes the
// corresponding core Event to the broker. It uses the given converter func to perform the conversion.
func PublishEvents(
	t *testing.T,
	ctx context.Context,
	b *broker.Broker,
	convertEvt func(be *eventspb.BusEvent) (events.Event, error),
	goldenFile string,
) {
	t.Helper()
	path := filepath.Join("testdata", goldenFile)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open golden file %s: %v", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	evts := map[events.Type][]events.Event{}
	for scanner.Scan() {
		jsonBytes := scanner.Bytes()
		var be eventspb.BusEvent
		unmarshaller := &jsonpb.Unmarshaler{AllowUnknownFields: true}
		err = unmarshaller.Unmarshal(bytes.NewReader(jsonBytes), &be)
		if err != nil {
			t.Fatal(err)
		}
		e, err := convertEvt(&be)
		if err != nil {
			t.Fatalf("failed to convert BusEvent to Event: %v", err)
		}
		s, ok := evts[e.Type()]
		if !ok {
			s = []events.Event{}
		}
		s = append(s, e)
		evts[e.Type()] = s
	}

	// we've grouped events per type, now send them all in batches
	for _, batch := range evts {
		for _, e := range batch {
			b.Send(e)
		}
	}

	// There used to be code here that would create a dummy-subscriber that also received events and could be used to wait
	// for the first event to be processed before sending the below time-event. Unfortunately, it didn't really work.
	// If the dummy-sub was the first sub to receive the first event, we would spot that it was received and move on to sending
	// the time-event. This *could* happen before the real subs had a chance to look at the first event, and so they would
	// *sometimes* end up receiving the time-event first meaning the real event was never flushed.
	// I think all we can really do here to always be sure the events are received in necessary order is to introduce a small
	// sleep. Its not ideal but it is what it is.
	time.Sleep(20 * time.Millisecond)

	// whatever time it is now + 1 second
	now := time.Now()
	// the broker reacts to Time events to trigger writes the data stores
	tue := events.NewTime(ctx, now)
	b.Send(tue)
}

type govStub struct{}

type voteStub struct{}

func (g govStub) Filter(_ bool, filters ...subscribers.ProposalFilter) []*types.GovernanceData {
	return nil
}

func (v voteStub) Filter(filters ...subscribers.VoteFilter) []*types.Vote {
	return nil
}
