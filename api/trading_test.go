package api_test

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/accounts"
	"code.vegaprotocol.io/data-node/api"
	"code.vegaprotocol.io/data-node/api/mocks"
	"code.vegaprotocol.io/data-node/assets"
	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/candles"
	"code.vegaprotocol.io/data-node/config"
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
	"code.vegaprotocol.io/data-node/risk"
	"code.vegaprotocol.io/data-node/stats"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/data-node/trades"
	"code.vegaprotocol.io/data-node/transfers"
	"code.vegaprotocol.io/data-node/vegatime"

	protoapi "code.vegaprotocol.io/data-node/proto/api"
	vegaprotoapi "code.vegaprotocol.io/data-node/proto/api"
	protoapiv1 "code.vegaprotocol.io/data-node/proto/api/v1"
	types "code.vegaprotocol.io/data-node/proto/vega"
	commandspb "code.vegaprotocol.io/data-node/proto/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type GRPCServer interface {
	Start()
	Stop()
}

type govStub struct{}

type voteStub struct{}

func (g govStub) Filter(_ bool, filters ...subscribers.ProposalFilter) []*types.GovernanceData {
	return nil
}

func (v voteStub) Filter(filters ...subscribers.VoteFilter) []*types.Vote {
	return nil
}

func waitForNode(t *testing.T, ctx context.Context, conn *grpc.ClientConn) {
	const maxSleep = 2000 // milliseconds

	c := protoapi.NewTradingDataServiceClient(conn)

	sleepTime := 10 // milliseconds
	for sleepTime < maxSleep {
		_, err := c.GetProposals(ctx, &protoapi.GetProposalsRequest{})
		if err == nil {
			return
		}

		fmt.Println(err)

		fmt.Printf("Sleeping for %d milliseconds\n", sleepTime)
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		sleepTime *= 2
	}
	if sleepTime >= maxSleep {
		t.Fatalf("Gave up waiting for gRPC server to respond properly.")
	}
}

func getTestGRPCServer(
	t *testing.T,
	ctx context.Context,
	port int,
	startAndWait bool,
) (
	tidy func(),
	conn *grpc.ClientConn, mockTradingServiceClient *mocks.MockTradingServiceClient,
	err error,
) {
	tidy = func() {}
	path := fmt.Sprintf("vegatest-%d-", port)
	tempDir, tidyTempDir, err := storage.TempDir(path)
	if err != nil {
		err = fmt.Errorf("failed to create tmp dir: %s", err.Error())
		return
	}

	conf := config.NewDefaultConfig(tempDir)
	conf.API.IP = "127.0.0.1"
	conf.API.Port = port

	logger := logging.NewTestLogger()

	// Mock BlockchainClient
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTradingServiceClient = mocks.NewMockTradingServiceClient(mockCtrl)

	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		cancel()
	}()

	// Account Store
	accountStore, err := storage.NewAccounts(logger, conf.Storage, cancel)
	if err != nil {
		err = errors.Wrap(err, "failed to create account store")
		return
	}

	// Candle Store
	candleStore, err := storage.NewCandles(logger, conf.Storage, cancel)
	if err != nil {
		err = errors.Wrap(err, "failed to create candle store")
		return
	}

	// Market Store
	marketStore, err := storage.NewMarkets(logger, conf.Storage, cancel)
	if err != nil {
		err = errors.Wrap(err, "failed to create market store")
		return
	}

	// Order Store
	orderStore, err := storage.NewOrders(logger, conf.Storage, cancel)
	if err != nil {
		err = errors.Wrap(err, "failed to create order store")
		return
	}

	// Party Store
	partyStore, err := storage.NewParties(conf.Storage)
	if err != nil {
		err = errors.Wrap(err, "failed to create party store")
		return
	}

	// Risk Store
	riskStore := storage.NewRisks(logger, conf.Storage)

	transferResponseStore, err := storage.NewTransferResponses(logger, conf.Storage)
	if err != nil {
		err = errors.Wrap(err, "failed to create risk store")
		return
	}

	// Trade Store
	tradeStore, err := storage.NewTrades(logger, conf.Storage, cancel)
	if err != nil {
		err = errors.Wrap(err, "failed to create trade store")
		return
	}

	// Account Service
	accountService := accounts.NewService(logger, conf.Accounts, accountStore)

	// Candle Service
	candleService, err := candles.NewService(logger, conf.Candles, candleStore)
	if err != nil {
		err = errors.Wrap(err, "failed to create candle service")
		return
	}

	marketDataStore := storage.NewMarketData(logger, conf.Storage)

	marketDepth := subscribers.NewMarketDepthBuilder(ctx, logger, true)
	if marketDepth == nil {
		return
	}

	// Market Service
	marketService, err := markets.NewService(logger, conf.Markets, marketStore, orderStore, marketDataStore, marketDepth)
	if err != nil {
		err = errors.Wrap(err, "failed to create market service")
		return
	}

	// Time Service (required for Order Service)
	timeService := vegatime.New(conf.Time)

	// Order Service
	orderService, err := orders.NewService(logger, conf.Orders, orderStore, timeService)
	if err != nil {
		err = errors.Wrap(err, "failed to create order service")
		return
	}

	// Party Service
	partyService, err := parties.NewService(logger, conf.Parties, partyStore)
	if err != nil {
		err = errors.Wrap(err, "failed to create party service")
		return
	}

	// Trade Service
	tradeService, err := trades.NewService(logger, conf.Trades, tradeStore, nil)
	if err != nil {
		err = errors.Wrap(err, "failed to create trade service")
		return
	}

	// TransferResponse Service
	transferResponseService := transfers.NewService(logger, conf.Transfers, transferResponseStore)
	if err != nil {
		err = errors.Wrap(err, "failed to create trade service")
		return
	}

	liquidityService := liquidity.NewService(ctx, logger, conf.Liquidity)

	riskService := risk.NewService(logger, conf.Risk, riskStore, marketStore, marketDataStore)
	// stub...
	gov, vote := govStub{}, voteStub{}
	broker := broker.New(ctx)

	governanceService := governance.NewService(logger, conf.Governance, broker, gov, vote)

	nplugin := plugins.NewNotary(context.Background())
	notaryService := notary.NewService(logger, conf.Notary, nplugin)

	aplugin := plugins.NewAsset(context.Background())
	assetService := assets.NewService(logger, conf.Assets, aplugin)
	feeService := fee.NewService(logger, conf.Fee, marketStore, marketDataStore)
	eventService := subscribers.NewService(broker)

	withdrawal := plugins.NewWithdrawal(ctx)
	deposit := plugins.NewDeposit(ctx)
	netparams := netparams.NewService(ctx)
	oracleService := oracles.NewService(ctx)

	g := api.NewGRPCServer(
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
	if g == nil {
		err = fmt.Errorf("failed to create gRPC server")
		return
	}

	tidy = func() {
		g.Stop()
		tidyTempDir()
		cancel()
	}

	if startAndWait {
		// Start the gRPC server, then wait for it to be ready.
		go g.Start()

		target := net.JoinHostPort(conf.API.IP, strconv.Itoa(conf.API.Port))
		conn, err = grpc.DialContext(ctx, target, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			t.Fatalf("Failed to create connection to gRPC server")
		}

		waitForNode(t, ctx, conn)
	}

	return
}

func TestSubmitTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("proxy call is successful", func(t *testing.T) {
		tidy, conn, mockTradingServiceClient, err := getTestGRPCServer(t, ctx, 64201, true)
		if err != nil {
			t.Fatalf("Failed to get test gRPC server: %s", err.Error())
		}
		defer tidy()

		req := &protoapiv1.SubmitTransactionRequest{
			Type: protoapiv1.SubmitTransactionRequest_TYPE_UNSPECIFIED,
			Tx: &commandspb.Transaction{
				InputData: []byte("input data"),
				Signature: &commandspb.Signature{
					Value:   "value",
					Algo:    "algo",
					Version: 1,
				},
			},
		}

		expectedRes := &protoapiv1.SubmitTransactionResponse{Success: true}

		vegaReq := &vegaprotoapi.SubmitTransactionV2Request{
			Type: vegaprotoapi.SubmitTransactionV2Request_TYPE_UNSPECIFIED,
			Tx: &commandspb.Transaction{
				InputData: []byte("input data"),
				Signature: &commandspb.Signature{
					Value:   "value",
					Algo:    "algo",
					Version: 1,
				},
			},
		}

		mockTradingServiceClient.EXPECT().
			SubmitTransactionV2(gomock.Any(), vegaReq).
			Return(&vegaprotoapi.SubmitTransactionV2Response{Success: true}, nil)

		proxyClient := protoapiv1.NewTradingProxyServiceClient(conn)
		assert.NotNil(t, proxyClient)

		actualResp, err := proxyClient.SubmitTransaction(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedRes, actualResp)
	})

	t.Run("proxy propagates an error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tidy, conn, mockTradingServiceClient, err := getTestGRPCServer(t, ctx, 64201, true)
		if err != nil {
			t.Fatalf("Failed to get test gRPC server: %s", err.Error())
		}
		defer tidy()

		req := &protoapiv1.SubmitTransactionRequest{
			Type: protoapiv1.SubmitTransactionRequest_TYPE_COMMIT,
			Tx: &commandspb.Transaction{
				InputData: []byte("input data"),
				Signature: &commandspb.Signature{
					Value:   "value",
					Algo:    "algo",
					Version: 1,
				},
			},
		}

		vegaReq := &vegaprotoapi.SubmitTransactionV2Request{
			Type: vegaprotoapi.SubmitTransactionV2Request_TYPE_COMMIT,
			Tx: &commandspb.Transaction{
				InputData: []byte("input data"),
				Signature: &commandspb.Signature{
					Value:   "value",
					Algo:    "algo",
					Version: 1,
				},
			},
		}

		mockTradingServiceClient.EXPECT().
			SubmitTransactionV2(gomock.Any(), vegaReq).
			Return(nil, errors.New("Critical error"))

		proxyClient := protoapiv1.NewTradingProxyServiceClient(conn)
		assert.NotNil(t, proxyClient)

		actualResp, err := proxyClient.SubmitTransaction(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, actualResp)
		assert.Contains(t, err.Error(), "Critical error")
	})
}
