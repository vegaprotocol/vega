package node_test

import (
	"context"
	"fmt"
	"testing"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"code.vegaprotocol.io/vega/wallet/api/node/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryingNode_Statistics(t *testing.T) {
	t.Run("Getting statistics is not retried", testRetryingNodeStatisticsNotRetried)
	t.Run("Getting statistics succeeds", testRetryingNodeStatisticsSucceeds)
}

func testRetryingNodeStatisticsNotRetried(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	adapter.EXPECT().Statistics(ctx).Times(1).Return(types.Statistics{}, assert.AnError)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	response, err := retryingNode.Statistics(ctx)

	// then
	require.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, response)
}

func testRetryingNodeStatisticsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	statistics := types.Statistics{
		BlockHash:   vgrand.RandomStr(5),
		BlockHeight: 123456,
		ChainID:     vgrand.RandomStr(5),
		VegaTime:    vgrand.RandomStr(5),
	}
	adapter.EXPECT().Statistics(ctx).Times(1).Return(statistics, nil)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	response, err := retryingNode.Statistics(ctx)

	// then
	require.NoError(t, err)
	assert.Equal(t, statistics, response)
}

func TestRetryingNode_LastBlock(t *testing.T) {
	t.Run("Retrying with one successful call succeeds", testRetryingNodeLastBlockRetryingWithOneSuccessfulCallSucceeds)
	t.Run("Retrying without successful calls fails", testRetryingNodeLastBlockRetryingWithoutSuccessfulCallsFails)
}

func testRetryingNodeLastBlockRetryingWithOneSuccessfulCallSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	expectedResponse := types.LastBlock{
		BlockHeight:             123,
		BlockHash:               vgrand.RandomStr(5),
		ProofOfWorkHashFunction: vgrand.RandomStr(5),
		ProofOfWorkDifficulty:   432,
	}
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	unsuccessfulCalls := adapter.EXPECT().LastBlock(ctx).Times(2).Return(types.LastBlock{}, assert.AnError)
	successfulCall := adapter.EXPECT().LastBlock(ctx).Times(1).Return(expectedResponse, nil)
	gomock.InOrder(unsuccessfulCalls, successfulCall)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	response, err := retryingNode.LastBlock(ctx)

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func testRetryingNodeLastBlockRetryingWithoutSuccessfulCallsFails(t *testing.T) {
	// given
	ctx := context.Background()
	log := newTestLogger(t)

	// setup
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	adapter.EXPECT().LastBlock(ctx).Times(4).Return(types.LastBlock{}, assert.AnError)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	nodeID, err := retryingNode.LastBlock(ctx)

	// then
	require.Error(t, err, assert.AnError)
	assert.Empty(t, nodeID)
}

func TestRetryingNode_SendTransaction(t *testing.T) {
	t.Run("Retrying with one successful call succeeds", testRetryingNodeSendTransactionRetryingWithOneSuccessfulCallSucceeds)
	t.Run("Retrying with a successful call but unsuccessful transaction fails", testRetryingNodeSendTransactionWithSuccessfulCallBuUnsuccessfulTxFails)
	t.Run("Retrying without successful calls fails", testRetryingNodeSendTransactionRetryingWithoutSuccessfulCallsFails)
}

func testRetryingNodeSendTransactionRetryingWithOneSuccessfulCallSucceeds(t *testing.T) {
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
			Tid:   vgrand.RandomStr(5),
			Nonce: 23214,
		},
	}

	// setup
	request := &apipb.SubmitTransactionRequest{
		Tx:   tx,
		Type: apipb.SubmitTransactionRequest_TYPE_SYNC,
	}
	expectedResponse := &apipb.SubmitTransactionResponse{
		Success: true,
		TxHash:  expectedTxHash,
	}
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	unsuccessfulCalls := adapter.EXPECT().SubmitTransaction(ctx, request).Times(2).Return(nil, assert.AnError)
	successfulCall := adapter.EXPECT().SubmitTransaction(ctx, request).Times(1).Return(expectedResponse, nil)
	gomock.InOrder(unsuccessfulCalls, successfulCall)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	response, err := retryingNode.SendTransaction(ctx, tx, apipb.SubmitTransactionRequest_TYPE_SYNC)

	// then
	require.NoError(t, err)
	assert.Equal(t, expectedResponse.TxHash, response)
}

func testRetryingNodeSendTransactionWithSuccessfulCallBuUnsuccessfulTxFails(t *testing.T) {
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
			Tid:   vgrand.RandomStr(5),
			Nonce: 23214,
		},
	}

	// setup
	request := &apipb.SubmitTransactionRequest{
		Tx:   tx,
		Type: apipb.SubmitTransactionRequest_TYPE_SYNC,
	}
	expectedResponse := &apipb.SubmitTransactionResponse{
		Success: false,
		TxHash:  expectedTxHash,
		Code:    42,
		Data:    vgrand.RandomStr(10),
	}
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	unsuccessfulCalls := adapter.EXPECT().SubmitTransaction(ctx, request).Times(2).Return(nil, assert.AnError)
	successfulCall := adapter.EXPECT().SubmitTransaction(ctx, request).Times(1).Return(expectedResponse, nil)
	gomock.InOrder(unsuccessfulCalls, successfulCall)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	response, err := retryingNode.SendTransaction(ctx, tx, apipb.SubmitTransactionRequest_TYPE_SYNC)

	// then
	require.EqualError(t, err, fmt.Sprintf("%s (ABCI code %d)", expectedResponse.Data, expectedResponse.Code))
	assert.Empty(t, response)
}

func testRetryingNodeSendTransactionRetryingWithoutSuccessfulCallsFails(t *testing.T) {
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
			Tid:   vgrand.RandomStr(5),
			Nonce: 23214,
		},
	}

	// setup
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	adapter.EXPECT().SubmitTransaction(ctx, &apipb.SubmitTransactionRequest{
		Tx:   tx,
		Type: apipb.SubmitTransactionRequest_TYPE_SYNC,
	}).Times(4).Return(nil, assert.AnError)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	nodeID, err := retryingNode.SendTransaction(ctx, tx, apipb.SubmitTransactionRequest_TYPE_SYNC)

	// then
	require.Error(t, err, assert.AnError)
	assert.Empty(t, nodeID)
}

func TestRetryingNode_Stop(t *testing.T) {
	t.Run("Stopping the node closes the underlying adapter", testRetryingNodeStoppingNodeClosesUnderlyingAdapter)
	t.Run("Stopping the node returns the underlying adapter error if any", testRetryingNodeStoppingNodeReturnUnderlyingErrorIfAny)
}

func testRetryingNodeStoppingNodeClosesUnderlyingAdapter(t *testing.T) {
	// given
	log := newTestLogger(t)

	// setup
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	adapter.EXPECT().Stop().Times(1).Return(nil)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	err := retryingNode.Stop()

	// then
	require.NoError(t, err)
}

func testRetryingNodeStoppingNodeReturnUnderlyingErrorIfAny(t *testing.T) {
	// given
	log := newTestLogger(t)

	// setup
	adapter := newGRPCAdapterMock(t)
	adapter.EXPECT().Host().AnyTimes().Return("test-client")
	adapter.EXPECT().Stop().Times(1).Return(assert.AnError)

	// when
	retryingNode := node.BuildRetryingNode(log, adapter, 3)
	err := retryingNode.Stop()

	// then
	require.EqualError(t, err, fmt.Errorf("could not close properly stop the gRPC API client: %w", assert.AnError).Error())
}

func newGRPCAdapterMock(t *testing.T) *mocks.MockGRPCAdapter {
	t.Helper()
	ctrl := gomock.NewController(t)
	grpcAdapter := mocks.NewMockGRPCAdapter(ctrl)
	return grpcAdapter
}
