package api_test

import (
	"context"
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
)

type GRPCServer interface {
	ReloadConf(cfg api.Config)
	Start()
	Stop()

	SubmitOrder(ctx context.Context, req *protoapi.SubmitOrderRequest) (*types.PendingOrder, error)
}

func TestSubmitOrder(t *testing.T) {
	var g GRPCServer

	logger := logging.NewTestLogger()
	grpcConf := api.NewDefaultConfig()
	grpcConf.IP = "127.0.0.1"
	grpcConf.Port = 64312

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

	g = api.NewGRPCServer(
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

	time.Sleep(time.Second)

	req := &protoapi.SubmitOrderRequest{
		Submission: &types.OrderSubmission{
			MarketID: "nonexistantmarket",
		},
		Token: "",
	}
	pendingOrder, err := g.SubmitOrder(context.Background(), req)
	assert.Equal(t, api.ErrInvalidMarketID, err)
	assert.Nil(t, pendingOrder)

	g.Stop()
}
