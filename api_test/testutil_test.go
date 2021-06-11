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
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/api/mocks"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/governance"
	mockgov "code.vegaprotocol.io/vega/governance/mocks"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/notary"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/plugins"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/mock/gomock"
	tmp2p "github.com/tendermint/tendermint/p2p"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc"
)

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

	logger := logging.NewTestLogger()

	mockCtrl := gomock.NewController(t)
	blockchainClient := mocks.NewMockBlockchainClient(mockCtrl)
	blockchainClient.EXPECT().Health().AnyTimes().Return(&tmctypes.ResultHealth{}, nil)
	blockchainClient.EXPECT().GetStatus(gomock.Any()).AnyTimes().Return(&tmctypes.ResultStatus{
		NodeInfo:      tmp2p.DefaultNodeInfo{Version: "0.33.8"},
		SyncInfo:      tmctypes.SyncInfo{},
		ValidatorInfo: tmctypes.ValidatorInfo{},
	}, nil)
	blockchainClient.EXPECT().GetUnconfirmedTxCount(gomock.Any()).AnyTimes().Return(0, nil)

	ctx, cancel := context.WithCancel(ctx)

	accountStore, err := storage.NewAccounts(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create account store: %v", err)
		return
	}
	accountService := accounts.NewService(logger, conf.Accounts, accountStore)
	accountSub := subscribers.NewAccountSub(ctx, accountStore, true)

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

	partyStore, err := storage.NewParties(conf.Storage)
	if err != nil {
		t.Fatalf("failed to create party store: %v", err)
	}

	partyService, err := parties.NewService(logger, conf.Parties, partyStore)
	if err != nil {
		t.Fatalf("failed to create party service: %v", err)
	}

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

	tradeStore, err := storage.NewTrades(logger, conf.Storage, cancel)
	if err != nil {
		t.Fatalf("failed to create trade store: %v", err)
	}

	tradeService, err := trades.NewService(logger, conf.Trades, tradeStore, nil)
	if err != nil {
		t.Fatalf("failed to create trade service: %v", err)
	}

	liquidityService := liquidity.NewService(ctx, logger, conf.Liquidity)

	eventBroker = broker.New(ctx)
	eventBroker.SubscribeBatch(accountSub)

	gov, vote := govStub{}, voteStub{}

	governanceService := governance.NewService(logger, conf.Governance, eventBroker, gov, vote, mockgov.NewMockNetParams(mockCtrl))

	nplugin := plugins.NewNotary(context.Background())
	notaryService := notary.NewService(logger, conf.Notary, nplugin)

	aplugin := plugins.NewAsset(context.Background())
	assetService := assets.NewService(logger, conf.Assets, aplugin)
	feeService := fee.NewService(logger, conf.Execution.Fee, marketStore, marketDataStore)
	eventService := subscribers.NewService(eventBroker)

	evtfwd := mocks.NewMockEvtForwarder(mockCtrl)
	withdrawal := plugins.NewWithdrawal(ctx)
	deposit := plugins.NewDeposit(ctx)
	netparams := netparams.NewService(ctx)
	oracleService := oracles.NewService(ctx)

	srv := api.NewGRPCServer(
		logger,
		conf.API,
		stats.New(logger, conf.Stats, "ver", "hash"),
		blockchainClient,
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
		evtfwd,
		assetService,
		feeService,
		eventService,
		oracleService,
		withdrawal,
		deposit,
		marketDepth,
		netparams,
		monitoring.New(logger, monitoring.NewDefaultConfig(), blockchainClient),
	)
	if srv == nil {
		t.Fatal("failed to create gRPC server")
	}

	t.Cleanup(func() {
		srv.Stop()
		cleanTempDir()
		cancel()
	})

	if blocking {
		go srv.Start()

		target := net.JoinHostPort(conf.API.IP, strconv.Itoa(conf.API.Port))
		conn, err = grpc.DialContext(ctx, target, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			t.Fatalf("failed to dial gRPC server: %v", err)
		}

		if err = waitForNode(t, ctx, conn); err != nil {
			t.Fatalf("failed to start gRPC server: %v", err)
		}
	}

	return
}

// PublishEvents reads JSON encoded BusEvents from golden file testdata/events.golden and publishes the
// corresponding core Event to the broker. It uses the given converter func to perform the conversion.
func PublishEvents(t *testing.T, ctx context.Context, b *broker.Broker, convertEvt func(be *eventspb.BusEvent) (events.Event, error)) {
	t.Helper()
	path := filepath.Join("testdata", "events.golden")
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

func waitForNode(t testing.TB, ctx context.Context, conn *grpc.ClientConn) error {
	const maxSleep = 2000

	req := &protoapi.PrepareSubmitOrderRequest{
		Submission: &commandspb.OrderSubmission{
			Type:     types.Order_TYPE_LIMIT,
			MarketId: "non-existent",
		},
	}

	c := protoapi.NewTradingServiceClient(conn)
	sleepTime := 10
	for sleepTime < maxSleep {
		_, err := c.PrepareSubmitOrder(ctx, req)
		if err == nil {
			return fmt.Errorf("expected error when calling PrepareSubmitOrderRequest API with invalid marketID")
		}

		if strings.Contains(err.Error(), "InvalidArgument") {
			return nil
		}

		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		sleepTime *= 2
	}
	if sleepTime >= maxSleep {
		return fmt.Errorf("timeout waiting for gRPC server to respond")
	}

	return nil
}

type govStub struct{}

type voteStub struct{}

func (g govStub) Filter(_ bool, filters ...subscribers.ProposalFilter) []*types.GovernanceData {
	return nil
}

func (v voteStub) Filter(filters ...subscribers.VoteFilter) []*types.Vote {
	return nil
}
