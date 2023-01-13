package api_test

import (
	"context"
	"fmt"
	"testing"

	vgencoding "code.vegaprotocol.io/vega/libs/encoding"
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
	t.Run("Updating a network with invalid params fails", testUpdatingNetworkWithInvalidParamsFails)
	t.Run("Updating a network with valid params succeeds", testUpdatingNetworkWithValidParamsSucceeds)
	t.Run("Updating a network that does not exists fails", testUpdatingNetworkThatDoesNotExistsFails)
	t.Run("Getting internal error during verification fails", testAdminUpdateNetworkGettingInternalErrorDuringNetworkVerificationFails)
	t.Run("Getting internal error during retrieval fails", testAdminUpdateNetworkGettingInternalErrorDuringNetworkSavingFails)
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
			params: api.AdminUpdateNetworkParams{
				Name:        "",
				Level:       "info",
				TokenExpiry: "2m",
			},
			expectedError: api.ErrNetworkNameIsRequired,
		},
		{
			name: "with invalid log level",
			params: api.AdminUpdateNetworkParams{
				Name:        vgrand.RandomStr(3),
				Level:       vgrand.RandomStr(3),
				TokenExpiry: "2m",
			},
			expectedError: api.ErrInvalidLogLevelValue,
		},
		{
			name: "with invalid token expiry",
			params: api.AdminUpdateNetworkParams{
				Name:        vgrand.RandomStr(3),
				Level:       "info",
				TokenExpiry: "100",
			},
			expectedError: api.ErrInvalidTokenExpiryValue,
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
	logLevel := &vgencoding.LogLevel{}
	_ = logLevel.UnmarshalText([]byte("info"))
	tokenExpiry := &vgencoding.Duration{}
	_ = tokenExpiry.UnmarshalText([]byte("2m"))

	// setup
	handler := newUpdateNetworkHandler(t)
	// -- expected calls
	handler.networkStore.EXPECT().NetworkExists(name).Times(1).Return(true, nil)
	handler.networkStore.EXPECT().SaveNetwork(&network.Network{
		Name:        name,
		LogLevel:    *logLevel,
		TokenExpiry: *tokenExpiry,
	}).Times(1).Return(nil)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminUpdateNetworkParams{
		Name:        name,
		Level:       "info",
		TokenExpiry: "2m",
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
	errorDetails := handler.handle(t, ctx, api.AdminUpdateNetworkParams{
		Name:        name,
		Level:       "info",
		TokenExpiry: "2m",
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
	errorDetails := handler.handle(t, ctx, api.AdminUpdateNetworkParams{
		Name:        name,
		Level:       "info",
		TokenExpiry: "2m",
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
	errorDetails := handler.handle(t, ctx, api.AdminUpdateNetworkParams{
		Name:        name,
		Level:       "info",
		TokenExpiry: "2m",
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
