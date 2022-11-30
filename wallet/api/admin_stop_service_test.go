package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminStopService(t *testing.T) {
	t.Run("Stopping the service with invalid params fails", testStoppingNetworkWithInvalidParamsFails)
	t.Run("Stopping the service with valid params succeeds", testStoppingNetworkWithValidParamsSucceeds)
	t.Run("Stopping the service that does not exists fails", testStoppingNetworkThatDoesNotExistsFails)
}

func testStoppingNetworkWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminStopServiceParams{
				Network: "",
			},
			expectedError: api.ErrNetworkIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newStopServiceHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testStoppingNetworkWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newStopServiceHandler(t)
	if err := handler.servicesManager.RegisterService(name, vgrand.RandomStr(5), session.NewSessions(), dummyServiceShutdownSwitch()); err != nil {
		t.Fatal(err)
	}

	// when
	errorDetails := handler.handle(t, ctx, api.AdminStopServiceParams{
		Network: name,
	})

	// then
	require.Nil(t, errorDetails)

	// when
	sessions, err := handler.servicesManager.Sessions(name)

	// then
	assert.Nil(t, sessions)
	assert.ErrorIs(t, err, api.ErrNoServiceIsRunningForThisNetwork)
}

func testStoppingNetworkThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	name := vgrand.RandomStr(5)

	// setup
	handler := newStopServiceHandler(t)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminStopServiceParams{
		Network: name,
	})

	// then
	require.Nil(t, errorDetails)
}

type stopNetworkHandler struct {
	*api.AdminStopService
	ctrl            *gomock.Controller
	servicesManager *api.ServicesManager
	walletStore     *mocks.MockWalletStore
	tokenStore      *mocks.MockTokenStore
}

func (h *stopNetworkHandler) handle(t *testing.T, ctx context.Context, params interface{}) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	assert.Nil(t, rawResult)
	return err
}

func newStopServiceHandler(t *testing.T) *stopNetworkHandler {
	t.Helper()

	ctrl := gomock.NewController(t)

	walletStore := mocks.NewMockWalletStore(ctrl)
	tokenStore := mocks.NewMockTokenStore(ctrl)
	tokenStore.EXPECT().ListTokens().AnyTimes().Return([]session.TokenSummary{}, nil)
	servicesManager := api.NewServicesManager(tokenStore, walletStore)

	return &stopNetworkHandler{
		AdminStopService: api.NewAdminStopService(servicesManager),
		ctrl:             ctrl,
		servicesManager:  servicesManager,
		walletStore:      walletStore,
		tokenStore:       tokenStore,
	}
}
