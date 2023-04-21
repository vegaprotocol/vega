package api_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/network"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminUpdateNetwork(t *testing.T) {
	t.Run("Documentation matches the code", testAdminUpdateNetworkSchemaCorrect)
	t.Run("Updating a network with invalid params fails", testUpdatingNetworkWithInvalidParamsFails)
	t.Run("Updating a network with valid params succeeds", testUpdatingNetworkWithValidParamsSucceeds)
	t.Run("Updating a network that does not exists fails", testUpdatingNetworkThatDoesNotExistsFails)
	t.Run("Getting internal error during verification fails", testAdminUpdateNetworkGettingInternalErrorDuringNetworkVerificationFails)
	t.Run("Getting internal error during retrieval fails", testAdminUpdateNetworkGettingInternalErrorDuringNetworkSavingFails)
}

func testAdminUpdateNetworkSchemaCorrect(t *testing.T) {
	assertEqualSchema(t, "admin.update_network", network.Network{}, nil)
}

func testUpdatingNetworkWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty network name",
			params: api.AdminNetwork{
				Name: "",
			},
			expectedError: api.ErrNetworkNameIsRequired,
		},
		{
			name: "without a single GRPC node",
			params: api.AdminNetwork{
				Name: "testnet",
				API: api.AdminAPIConfig{
					GRPC: api.AdminGRPCConfig{
						Hosts: []string{},
					},
				},
			},
			expectedError: network.ErrNetworkDoesNotHaveGRPCHostConfigured,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newUpdateNetworkHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testUpdatingNetworkWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdateNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().SaveNetwork(&network.Network{
		Name: name,
		API: network.APIConfig{
			GRPC: network.GRPCConfig{
				Hosts: []string{
					"localhost:1234",
				},
			},
		},
	}).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminNetwork{
		Name: name,
		API: api.AdminAPIConfig{
			GRPC: api.AdminGRPCConfig{
				Hosts: []string{
					"localhost:1234",
				},
			},
		},
	})

	// then
	require.Nil(t, errorDetails)
}

func testUpdatingNetworkThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdateNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(false, nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminNetwork{
		Name: name,
		API: api.AdminAPIConfig{
			GRPC: api.AdminGRPCConfig{
				Hosts: []string{
					"localhost:1234",
				},
			},
		},
	})

	// then
	require.NotNil(t, errorDetails)
	assertInvalidParams(t, errorDetails, api.ErrNetworkDoesNotExist)
}

func testAdminUpdateNetworkGettingInternalErrorDuringNetworkVerificationFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdateNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(false, assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminNetwork{
		Name: name,
		API: api.AdminAPIConfig{
			GRPC: api.AdminGRPCConfig{
				Hosts: []string{
					"localhost:1234",
				},
			},
		},
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the network existence: %w", assert.AnError))
}

func testAdminUpdateNetworkGettingInternalErrorDuringNetworkSavingFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newUpdateNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().SaveNetwork(gomock.Any()).Times(1).Return(assert.AnError)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminNetwork{
		Name: name,
		API: api.AdminAPIConfig{
			GRPC: api.AdminGRPCConfig{
				Hosts: []string{
					"localhost:1234",
				},
			},
		},
	})

	// then
	require.NotNil(t, errorDetails)
	assertInternalError(t, errorDetails, fmt.Errorf("could not save the network: %w", assert.AnError))
}

type updateNetworkHandler struct {
	*api.AdminUpdateNetwork
	ctrl         *gomock.Controller
	networkStore *mocks.MockNetworkStore
}

func (h *updateNetworkHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	assert.Nil(t, rawResult)
	return err
}

func newUpdateNetworkHandler(t *testing.T) *updateNetworkHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	networkStore := mocks.NewMockNetworkStore(ctrl)

	return &updateNetworkHandler{
		AdminUpdateNetwork: api.NewAdminUpdateNetwork(networkStore),
		ctrl:               ctrl,
		networkStore:       networkStore,
	}
}
