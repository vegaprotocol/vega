package api_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	walletnode "code.vegaprotocol.io/vega/wallet/api/node"
	nodemocks "code.vegaprotocol.io/vega/wallet/api/node/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAdminSendTransaction(t *testing.T) {
	t.Run("Sending transaction with invalid params fails", testAdminSendingTransactionWithInvalidParamsFails)
	t.Run("Sending transaction with valid params succeeds", testAdminSendingTransactionWithValidParamsSucceeds)
	t.Run("Sending transaction with network that doesn't exist fails", testAdminSendingTransactionWithNetworkThatDoesntExistFails)
	t.Run("Sending transaction with network that fails existence check fails", testAdminSendingTransactionWithNetworkThatFailsExistenceCheckFails)
	t.Run("Sending transaction with failure to get network", testAdminSendingTransactionWithFailureToGetNetworkFails)
	t.Run("Getting internal error during node selector building fails", testAdminSendingTransactionGettingInternalErrorDuringNodeSelectorBuildingFails)
	t.Run("Sending transaction without healthy node fails", testAdminSendingTransactionWithoutHealthyNodeFails)
	t.Run("Sending transaction with failed sending fails", testAdminSendingTransactionWithFailedSendingFails)
}

func testAdminSendingTransactionWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with nil params",
			params:        nil,
			expectedError: api.ErrParamsRequired,
		},
		{
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		},
		{
			name: "with empty sending mode",
			params: api.AdminSendTransactionParams{
				Network:     "fairground",
				SendingMode: "",
			},
			expectedError: api.ErrSendingModeIsRequired,
		},
		{
			name: "with empty transaction",
			params: api.AdminSendTransactionParams{
				Network:            "fairground",
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "",
			},
			expectedError: api.ErrEncodedTransactionIsRequired,
		},
		{
			name: "with non-base64 transaction",
			params: api.AdminSendTransactionParams{
				Network:            "fairground",
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "1234567890",
			},
			expectedError: api.ErrEncodedTransactionIsNotValidBase64String,
		},
		{
			name: "with network and node address",
			params: api.AdminSendTransactionParams{
				Network:            "fairground",
				NodeAddress:        "localhost:3002",
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
			},
			expectedError: api.ErrSpecifyingNetworkAndNodeAddressIsNotSupported,
		},
		{
			name: "with network and node address missing",
			params: api.AdminSendTransactionParams{
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
			},
			expectedError: api.ErrNetworkOrNodeAddressIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := contextWithTraceID()

			// setup
			handler := newAdminSendTransactionHandler(tt, unexpectedNodeSelectorCall(tt))

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
			assert.Empty(tt, result)
		})
	}
}

func testAdminSendingTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	sendingMode := "TYPE_SYNC"
	network := newNetwork(t)
	txHash := vgrand.RandomStr(64)

	// setup
	handler := newAdminSendTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx).Times(1).Return(node, nil)
		node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return(txHash, nil)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Network:            network.Name,
		SendingMode:        sendingMode,
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assert.Nil(t, errorDetails)
	assert.NotEmpty(t, result.Tx)
	assert.Equal(t, txHash, result.TxHash)
	assert.NotEmpty(t, result.ReceivedAt)
	assert.NotEmpty(t, result.SentAt)
}

func testAdminSendingTransactionWithNetworkThatDoesntExistFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()

	// setup
	handler := newAdminSendTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(gomock.Any()).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Network:            "fairground",
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertInvalidParams(t, errorDetails, api.ErrNetworkDoesNotExist)
	assert.Empty(t, result)
}

func testAdminSendingTransactionWithNetworkThatFailsExistenceCheckFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()

	// setup
	handler := newAdminSendTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(gomock.Any()).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Network:            "fairground",
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not check the network existence: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSendingTransactionWithFailureToGetNetworkFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	network := "fairground"

	// setup
	handler := newAdminSendTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Network:            network,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the network configuration: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSendingTransactionGettingInternalErrorDuringNodeSelectorBuildingFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	network := newNetwork(t)

	// setup
	handler := newAdminSendTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		return nil, assert.AnError
	})

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Network:            network.Name,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not initializing the node selector: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSendingTransactionWithoutHealthyNodeFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	network := newNetwork(t)

	// setup
	handler := newAdminSendTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		nodeSelector.EXPECT().Node(ctx).Times(1).Return(nil, assert.AnError)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Network:            network.Name,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertNetworkError(t, errorDetails, api.ErrNoHealthyNodeAvailable)
	assert.Empty(t, result)
}

func testAdminSendingTransactionWithFailedSendingFails(t *testing.T) {
	// given
	ctx, _ := contextWithTraceID()
	sendingMode := "TYPE_SYNC"
	network := newNetwork(t)

	// setup
	handler := newAdminSendTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx).Times(1).Return(node, nil)
		node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return("", assert.AnError)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendTransactionParams{
		Network:            network.Name,
		SendingMode:        sendingMode,
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertNetworkError(t, errorDetails, fmt.Errorf("the transaction failed: %w", assert.AnError))
	assert.Empty(t, result)
}

type adminSendTransactionHandler struct {
	*api.AdminSendTransaction
	ctrl         *gomock.Controller
	networkStore *mocks.MockNetworkStore
}

func (h *adminSendTransactionHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminSendTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminSendTransactionResult)
		if !ok {
			t.Fatal("AdminUpdatePermissions handler result is not a AdminSignTransactionResult")
		}
		return result, err
	}
	return api.AdminSendTransactionResult{}, err
}

func newAdminSendTransactionHandler(t *testing.T, builder api.NodeSelectorBuilder) *adminSendTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &adminSendTransactionHandler{
		AdminSendTransaction: api.NewAdminSendTransaction(networkStore, builder),
		ctrl:                 ctrl,
		networkStore:         networkStore,
	}
}
