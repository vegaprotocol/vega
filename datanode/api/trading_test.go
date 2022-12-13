// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/api"
	"code.vegaprotocol.io/vega/datanode/api/mocks"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/config"
	vgtesting "code.vegaprotocol.io/vega/datanode/libs/testing"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/subscribers"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vegaprotoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/test/bufconn"
)

const connBufSize = 1024 * 1024

func waitForNode(ctx context.Context, t *testing.T, conn *grpc.ClientConn) {
	t.Helper()
	const maxSleep = 2000 // milliseconds

	c := v2.NewTradingDataServiceClient(conn)

	sleepTime := 10 // milliseconds
	for sleepTime < maxSleep {
		_, err := c.Ping(ctx, &v2.PingRequest{})
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

func getTestGRPCServer(t *testing.T, ctx context.Context) (tidy func(), conn *grpc.ClientConn, mockCoreServiceClient *mocks.MockCoreServiceClient, err error) {
	t.Helper()
	_, cleanupFn := vgtesting.NewVegaPaths()

	conf := config.NewDefaultConfig()
	conf.API.IP = "127.0.0.1"
	conf.API.Port = 64201

	// Mock BlockchainClient
	mockCtrl := gomock.NewController(t)

	mockCoreServiceClient = mocks.NewMockCoreServiceClient(mockCtrl)

	mockCoreServiceClient.EXPECT().GetState().Return(connectivity.Ready).AnyTimes()

	mockDeHistoryService := mocks.NewMockDeHistoryService(mockCtrl)

	eventSource, err := broker.NewEventSource(conf.Broker, logging.NewTestLogger(), "")
	if err != nil {
		t.Fatalf("failed to create event source: %v", err)
	}

	conf.CandlesV2.CandleStore.DefaultCandleIntervals = ""

	sqlConn := &sqlstore.ConnectionSource{
		Connection: dummyConnection{},
	}

	bro, err := broker.New(ctx, logging.NewTestLogger(), conf.Broker, "", eventSource)
	if err != nil {
		err = errors.Wrap(err, "failed to create broker")
		return
	}

	logger := logging.NewTestLogger()
	eventService := subscribers.NewService(bro)
	sqlOrderStore := sqlstore.NewOrders(sqlConn, logger)
	sqlOrderService := service.NewOrder(sqlOrderStore, logger)
	sqlNetworkLimitsService := service.NewNetworkLimits(sqlstore.NewNetworkLimits(sqlConn), logger)
	sqlMarketDataService := service.NewMarketData(sqlstore.NewMarketData(sqlConn), logger)
	sqlCandleStore := sqlstore.NewCandles(ctx, sqlConn, conf.CandlesV2.CandleStore)
	sqlCandlesService := candlesv2.NewService(ctx, logger, conf.CandlesV2, sqlCandleStore)
	sqlTradeService := service.NewTrade(sqlstore.NewTrades(sqlConn), logger)
	sqlPositionService := service.NewPosition(sqlstore.NewPositions(sqlConn), logger)
	sqlAssetService := service.NewAsset(sqlstore.NewAssets(sqlConn), logger)
	sqlAccountService := service.NewAccount(sqlstore.NewAccounts(sqlConn), sqlstore.NewBalances(sqlConn), logger)
	sqlRewardsService := service.NewReward(sqlstore.NewRewards(sqlConn), logger)
	sqlMarketsService := service.NewMarkets(sqlstore.NewMarkets(sqlConn), logger)
	sqlDelegationService := service.NewDelegation(sqlstore.NewDelegations(sqlConn), logger)
	sqlEpochService := service.NewEpoch(sqlstore.NewEpochs(sqlConn), logger)
	sqlDepositService := service.NewDeposit(sqlstore.NewDeposits(sqlConn), logger)
	sqlWithdrawalService := service.NewWithdrawal(sqlstore.NewWithdrawals(sqlConn), logger)
	sqlGovernanceService := service.NewGovernance(sqlstore.NewProposals(sqlConn), sqlstore.NewVotes(sqlConn), logger)
	sqlRiskFactorsService := service.NewRiskFactor(sqlstore.NewRiskFactors(sqlConn), logger)
	sqlMarginLevelsService := service.NewRisk(sqlstore.NewMarginLevels(sqlConn), sqlAccountService, logger)
	sqlNetParamService := service.NewNetworkParameter(sqlstore.NewNetworkParameters(sqlConn), logger)
	sqlBlockService := service.NewBlock(sqlstore.NewBlocks(sqlConn), logger)
	sqlCheckpointService := service.NewCheckpoint(sqlstore.NewCheckpoints(sqlConn), logger)
	sqlPartyService := service.NewParty(sqlstore.NewParties(sqlConn), logger)
	sqlOracleSpecService := service.NewOracleSpec(sqlstore.NewOracleSpec(sqlConn), logger)
	sqlOracleDataService := service.NewOracleData(sqlstore.NewOracleData(sqlConn), logger)
	sqlLPDataService := service.NewLiquidityProvision(sqlstore.NewLiquidityProvision(sqlConn, logger), logger)
	sqlTransferService := service.NewTransfer(sqlstore.NewTransfers(sqlConn), logger)
	sqlStakeLinkingService := service.NewStakeLinking(sqlstore.NewStakeLinking(sqlConn), logger)
	sqlNotaryService := service.NewNotary(sqlstore.NewNotary(sqlConn), logger)
	sqlMultiSigService := service.NewMultiSig(sqlstore.NewERC20MultiSigSignerEvent(sqlConn), logger)
	sqlKeyRotationsService := service.NewKeyRotations(sqlstore.NewKeyRotations(sqlConn), logger)
	sqlEthereumKeyRotationService := service.NewEthereumKeyRotation(sqlstore.NewEthereumKeyRotations(sqlConn), logger)
	sqlNodeService := service.NewNode(sqlstore.NewNode(sqlConn), logger)
	sqlMarketDepthService := service.NewMarketDepth(sqlOrderService, logger)
	sqlLedgerService := service.NewLedger(sqlstore.NewLedger(sqlConn), logger)
	sqlProtocolUpgradeService := service.NewProtocolUpgrade(sqlstore.NewProtocolUpgradeProposals(sqlConn), logger)
	sqlCoreSnapshotService := service.NewSnapshotData(sqlstore.NewCoreSnapshotData(sqlConn), logger)

	g := api.NewGRPCServer(
		logger,
		conf.API,
		mockCoreServiceClient,
		eventService,
		sqlOrderService,
		sqlNetworkLimitsService,
		sqlMarketDataService,
		sqlTradeService,
		sqlAssetService,
		sqlAccountService,
		sqlRewardsService,
		sqlMarketsService,
		sqlDelegationService,
		sqlEpochService,
		sqlDepositService,
		sqlWithdrawalService,
		sqlGovernanceService,
		sqlRiskFactorsService,
		sqlMarginLevelsService,
		sqlNetParamService,
		sqlBlockService,
		sqlCheckpointService,
		sqlPartyService,
		sqlCandlesService,
		sqlOracleSpecService,
		sqlOracleDataService,
		sqlLPDataService,
		sqlPositionService,
		sqlTransferService,
		sqlStakeLinkingService,
		sqlNotaryService,
		sqlMultiSigService,
		sqlKeyRotationsService,
		sqlEthereumKeyRotationService,
		sqlNodeService,
		sqlMarketDepthService,
		sqlLedgerService,
		sqlProtocolUpgradeService,
		mockDeHistoryService,
		sqlCoreSnapshotService,
	)
	if g == nil {
		err = fmt.Errorf("failed to create gRPC server")
		return
	}

	tidy = func() {
		mockCtrl.Finish()
		cleanupFn()
	}

	lis := bufconn.Listen(connBufSize)
	ctxDialer := func(context.Context, string) (net.Conn, error) { return lis.Dial() }

	// Start the gRPC server, then wait for it to be ready.
	go func() {
		_ = g.Start(ctx, lis)
	}()

	conn, err = grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(ctxDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to create connection to gRPC server")
	}

	waitForNode(ctx, t, conn)

	return tidy, conn, mockCoreServiceClient, err
}

type dummyConnection struct {
	sqlstore.Connection
}

func (d dummyConnection) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, pgx.ErrNoRows
}

func TestSubmitTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("proxy call is successful", func(t *testing.T) {
		tidy, conn, mockTradingServiceClient, err := getTestGRPCServer(t, ctx)
		if err != nil {
			t.Fatalf("Failed to get test gRPC server: %s", err.Error())
		}
		defer tidy()

		req := &vegaprotoapi.SubmitTransactionRequest{
			Type: vegaprotoapi.SubmitTransactionRequest_TYPE_UNSPECIFIED,
			Tx: &commandspb.Transaction{
				InputData: []byte("input data"),
				Signature: &commandspb.Signature{
					Value:   "value",
					Algo:    "algo",
					Version: 1,
				},
			},
		}

		expectedRes := &vegaprotoapi.SubmitTransactionResponse{Success: true}

		vegaReq := &vegaprotoapi.SubmitTransactionRequest{
			Type: vegaprotoapi.SubmitTransactionRequest_TYPE_UNSPECIFIED,
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
			SubmitTransaction(gomock.Any(), vgtesting.ProtosEq(vegaReq)).
			Return(&vegaprotoapi.SubmitTransactionResponse{Success: true}, nil).Times(1)

		proxyClient := vegaprotoapi.NewCoreServiceClient(conn)
		assert.NotNil(t, proxyClient)

		actualResp, err := proxyClient.SubmitTransaction(ctx, req)
		assert.NoError(t, err)
		vgtesting.AssertProtoEqual(t, expectedRes, actualResp)
	})

	t.Run("proxy propagates an error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tidy, conn, mockTradingServiceClient, err := getTestGRPCServer(t, ctx)
		if err != nil {
			t.Fatalf("Failed to get test gRPC server: %s", err.Error())
		}
		defer tidy()

		req := &vegaprotoapi.SubmitTransactionRequest{
			Type: vegaprotoapi.SubmitTransactionRequest_TYPE_COMMIT,
			Tx: &commandspb.Transaction{
				InputData: []byte("input data"),
				Signature: &commandspb.Signature{
					Value:   "value",
					Algo:    "algo",
					Version: 1,
				},
			},
		}

		vegaReq := &vegaprotoapi.SubmitTransactionRequest{
			Type: vegaprotoapi.SubmitTransactionRequest_TYPE_COMMIT,
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
			SubmitTransaction(gomock.Any(), vgtesting.ProtosEq(vegaReq)).
			Return(nil, errors.New("Critical error"))

		proxyClient := vegaprotoapi.NewCoreServiceClient(conn)
		assert.NotNil(t, proxyClient)

		actualResp, err := proxyClient.SubmitTransaction(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, actualResp)
		assert.Contains(t, err.Error(), "Critical error")
	})
}

func TestSubmitRawTransaction(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("proxy call is successful", func(t *testing.T) {
		tidy, conn, mockTradingServiceClient, err := getTestGRPCServer(t, ctx)
		if err != nil {
			t.Fatalf("Failed to get test gRPC server: %s", err.Error())
		}
		defer tidy()

		tx := &commandspb.Transaction{
			InputData: []byte("input data"),
			Signature: &commandspb.Signature{
				Value:   "value",
				Algo:    "algo",
				Version: 1,
			},
		}

		bs, err := proto.Marshal(tx)
		assert.NoError(t, err)

		req := &vegaprotoapi.SubmitRawTransactionRequest{
			Type: vegaprotoapi.SubmitRawTransactionRequest_TYPE_UNSPECIFIED,
			Tx:   bs,
		}

		expectedRes := &vegaprotoapi.SubmitRawTransactionResponse{Success: true}

		vegaReq := &vegaprotoapi.SubmitRawTransactionRequest{
			Type: vegaprotoapi.SubmitRawTransactionRequest_TYPE_UNSPECIFIED,
			Tx:   bs,
		}

		mockTradingServiceClient.EXPECT().
			SubmitRawTransaction(gomock.Any(), vgtesting.ProtosEq(vegaReq)).
			Return(&vegaprotoapi.SubmitRawTransactionResponse{Success: true}, nil).Times(1)

		proxyClient := vegaprotoapi.NewCoreServiceClient(conn)
		assert.NotNil(t, proxyClient)

		actualResp, err := proxyClient.SubmitRawTransaction(ctx, req)
		assert.NoError(t, err)
		vgtesting.AssertProtoEqual(t, expectedRes, actualResp)
	})

	t.Run("proxy propagates an error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tidy, conn, mockTradingServiceClient, err := getTestGRPCServer(t, ctx)
		if err != nil {
			t.Fatalf("Failed to get test gRPC server: %s", err.Error())
		}
		defer tidy()
		tx := &commandspb.Transaction{
			InputData: []byte("input data"),
			Signature: &commandspb.Signature{
				Value:   "value",
				Algo:    "algo",
				Version: 1,
			},
		}

		bs, err := proto.Marshal(tx)
		assert.NoError(t, err)

		req := &vegaprotoapi.SubmitRawTransactionRequest{
			Type: vegaprotoapi.SubmitRawTransactionRequest_TYPE_COMMIT,
			Tx:   bs,
		}

		vegaReq := &vegaprotoapi.SubmitRawTransactionRequest{
			Type: vegaprotoapi.SubmitRawTransactionRequest_TYPE_COMMIT,
			Tx:   bs,
		}

		mockTradingServiceClient.EXPECT().
			SubmitRawTransaction(gomock.Any(), vgtesting.ProtosEq(vegaReq)).
			Return(nil, errors.New("Critical error"))

		proxyClient := vegaprotoapi.NewCoreServiceClient(conn)
		assert.NotNil(t, proxyClient)

		actualResp, err := proxyClient.SubmitRawTransaction(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, actualResp)
		assert.Contains(t, err.Error(), "Critical error")
	})
}

func TestLastBlockHeight(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("proxy call is successful", func(t *testing.T) {
		tidy, conn, mockTradingServiceClient, err := getTestGRPCServer(t, ctx)
		if err != nil {
			t.Fatalf("Failed to get test gRPC server: %s", err.Error())
		}
		defer tidy()

		req := &vegaprotoapi.LastBlockHeightRequest{}
		expectedRes := &vegaprotoapi.LastBlockHeightResponse{Height: 20}

		vegaReq := &vegaprotoapi.LastBlockHeightRequest{}

		mockTradingServiceClient.EXPECT().
			LastBlockHeight(gomock.Any(), vgtesting.ProtosEq(vegaReq)).
			Return(&vegaprotoapi.LastBlockHeightResponse{Height: 20}, nil).Times(1)

		proxyClient := vegaprotoapi.NewCoreServiceClient(conn)
		assert.NotNil(t, proxyClient)

		actualResp, err := proxyClient.LastBlockHeight(ctx, req)
		assert.NoError(t, err)
		vgtesting.AssertProtoEqual(t, expectedRes, actualResp)
	})

	t.Run("proxy propagates an error", func(t *testing.T) {
		tidy, conn, mockTradingServiceClient, err := getTestGRPCServer(t, ctx)
		if err != nil {
			t.Fatalf("Failed to get test gRPC server: %s", err.Error())
		}
		defer tidy()

		req := &vegaprotoapi.LastBlockHeightRequest{}
		vegaReq := &vegaprotoapi.LastBlockHeightRequest{}

		mockTradingServiceClient.EXPECT().
			LastBlockHeight(gomock.Any(), vgtesting.ProtosEq(vegaReq)).
			Return(nil, fmt.Errorf("Critical error")).Times(1)

		proxyClient := vegaprotoapi.NewCoreServiceClient(conn)
		assert.NotNil(t, proxyClient)

		actualResp, err := proxyClient.LastBlockHeight(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, actualResp)
		assert.Contains(t, err.Error(), "Critical error")
	})
}
