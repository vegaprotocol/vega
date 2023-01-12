package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestAdminCloseConnectionsToHostname(t *testing.T) {
	t.Run("Closing a connection with invalid params fails", testAdminCloseConnectionsToHostnameWithInvalidParamsFails)
	t.Run("Closing a connection with valid params succeeds", testAdminCloseConnectionsToHostnameWithValidParamsSucceeds)
}

func testAdminCloseConnectionsToHostnameWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with nil params",
			params:        nil,
			expectedError: api.ErrParamsRequired,
		}, {
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		}, {
			name: "with empty hostname",
			params: api.AdminCloseConnectionsToHostnameParams{
				Hostname: "",
			},
			expectedError: api.ErrHostnameIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newCloseConnectionsToHostnameHandler(tt)

			// when
			errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testAdminCloseConnectionsToHostnameWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	hostname1 := vgrand.RandomStr(5)
	hostname2 := vgrand.RandomStr(5)
	wallet1 := vgrand.RandomStr(5)
	wallet2 := vgrand.RandomStr(5)
	wallet3 := vgrand.RandomStr(5)

	// setup
	handler := newCloseConnectionsToHostnameHandler(t)
	// -- expected calls
	handler.connectionsManager.EXPECT().ListSessionConnections().Times(1).Return([]api.Connection{
		{
			Hostname: hostname1,
			Wallet:   wallet1,
		}, {
			Hostname: hostname1,
			Wallet:   wallet2,
		}, {
			Hostname: hostname1,
			Wallet:   wallet3,
		}, {
			Hostname: hostname2,
			Wallet:   wallet1,
		}, {
			Hostname: hostname2,
			Wallet:   wallet2,
		}, {
			Hostname: hostname2,
			Wallet:   wallet3,
		},
	})
	handler.connectionsManager.EXPECT().EndSessionConnection(hostname1, wallet1).Times(1)
	handler.connectionsManager.EXPECT().EndSessionConnection(hostname1, wallet2).Times(1)
	handler.connectionsManager.EXPECT().EndSessionConnection(hostname1, wallet3).Times(1)

	// when
	errorDetails := handler.handle(t, ctx, api.AdminCloseConnectionsToHostnameParams{
		Hostname: hostname1,
	})

	// then
	require.Nil(t, errorDetails)
}

type adminCloseConnectionsToHostnameHandler struct {
	*api.AdminCloseConnectionsToHostname
	ctrl               *gomock.Controller
	connectionsManager *mocks.MockConnectionsManager
	walletStore        *mocks.MockWalletStore
}

func (h *adminCloseConnectionsToHostnameHandler) handle(t *testing.T, ctx context.Context, params jsonrpc.Params) *jsonrpc.ErrorDetails {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	require.Empty(t, rawResult)
	return err
}

func newCloseConnectionsToHostnameHandler(t *testing.T) *adminCloseConnectionsToHostnameHandler {
	t.Helper()

	ctrl := gomock.NewController(t)

	walletStore := mocks.NewMockWalletStore(ctrl)
	connectionsManager := mocks.NewMockConnectionsManager(ctrl)

	return &adminCloseConnectionsToHostnameHandler{
		AdminCloseConnectionsToHostname: api.NewAdminCloseConnectionsToHostname(connectionsManager),
		ctrl:                            ctrl,
		connectionsManager:              connectionsManager,
		walletStore:                     walletStore,
	}
}
