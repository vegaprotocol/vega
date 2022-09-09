package api_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminDescribeNetwork(t *testing.T) {
	t.Run("Describing a network with invalid params fails", testDescribingNetworkWithInvalidParamsFails)
	t.Run("Describing a network that does not exists fails", testDescribingNetworkThatDoesNotExistsFails)
	t.Run("Getting internal error during verification fails", testGettingInternalErrorDuringNetworkVerificationFails)
	t.Run("Getting internal error during retrieval fails", testGettingInternalErrorDuringNetworkRetrievalFails)
}

func testDescribingNetworkWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty passphrase",
			params: api.AdminDescribeNetworkParams{
				Network: "",
			},
			expectedError: api.ErrNetworkIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := contextWithTraceID()

			// setup
			handler := newDescribeNetworkHandler(tt)
			// -- unexpected calls
			handler.networkStore.EXPECT().ListNetworks().Times(0)
			handler.networkStore.EXPECT().NetworkExists(gomock.Any()).Times(0)
			handler.networkStore.EXPECT().GetNetwork(gomock.Any()).Times(0)
			handler.networkStore.EXPECT().SaveNetwork(gomock.Any()).Times(0)
			handler.networkStore.EXPECT().GetNetworkPath(gomock.Any()).Times(0)
			handler.networkStore.EXPECT().DeleteNetwork(gomock.Any()).Times(0)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testDescribingNetworkThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(false, nil)
	// -- unexpected calls
	handler.networkStore.EXPECT().GetNetwork(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().ListNetworks().Times(0)
	handler.networkStore.EXPECT().SaveNetwork(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().GetNetworkPath(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().DeleteNetwork(gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeNetworkParams{
		Network: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrNetworkDoesNotExist)
}

func testGettingInternalErrorDuringNetworkVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, assert.AnError)

	// -- unexpected calls
	handler.networkStore.EXPECT().ListNetworks().Times(0)
	handler.networkStore.EXPECT().NetworkExists(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().GetNetwork(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().SaveNetwork(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().GetNetworkPath(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().DeleteNetwork(gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeNetworkParams{
		Network: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the network existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringNetworkRetrievalFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(gomock.Any()).Times(1).Return(nil, assert.AnError)
	// -- unexpected calls
	handler.networkStore.EXPECT().ListNetworks().Times(0)
	handler.networkStore.EXPECT().SaveNetwork(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().GetNetworkPath(gomock.Any()).Times(0)
	handler.networkStore.EXPECT().DeleteNetwork(gomock.Any()).Times(0)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeNetworkParams{
		Network: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the network: %w", assert.AnError))
}

type describeNetworkHandler struct {
	*api.AdminDescribeNetwork
	ctrl         *gomock.Controller
	networkStore *mocks.MockNetworkStore
}

func (h *describeNetworkHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminDescribeNetworkResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminDescribeNetworkResult)
		if !ok {
			t.Fatal("AdminDescribeNetwork handler result is not a AdminDescribeNetworkResult")
		}
		return result, err
	}
	return api.AdminDescribeNetworkResult{}, err
}

func newDescribeNetworkHandler(t *testing.T) *describeNetworkHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &describeNetworkHandler{
		AdminDescribeNetwork: api.NewAdminDescribeNetwork(networkStore),
		ctrl:                 ctrl,
		networkStore:         networkStore,
	}
}
