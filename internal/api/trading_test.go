package api_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal"
	"code.vegaprotocol.io/vega/internal/api"
	"code.vegaprotocol.io/vega/internal/api/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/monitoring"
	types "code.vegaprotocol.io/vega/proto"
	protoapi "code.vegaprotocol.io/vega/proto/api"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	tmp2p "github.com/tendermint/tendermint/p2p"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc"
)

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
		if strings.Contains(err.Error(), "invalid market ID") {
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

func TestSubmitOrder(t *testing.T) {
	const (
		host = "127.0.0.1"
		port = 64312
	)
	logger := logging.NewTestLogger()
	grpcConf := api.NewDefaultConfig()
	grpcConf.IP = host
	grpcConf.Port = port

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	blockchainClient := mocks.NewMockBlockchainClient(mockCtrl)
	blockchainClient.EXPECT().Health().MinTimes(1).Return(&tmctypes.ResultHealth{}, nil)
	blockchainClient.EXPECT().GetStatus(gomock.Any()).MinTimes(1).Return(&tmctypes.ResultStatus{
		NodeInfo:      tmp2p.DefaultNodeInfo{Version: "0.31.5"},
		SyncInfo:      tmctypes.SyncInfo{},
		ValidatorInfo: tmctypes.ValidatorInfo{},
	}, nil)
	blockchainClient.EXPECT().GetUnconfirmedTxCount(gomock.Any()).MinTimes(1).Return(0, nil)

	marketService := mocks.NewMockMarketService(mockCtrl)
	marketService.EXPECT().GetByID(gomock.Any(), "nonexistantmarket").MinTimes(1).Return(nil, api.ErrInvalidMarketID)

	g := api.NewGRPCServer(
		logger,
		grpcConf,
		internal.NewStats(logger, "ver", "hash"),
		blockchainClient,
		nil, // time
		marketService,
		nil, // party
		nil, // orders
		nil, // trades
		nil, // candles
		nil, // accounts
		monitoring.New(logger, monitoring.NewDefaultConfig(), blockchainClient),
	)
	if g == nil {
		t.Fatalf("Failed to create gRPC server")
	}
	grpcConf.Level.Level = logging.DebugLevel
	g.ReloadConf(grpcConf)
	grpcConf.Level.Level = logging.InfoLevel
	g.ReloadConf(grpcConf)

	go g.Start()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("%s:%d", host, port), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		t.Fatalf("Failed to create connection to gRPC server")
	}

	waitForNode(t, ctx, conn)

	req := &protoapi.SubmitOrderRequest{
		Submission: &types.OrderSubmission{
			MarketID: "nonexistantmarket",
		},
		Token: "",
	}
	c := protoapi.NewTradingClient(conn)
	pendingOrder, err := c.SubmitOrder(ctx, req)
	assert.Contains(t, err.Error(), "invalid market ID")
	assert.Nil(t, pendingOrder)

	g.Stop()
}
