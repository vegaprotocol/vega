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
	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/fee"
	"code.vegaprotocol.io/data-node/governance"
	"code.vegaprotocol.io/data-node/liquidity"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/markets"
	"code.vegaprotocol.io/data-node/netparams"
	"code.vegaprotocol.io/data-node/notary"
	"code.vegaprotocol.io/data-node/oracles"
	"code.vegaprotocol.io/data-node/orders"
	"code.vegaprotocol.io/data-node/parties"
	"code.vegaprotocol.io/data-node/plugins"
	vegaprotoapi "code.vegaprotocol.io/protos/data-node/api"
	types "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/data-node/risk"
	"code.vegaprotocol.io/data-node/stats"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"

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
		Return(&vegaprotoapi.SubmitTransactionV2Request{}, nil)

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

	candleService, err := candles.NewService(logger, conf.Candles, candleStore)
	if err != nil {
		t.Fatalf("failed to create candle service: %v", err)
	}

	orderStore, err := storage.NewOrders(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create order store: %v", err)
	}

	timeService := vegatime.New(conf.Time)

	orderService, err := orders.NewService(logger, conf.Orders, orderStore, timeService)
	if err != nil {
		t.Fatalf("failed to create order service: %v", err)
	}
	orderSub := subscribers.NewOrderEvent(ctx, conf.Subscribers, logger, orderStore, true)

	marketStore, err := storage.NewMarkets(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create market store: %v", err)
	}
	marketDataStore := storage.NewMarketData(logger, conf.Storage)

	marketDepth := subscribers.NewMarketDepthBuilder(ctx, logger, true)
	if marketDepth == nil {
		return
	}

	marketService, err := markets.NewService(logger, conf.Markets, marketStore, orderStore, marketDataStore, marketDepth)
	if err != nil {
		t.Fatalf("failed to create market service: %v", err)
		return
	}
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

	tradeService, err := trades.NewService(logger, conf.Trades, tradeStore, nil)
	if err != nil {
		t.Fatalf("failed to create trade service: %v", err)
	}
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

	eventBroker = broker.New(ctx)
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
		withdrawal)

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
	)
	if srv == nil {
		t.Fatal("failed to create gRPC server")
	}

	go srv.Start()

	t.Cleanup(func() {
		srv.Stop()
		cleanTempDir()
		cancel()
	})

	if blocking {
		target := net.JoinHostPort(conf.API.IP, strconv.Itoa(conf.API.Port))
		conn, err = grpc.DialContext(ctx, target, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			t.Fatalf("failed to dial gRPC server: %v", err)
		}

		// if err = waitForNode(t, ctx, conn); err != nil {
		// 	t.Fatalf("failed to start gRPC server: %v", err)
		// }
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
	// we've grouped events per type, now send them all in batches
	for _, e := range evts {
		b.SendBatch(e)
	}

	t.Logf("%d events sent", len(evts))

	// add time event subscriber so we can verify the time event was received at the end
	sCtx, cfunc := context.WithCancel(ctx)
	tmConf := NewTimeSub(sCtx)
	id := b.Subscribe(tmConf)

	// whatever time it is now + 1 second
	now := time.Now()
	// the broker reacts to Time events to trigger writes the data stores
	b.Send(events.NewTime(ctx, now))
	// await confirmation that we've actually received the time update event
	if !waitForTime(tmConf, now) {
		t.Fatal("Did not receive the expected time event within reasonable time")
	}

	t.Log("time event received")

	// halt the subscriber
	tmConf.Halt()
	// cancel the subscriber ctx
	cfunc()
	// unsubscribe the ad-hoc subscriber
	b.Unsubscribe(id)
	// we've received the time event, but that could've been received before the other events.
	// Now send out a second time event to ensure the other events get flushed/persisted
	b.Send(events.NewTime(ctx, now.Add(time.Second)))
}

func waitForTime(tmConf *TimeSub, now time.Time) bool {
	for {
		times := tmConf.GetReveivedTimes()
		if times == nil {
			// the subscriber context was cancelled, no need to wait for this anylonger
			return false
		}
		for _, tm := range times {
			if tm.Equal(now) {
				return true
			}
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
