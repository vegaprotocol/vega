package api_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/api"
	"code.vegaprotocol.io/vega/api/mocks"
	"code.vegaprotocol.io/vega/candles"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/governance"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/parties"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/storage"
	"code.vegaprotocol.io/vega/trades"
	"code.vegaprotocol.io/vega/transfers"
	"code.vegaprotocol.io/vega/vegatime"

	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	tmp2p "github.com/tendermint/tendermint/p2p"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc"
)

type GRPCServer interface {
	Start()
	Stop()
}

func waitForNode(t *testing.T, ctx context.Context, conn *grpc.ClientConn) {
	const maxSleep = 2000 // milliseconds

	req := &protoapi.SubmitOrderRequest{
		Submission: &types.OrderSubmission{
			MarketID: "nonexistantmarket",
		},
		Token: "",
	}

	c := protoapi.NewTradingClient(conn)
	sleepTime := 10 // milliseconds
	for sleepTime < maxSleep {
		_, err := c.SubmitOrder(ctx, req)
		if err == nil {
			t.Fatalf("Expected some sort of error, but got none.")
		}
		fmt.Println(err.Error())
		if strings.Contains(err.Error(), "Internal error") {
			return
		}
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
	g GRPCServer, tidy func(),
	conn *grpc.ClientConn, err error,
) {
	tidy = func() {}
	path := fmt.Sprintf("vegatest-%d-", port)
	tempDir, tidyTempDir, err := storage.TempDir(path)
	if err != nil {
		err = fmt.Errorf("Failed to create tmp dir: %s", err.Error())
		return
	}

	conf := config.NewDefaultConfig(tempDir)
	conf.API.IP = "127.0.0.1"
	conf.API.Port = port

	logger := logging.NewTestLogger()

	// Mock BlockchainClient
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	blockchainClient := mocks.NewMockBlockchainClient(mockCtrl)
	blockchainClient.EXPECT().Health().AnyTimes().Return(&tmctypes.ResultHealth{}, nil)
	blockchainClient.EXPECT().GetStatus(gomock.Any()).AnyTimes().Return(&tmctypes.ResultStatus{
		NodeInfo:      tmp2p.DefaultNodeInfo{Version: "0.32.9"},
		SyncInfo:      tmctypes.SyncInfo{},
		ValidatorInfo: tmctypes.ValidatorInfo{},
	}, nil)
	blockchainClient.EXPECT().GetUnconfirmedTxCount(gomock.Any()).AnyTimes().Return(0, nil)

	_, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
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
	accountService := accounts.NewService(logger, conf.Accounts, accountStore, blockchainClient)

	// Candle Service
	candleService, err := candles.NewService(logger, conf.Candles, candleStore)
	if err != nil {
		err = errors.Wrap(err, "failed to create candle service")
		return
	}

	marketDataStore := storage.NewMarketData(logger, conf.Storage)

	// Market Service
	marketService, err := markets.NewService(logger, conf.Markets, marketStore, orderStore, marketDataStore)
	if err != nil {
		err = errors.Wrap(err, "failed to create market service")
		return
	}

	// Time Service (required for Order Service)
	timeService := vegatime.New(conf.Time)

	// Order Service
	orderService, err := orders.NewService(logger, conf.Orders, orderStore, timeService, blockchainClient)
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
	tradeService, err := trades.NewService(logger, conf.Trades, tradeStore, riskStore, nil)
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

	riskService := risk.NewService(logger, conf.Risk, riskStore)

	governanceService := governance.NewService(logger, conf.Governance)

	g = api.NewGRPCServer(
		logger,
		conf.API,
		stats.New(logger, "ver", "hash"),
		blockchainClient,
		timeService,
		marketService,
		partyService,
		orderService,
		tradeService,
		candleService,
		accountService,
		transferResponseService,
		riskService,
		governanceService,
		monitoring.New(logger, monitoring.NewDefaultConfig(), blockchainClient),
	)
	if g == nil {
		err = fmt.Errorf("Failed to create gRPC server")
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

		conn, err = grpc.DialContext(ctx, fmt.Sprintf("%s:%d", conf.API.IP, conf.API.Port), grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			t.Fatalf("Failed to create connection to gRPC server")
		}

		waitForNode(t, ctx, conn)
	}

	return
}

func TestSubmitOrder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancel()

	g, tidy, conn, err := getTestGRPCServer(t, ctx, 64201, true)
	if err != nil {
		t.Fatalf("Failed to get test gRPC server: %s", err.Error())
	}
	defer tidy()

	req := &protoapi.SubmitOrderRequest{
		Submission: &types.OrderSubmission{
			MarketID: "nonexistantmarket",
		},
		Token: "",
	}
	c := protoapi.NewTradingClient(conn)
	pendingOrder, err := c.SubmitOrder(ctx, req)
	assert.Contains(t, err.Error(), "Internal error")
	assert.Nil(t, pendingOrder)

	g.Stop()
}

func TestPrepareProposal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancel()

	g, tidy, conn, err := getTestGRPCServer(t, ctx, 64201, true)
	if err != nil {
		t.Fatalf("Failed to get test gRPC server: %s", err.Error())
	}
	defer tidy()

	client := protoapi.NewTradingClient(conn)
	assert.NotNil(t, client)

	proposal, err := client.PrepareProposal(ctx, &protoapi.PrepareProposalRequest{
		PartyID: "invalid-party",
		Proposal: &types.ProposalTerms{
			Change: &types.ProposalTerms_UpdateNetwork{
				UpdateNetwork: &types.UpdateNetwork{
					Changes: &types.NetworkConfiguration{},
				},
			},
		},
	})
	assert.Contains(t, err.Error(), "Internal error")
	assert.Nil(t, proposal)

	g.Stop()
}
