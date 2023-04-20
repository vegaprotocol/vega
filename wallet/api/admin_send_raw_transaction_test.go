package api_test

import (
	"context"
	"errors"
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

func TestAdminSendRawTransaction(t *testing.T) {
	t.Run("Documentation matches the code", testAdminSendRawTransactionSchemaCorrect)
	t.Run("Sending transaction with invalid params fails", testAdminSendingRawTransactionWithInvalidParamsFails)
	t.Run("Sending transaction with valid params succeeds", testAdminSendingRawTransactionWithValidParamsSucceeds)
	t.Run("Sending transaction with network that doesn't exist fails", testAdminSendingRawTransactionWithNetworkThatDoesntExistFails)
	t.Run("Sending transaction with network that fails existence check fails", testAdminSendingRawTransactionWithNetworkThatFailsExistenceCheckFails)
	t.Run("Sending transaction with failure to get network", testAdminSendingRawTransactionWithFailureToGetNetworkFails)
	t.Run("Getting internal error during node selector building fails", testAdminSendingRawTransactionGettingInternalErrorDuringNodeSelectorBuildingFails)
	t.Run("Sending transaction without healthy node fails", testAdminSendingRawTransactionWithoutHealthyNodeFails)
	t.Run("Sending transaction with failed sending fails", testAdminSendingRawTransactionWithFailedSendingFails)
}

func testAdminSendRawTransactionSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.send_raw_transaction", api.AdminSendRawTransactionParams{}, api.AdminSendRawTransactionResult{})
}

func testAdminSendingRawTransactionWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminSendRawTransactionParams{
				Network:     "fairground",
				SendingMode: "",
			},
			expectedError: api.ErrSendingModeIsRequired,
		},
		{
			name: "with empty transaction",
			params: api.AdminSendRawTransactionParams{
				Network:            "fairground",
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "",
			},
			expectedError: api.ErrEncodedTransactionIsRequired,
		},
		{
			name: "with non-base64 transaction",
			params: api.AdminSendRawTransactionParams{
				Network:            "fairground",
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "1234567890",
			},
			expectedError: api.ErrEncodedTransactionIsNotValidBase64String,
		},
		{
			name: "with network and node address",
			params: api.AdminSendRawTransactionParams{
				Network:            "fairground",
				NodeAddress:        "localhost:3002",
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
			},
			expectedError: api.ErrSpecifyingNetworkAndNodeAddressIsNotSupported,
		},
		{
			name: "with network and node address missing",
			params: api.AdminSendRawTransactionParams{
				SendingMode:        "TYPE_SYNC",
				EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
			},
			expectedError: api.ErrNetworkOrNodeAddressIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newAdminSendRawTransactionHandler(tt, unexpectedNodeSelectorCall(tt))

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
			assert.Empty(tt, result)
		})
	}
}

func testAdminSendingRawTransactionWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	sendingMode := "TYPE_SYNC"
	network := newNetwork(t)
	txHash := vgrand.RandomStr(64)
	nodeHost := vgrand.RandomStr(5)

	// setup
	handler := newAdminSendRawTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(node, nil)
		node.EXPECT().Host().Times(1).Return(nodeHost)
		node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return(txHash, nil)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendRawTransactionParams{
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

func testAdminSendingRawTransactionWithNetworkThatDoesntExistFails(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newAdminSendRawTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(gomock.Any()).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendRawTransactionParams{
		Network:            "fairground",
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertInvalidParams(t, errorDetails, api.ErrNetworkDoesNotExist)
	assert.Empty(t, result)
}

func testAdminSendingRawTransactionWithNetworkThatFailsExistenceCheckFails(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newAdminSendRawTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(gomock.Any()).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendRawTransactionParams{
		Network:            "fairground",
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not determine if the network exists: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSendingRawTransactionWithFailureToGetNetworkFails(t *testing.T) {
	// given
	ctx := context.Background()
	network := "fairground"

	// setup
	handler := newAdminSendRawTransactionHandler(t, unexpectedNodeSelectorCall(t))

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendRawTransactionParams{
		Network:            network,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the network configuration: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSendingRawTransactionGettingInternalErrorDuringNodeSelectorBuildingFails(t *testing.T) {
	// given
	ctx := context.Background()
	network := newNetwork(t)

	// setup
	handler := newAdminSendRawTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		return nil, assert.AnError
	})

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendRawTransactionParams{
		Network:            network.Name,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertInternalError(t, errorDetails, fmt.Errorf("could not initialize the node selector: %w", assert.AnError))
	assert.Empty(t, result)
}

func testAdminSendingRawTransactionWithoutHealthyNodeFails(t *testing.T) {
	// given
	ctx := context.Background()
	network := newNetwork(t)

	// setup
	handler := newAdminSendRawTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(nil, assert.AnError)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendRawTransactionParams{
		Network:            network.Name,
		SendingMode:        "TYPE_SYNC",
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertNetworkError(t, errorDetails, api.ErrNoHealthyNodeAvailable)
	assert.Empty(t, result)
}

func testAdminSendingRawTransactionWithFailedSendingFails(t *testing.T) {
	// given
	ctx := context.Background()
	sendingMode := "TYPE_SYNC"
	network := newNetwork(t)

	// setup
	handler := newAdminSendRawTransactionHandler(t, func(hosts []string, retries uint64) (walletnode.Selector, error) {
		ctrl := gomock.NewController(t)
		nodeSelector := nodemocks.NewMockSelector(ctrl)
		node := nodemocks.NewMockNode(ctrl)
		nodeSelector.EXPECT().Node(ctx, gomock.Any()).Times(1).Return(node, nil)
		node.EXPECT().SendTransaction(ctx, gomock.Any(), apipb.SubmitTransactionRequest_TYPE_SYNC).Times(1).Return("", assert.AnError)
		return nodeSelector, nil
	})

	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminSendRawTransactionParams{
		Network:            network.Name,
		SendingMode:        sendingMode,
		EncodedTransaction: "Cip0ZXN0bmV0LWU5MGU2NwAI85ubpLO4mOnIARCEyogByrsBBwgCEgN7fQoSkwEKgAFlY2E4YjQ1MzNhNGNiZmFkY2VlMGNhYmZlNjdmMWRjZTAwN2RlODFlZjFlMTE3YTM4ZWVhMDJmYTNlMTcxMWM5NzI3YTQ3MmM3ZmNiNzU3ZDJmNTE4MTIxZTg2MzNiNjNlNTNmMWZjNjY0MTA1NjhmYjI5ODBmNDc4NjhiOTIwNRIMdmVnYS9lZDI1NTE5GAGAfQPCuwFGCkAxMjUzOGU0OTQ0ZjhjOWQ4MmU4MDNlNDE2YjM0MGQ2YmE0Mzk0NDIyZWQ1YWVmYmM2ZDYwNzYyZTcxMGFhNzk0ENLQAtI+QDNmZDQyZmQ1Y2ViMjJkOTlhYzQ1MDg2ZjFkODJkNTE2MTE4YTVjYjdhZDlhMmUwOTZjZDc4Y2EyYzg5NjBjODA=",
	})

	// then
	assertNetworkError(t, errorDetails, errors.New("the transaction failed: assert.AnError general error for testing"))
	assert.Empty(t, result)
}

type adminSendRawTransactionHandler struct {
	*api.AdminSendRawTransaction
	ctrl         *gomock.Controller
	networkStore *mocks.MockNetworkStore
}

func (h *adminSendRawTransactionHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminSendRawTransactionResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminSendRawTransactionResult)
		if !ok {
			t.Fatal("AdminUpdatePermissions handler result is not a AdminSignTransactionResult")
		}
		return result, err
	}
	return api.AdminSendRawTransactionResult{}, err
}

func newAdminSendRawTransactionHandler(t *testing.T, builder api.NodeSelectorBuilder) *adminSendRawTransactionHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &adminSendRawTransactionHandler{
		AdminSendRawTransaction: api.NewAdminSendRawTransaction(networkStore, builder),
		ctrl:                    ctrl,
		networkStore:            networkStore,
	}
}
