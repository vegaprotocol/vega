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
	t.Run("Describing a network with valid params succeeds", testDescribingNetworkWithValidParamsSucceeds)
	t.Run("Describing a network with empty hosts returns non-nil slice", testDescribeNetworkEmptyHosts)
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
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		},
		{
			name: "with empty network",
			params: api.AdminDescribeNetworkParams{
				Name: "",
			},
			expectedError: api.ErrNetworkNameIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newDescribeNetworkHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testDescribingNetworkWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	network := newNetwork(t)

	// setup
	handler := newDescribeNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeNetworkParams{
		Name: network.Name,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, network.Name, result.Name)
	assert.Equal(t, network.API.GRPC.Hosts, result.API.GRPC.Hosts)
	assert.Equal(t, network.API.GRPC.Retries, result.API.GRPC.Retries)
	assert.Equal(t, network.API.REST.Hosts, result.API.REST.Hosts)
	assert.Equal(t, network.API.GraphQL.Hosts, result.API.GraphQL.Hosts)
}

func testDescribeNetworkEmptyHosts(t *testing.T) {
	// given
	ctx := context.Background()
	network := newNetwork(t)
	network.API.GRPC.Hosts = nil
	network.API.REST.Hosts = nil
	network.API.GraphQL.Hosts = nil

	// setup
	handler := newDescribeNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(network.Name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().GetNetwork(network.Name).Times(1).Return(&network, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeNetworkParams{
		Name: network.Name,
	})

	// then
	require.Nil(t, errorDetails)
	assert.NotNil(t, result.API.GRPC.Hosts)
	assert.NotNil(t, result.API.REST.Hosts)
	assert.NotNil(t, result.API.GraphQL.Hosts)
}

func testDescribingNetworkThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newDescribeNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeNetworkParams{
		Name: name,
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

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeNetworkParams{
		Name: name,
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
	handler.networkStore.EXPECT().GetNetwork(name).Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeNetworkParams{
		Name: name,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the network configuration: %w", assert.AnError))
}

type describeNetworkHandler struct {
	*api.AdminDescribeNetwork
	ctrl         *gomock.Controller
	networkStore *mocks.MockNetworkStore
}

func (h *describeNetworkHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) (api.AdminNetwork, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.AdminNetwork)
		if !ok {
			t.Fatal("AdminDescribeNetwork handler result is not a AdminNetwork")
		}
		return result, err
	}
	return api.AdminNetwork{}, err
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
