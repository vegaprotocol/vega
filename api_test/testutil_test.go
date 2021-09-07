package api_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"

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
	"code.vegaprotocol.io/data-node/stats"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"
	types "code.vegaprotocol.io/protos/vega"
	vegaprotoapi "code.vegaprotocol.io/protos/vega/api"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/golang/mock/gomock"
	"google.golang.org/grpc"
)

var logger = logging.NewTestLogger()

const defaultTimout = 30 * time.Second

// NewTestServer instantiates a new api.GRPCServer and returns a conn to it and the broker this server subscribes to.
// Any error will fail and terminate the test.
func NewTestServer(t testing.TB, ctx context.Context, blocking bool) (conn *grpc.ClientConn, eventBroker *broker.Broker) {
	t.Helper()

	port := randomPort()
	path := fmt.Sprintf("vegatest-%d-", port)
	tmpDir, cleanTempDir, err := storage.TempDir(path)
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}

	conf := config.NewDefaultConfig(tmpDir)
	conf.API.IP = "127.0.0.1"
	conf.API.Port = port

	mockCtrl := gomock.NewController(t)

	mockTradingServiceClient := apimocks.NewMockTradingServiceClient(mockCtrl)
	mockTradingServiceClient.EXPECT().
		SubmitTransactionV2(gomock.Any(), gomock.Any()).
		Return(&vegaprotoapi.SubmitTransactionV2Response{}, nil)

	ctx, cancel := context.WithCancel(ctx)

	accountStore, err := storage.NewAccounts(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create account store: %v", err)
		return
	}
	accountService := accounts.NewService(logger, conf.Accounts, accountStore)
	accountSub := subscribers.NewAccountSub(ctx, accountStore, logger, true)

	candleStore, err := storage.NewCandles(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create candle store: %v", err)
	}

	candleService := candles.NewService(logger, conf.Candles, candleStore)

	orderStore, err := storage.NewOrders(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create order store: %v", err)
	}

	timeService := vegatime.New(conf.Time)

	orderService := orders.NewService(logger, conf.Orders, orderStore, timeService)
	orderSub := subscribers.NewOrderEvent(ctx, conf.Subscribers, logger, orderStore, true)

	marketStore, err := storage.NewMarkets(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create market store: %v", err)
	}
	marketDataStore := storage.NewMarketData(logger, conf.Storage)

	delegationStore := storage.NewDelegations(logger, conf.Storage)

	marketDepth := subscribers.NewMarketDepthBuilder(ctx, logger, true)
	if marketDepth == nil {
		return
	}

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
	transferResponseService := transfers.NewService(logger, conf.Transfers, transferResponseStore)
	if err != nil {
		t.Fatalf("failed to create trade service: %v", err)
	}
	transferSub := subscribers.NewTransferResponse(ctx, transferResponseStore, logger, true)

	tradeStore, err := storage.NewTrades(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create trade store: %v", err)
	}

	nodeStore := storage.NewNode(logger, conf.Storage)
	epochStore := storage.NewEpoch(logger, nodeStore, conf.Storage)

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

	checkpointStore, err := storage.NewCheckpoints(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create checkpoint store: %v", err)
	}
	checkpointSub := subscribers.NewCheckpointSub(ctx, logger, checkpointStore, true)
	checkpointSvc := checkpoint.NewService(logger, conf.Checkpoint, checkpointStore)

	eventBroker, err = broker.New(ctx, logger, conf.Broker)
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
	)

	srv := api.NewGRPCServer(
		logger,
		conf.API,
		stats.New(logger, conf.Stats, "ver", "hash"),
		mockTradingServiceClient,
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
	)
	if srv == nil {
		t.Fatal("failed to create gRPC server")
	}

	go srv.Start(ctx)

	t.Cleanup(func() {
		cleanTempDir()
		cancel()
	})

	if blocking {
		target := net.JoinHostPort(conf.API.IP, strconv.Itoa(conf.API.Port))
		conn, err = grpc.DialContext(ctx, target, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			t.Fatalf("failed to dial gRPC server: %v", err)
		}
	}

	return
}

// PublishEvents reads JSON encoded BusEvents from golden file testdata/<type>-events.golden and publishes the
// corresponding core Event to the broker. It uses the given converter func to perform the conversion.
func PublishEvents(
	t *testing.T,
	ctx context.Context,
	b *broker.Broker,
	convertEvt func(be *eventspb.BusEvent) (events.Event, error),
	goldenFile string) {

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

	// add time event subscriber so we can verify the time event was received at the end
	sCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	sub := NewEventSubscriber(sCtx)
	id := b.Subscribe(sub)

	// we've grouped events per type, now send them all in batches
	for _, batch := range evts {
		for _, e := range batch {
			b.Send(e)

			if err := waitForEvent(sCtx, sub, e); err != nil {
				t.Fatalf("Did not receive the expected event within reasonable time: %+v", e)
			}
		}
	}

	t.Logf("%d events sent", len(evts))

	// whatever time it is now + 1 second
	now := time.Now()
	// the broker reacts to Time events to trigger writes the data stores
	tue := events.NewTime(ctx, now)
	b.Send(tue)
	// await confirmation that we've actually received the time update event
	if err := waitForEvent(sCtx, sub, tue); err != nil {
		t.Fatalf("Did not receive the expected event within reasonable time: %+v", tue)
	}

	t.Log("time event received")

	// cancel the subscriber ctx
	cancel()

	sub.Halt()
	// unsubscribe the ad-hoc subscriber
	b.Unsubscribe(id)
}

func waitForEvent(ctx context.Context, sub *EventSubscriber, event events.Event) error {
	for {
		receivedEvent, err := sub.ReceivedEvent(ctx)
		if err != nil {
			return err
		}

		if receivedEvent == event {
			return nil
		}
	}
}

func randomPort() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(65535-1023) + 1023
}

type govStub struct{}

type voteStub struct{}

func (g govStub) Filter(_ bool, filters ...subscribers.ProposalFilter) []*types.GovernanceData {
	return nil
}

func (v voteStub) Filter(filters ...subscribers.VoteFilter) []*types.Vote {
	return nil
}
