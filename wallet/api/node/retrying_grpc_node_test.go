package node_test

import (
	"context"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryingGRPCNode_HealthCheck(t *testing.T) {
	t.Run("Retrying with one successful call succeeds", testRetryingGRPCNodeHealthCheckRetryingWithOneSuccessfulCallSucceeds)
	t.Run("Retrying without successful calls fails", testRetryingGRPCNodeHealthCheckRetryingWithoutSuccessfulCallsFails)
}

func testRetryingGRPCNodeHealthCheckRetryingWithOneSuccessfulCallSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	request := &apipb.GetVegaTimeRequest{}
	unsuccessfulCalls := client.EXPECT().GetVegaTime(ctx, request).Times(2).Return(nil, assert.AnError)
	successfulCall := client.EXPECT().GetVegaTime(ctx, request).Times(1).Return(&apipb.GetVegaTimeResponse{
		Timestamp: 1234,
	}, nil)
	gomock.InOrder(unsuccessfulCalls, successfulCall)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	err := grpcNode.HealthCheck(ctx)

	// then
	require.NoError(t, err)
}

func testRetryingGRPCNodeHealthCheckRetryingWithoutSuccessfulCallsFails(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	ctrl := gomock.NewController(t)
	client := mocks.NewMockCoreClient(ctrl)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	client.EXPECT().GetVegaTime(ctx, &apipb.GetVegaTimeRequest{}).Times(4).Return(nil, assert.AnError)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	err := grpcNode.HealthCheck(ctx)

	// then
	require.Error(t, err, assert.AnError)
}

func TestRetryingGRPCNode_NetworkChainID(t *testing.T) {
	t.Run("Retrying with one successful call succeeds", testRetryingGRPCNodeNetworkChainIDRetryingWithOneSuccessfulCallSucceeds)
	t.Run("Retrying without successful calls fails", testRetryingGRPCNodeNetworkChainIDRetryingWithoutSuccessfulCallsFails)
}

func testRetryingGRPCNodeNetworkChainIDRetryingWithOneSuccessfulCallSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)
	expectedNodeID := vgrand.RandomStr(5)

	// setup
	request := &apipb.StatisticsRequest{}
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	unsuccessfulCalls := client.EXPECT().Statistics(ctx, request).Times(2).Return(nil, assert.AnError)
	successfulCall := client.EXPECT().Statistics(ctx, request).Times(1).Return(&apipb.StatisticsResponse{
		Statistics: &apipb.Statistics{
			ChainId: expectedNodeID,
		},
	}, nil)
	gomock.InOrder(unsuccessfulCalls, successfulCall)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	nodeID, err := grpcNode.NetworkChainID(ctx)

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedNodeID, nodeID)
}

func testRetryingGRPCNodeNetworkChainIDRetryingWithoutSuccessfulCallsFails(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	client.EXPECT().Statistics(ctx, &apipb.StatisticsRequest{}).Times(4).Return(nil, assert.AnError)
	grpcNode := node.BuildGRPCNode(log, client, 3)

	// when
	nodeID, err := grpcNode.NetworkChainID(ctx)

	// then
	require.Error(t, err, assert.AnError)
	assert.Empty(t, nodeID)
}

func TestRetryingGRPCNode_LastBlock(t *testing.T) {
	t.Run("Retrying with one successful call succeeds", testRetryingGRPCNodeLastBlockRetryingWithOneSuccessfulCallSucceeds)
	t.Run("Retrying without successful calls fails", testRetryingGRPCNodeLastBlockRetryingWithoutSuccessfulCallsFails)
}

func testRetryingGRPCNodeLastBlockRetryingWithOneSuccessfulCallSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	request := &apipb.LastBlockHeightRequest{}
	expectedResponse := &apipb.LastBlockHeightResponse{
		Height:                      123,
		Hash:                        vgrand.RandomStr(5),
		SpamPowHashFunction:         vgrand.RandomStr(5),
		SpamPowDifficulty:           432,
		SpamPowNumberOfPastBlocks:   7689,
		SpamPowNumberOfTxPerBlock:   8987656,
		SpamPowIncreasingDifficulty: true,
	}
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	unsuccessfulCalls := client.EXPECT().LastBlockHeight(ctx, request).Times(2).Return(nil, assert.AnError)
	successfulCall := client.EXPECT().LastBlockHeight(ctx, request).Times(1).Return(expectedResponse, nil)
	gomock.InOrder(unsuccessfulCalls, successfulCall)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	response, err := grpcNode.LastBlock(ctx)

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func testRetryingGRPCNodeLastBlockRetryingWithoutSuccessfulCallsFails(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	client.EXPECT().LastBlockHeight(ctx, &apipb.LastBlockHeightRequest{}).Times(4).Return(nil, assert.AnError)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	nodeID, err := grpcNode.LastBlock(ctx)

	// then
	require.Error(t, err, assert.AnError)
	assert.Empty(t, nodeID)
}

func TestRetryingGRPCNode_CheckTransaction(t *testing.T) {
	t.Run("Retrying with one successful call succeeds", testRetryingGRPCNodeCheckTransactionRetryingWithOneSuccessfulCallSucceeds)
	t.Run("Retrying without successful calls fails", testRetryingGRPCNodeCheckTransactionRetryingWithoutSuccessfulCallsFails)
}

func testRetryingGRPCNodeCheckTransactionRetryingWithOneSuccessfulCallSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)
	tx := &commandspb.Transaction{
		Version:   3,
		InputData: []byte{},
		Signature: &commandspb.Signature{
			Value:   "345678",
			Algo:    vgrand.RandomStr(5),
			Version: 2,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: vgrand.RandomStr(5),
		},
		Pow: &commandspb.ProofOfWork{
			Tid:          vgrand.RandomStr(5),
			Nonce:        23214,
			HashFunction: vgrand.RandomStr(5),
		},
	}

	// setup
	request := &apipb.CheckTransactionRequest{
		Tx: tx,
	}
	expectedResponse := &apipb.CheckTransactionResponse{
		Success:   true,
		Code:      123,
		GasWanted: 345678,
		GasUsed:   9432356,
	}
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	unsuccessfulCalls := client.EXPECT().CheckTransaction(ctx, request).Times(2).Return(nil, assert.AnError)
	successfulCall := client.EXPECT().CheckTransaction(ctx, request).Times(1).Return(expectedResponse, nil)
	gomock.InOrder(unsuccessfulCalls, successfulCall)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	response, err := grpcNode.CheckTransaction(ctx, tx)

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func testRetryingGRPCNodeCheckTransactionRetryingWithoutSuccessfulCallsFails(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)
	tx := &commandspb.Transaction{
		Version:   3,
		InputData: []byte{},
		Signature: &commandspb.Signature{
			Value:   "345678",
			Algo:    vgrand.RandomStr(5),
			Version: 2,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: vgrand.RandomStr(5),
		},
		Pow: &commandspb.ProofOfWork{
			Tid:          vgrand.RandomStr(5),
			Nonce:        23214,
			HashFunction: vgrand.RandomStr(5),
		},
	}

	// setup
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	client.EXPECT().CheckTransaction(ctx, &apipb.CheckTransactionRequest{
		Tx: tx,
	}).Times(4).Return(nil, assert.AnError)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	nodeID, err := grpcNode.CheckTransaction(ctx, tx)

	// then
	require.Error(t, err, assert.AnError)
	assert.Empty(t, nodeID)
}

func TestRetryingGRPCNode_SendTransaction(t *testing.T) {
	t.Run("Retrying with one successful call succeeds", testRetryingGRPCNodeSendTransactionRetryingWithOneSuccessfulCallSucceeds)
	t.Run("Retrying without successful calls fails", testRetryingGRPCNodeSendTransactionRetryingWithoutSuccessfulCallsFails)
}

func testRetryingGRPCNodeSendTransactionRetryingWithOneSuccessfulCallSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)
	expectedTxHash := vgrand.RandomStr(10)
	tx := &commandspb.Transaction{
		Version:   3,
		InputData: []byte{},
		Signature: &commandspb.Signature{
			Value:   "345678",
			Algo:    vgrand.RandomStr(5),
			Version: 2,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: vgrand.RandomStr(5),
		},
		Pow: &commandspb.ProofOfWork{
			Tid:          vgrand.RandomStr(5),
			Nonce:        23214,
			HashFunction: vgrand.RandomStr(5),
		},
	}

	// setup
	request := &apipb.SubmitTransactionRequest{
		Tx:   tx,
		Type: apipb.SubmitTransactionRequest_TYPE_SYNC,
	}
	expectedResponse := &apipb.SubmitTransactionResponse{
		TxHash: expectedTxHash,
	}
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	unsuccessfulCalls := client.EXPECT().SubmitTransaction(ctx, request).Times(2).Return(nil, assert.AnError)
	successfulCall := client.EXPECT().SubmitTransaction(ctx, request).Times(1).Return(expectedResponse, nil)
	gomock.InOrder(unsuccessfulCalls, successfulCall)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	response, err := grpcNode.SendTransaction(ctx, tx, apipb.SubmitTransactionRequest_TYPE_SYNC)

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedResponse.TxHash, response)
}

func testRetryingGRPCNodeSendTransactionRetryingWithoutSuccessfulCallsFails(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)
	tx := &commandspb.Transaction{
		Version:   3,
		InputData: []byte{},
		Signature: &commandspb.Signature{
			Value:   "345678",
			Algo:    vgrand.RandomStr(5),
			Version: 2,
		},
		From: &commandspb.Transaction_PubKey{
			PubKey: vgrand.RandomStr(5),
		},
		Pow: &commandspb.ProofOfWork{
			Tid:          vgrand.RandomStr(5),
			Nonce:        23214,
			HashFunction: vgrand.RandomStr(5),
		},
	}

	// setup
	client := newClientMock(t)
	client.EXPECT().Host().AnyTimes().Return("test-client")
	client.EXPECT().SubmitTransaction(ctx, &apipb.SubmitTransactionRequest{
		Tx:   tx,
		Type: apipb.SubmitTransactionRequest_TYPE_SYNC,
	}).Times(4).Return(nil, assert.AnError)

	// when
	grpcNode := node.BuildGRPCNode(log, client, 3)
	nodeID, err := grpcNode.SendTransaction(ctx, tx, apipb.SubmitTransactionRequest_TYPE_SYNC)

	// then
	require.Error(t, err, assert.AnError)
	assert.Empty(t, nodeID)
}

func newClientMock(t *testing.T) *mocks.MockCoreClient {
	t.Helper()
	ctrl := gomock.NewController(t)
	client := mocks.NewMockCoreClient(ctrl)
	return client
}
